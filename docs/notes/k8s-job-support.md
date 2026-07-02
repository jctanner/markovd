# Kubernetes Job Support Requirements

markovd needs a `KubernetesRunner` that spawns `markov run` as a Kubernetes Job instead of a local subprocess. The markov CLI already has full k8s_job executor support and client-go wired up — this work is about making markovd orchestrate markov as a Job, and ensuring the jobs it spawns in turn have the right credentials.

## Context

Today markovd runs markov via `ShellRunner` (`internal/runner/shell.go`), which calls `exec.CommandContext("markov", "run", ...)`. In Kubernetes, markovd should instead create a `batch/v1.Job` that runs `markov run` inside a pod. That pod then uses its ServiceAccount to create child Jobs via markov's existing `k8s_job` executor.

The chain looks like:

```
markovd (SA: markovd)
  └─ creates K8s Job running "markov run workflow.yaml" (SA: pipeline-agent)
       └─ markov's k8s_job executor creates child Jobs (SA: pipeline-agent)
            └─ child jobs inherit SA from pod spec
```

## What markovd Needs (KubernetesRunner)

A new `internal/runner/k8s.go` implementing the existing `Runner` interface:

```go
type Runner interface {
    Start(ctx context.Context, req RunRequest) (runID string, err error)
    Cancel(runID string) error
}
```

### Runner Selection

Add `MARKOVD_RUNNER` env var in `cmd/markovd/main.go`:

| Value | Behavior |
|-------|----------|
| `shell` (default) | Current ShellRunner — subprocess |
| `kubernetes` | New KubernetesRunner — K8s Job |

### Start()

1. Create a ConfigMap containing the workflow YAML from `req.WorkflowYAML`
2. Create a `batch/v1.Job` with:
   - **Image**: configurable via `MARKOVD_MARKOV_IMAGE` (e.g., `ghcr.io/jctanner/markov:latest`)
   - **Command**: `["markov", "run", "/etc/markov/workflow.yaml"]`
   - **ServiceAccountName**: `pipeline-agent` (configurable via `MARKOVD_JOB_SERVICE_ACCOUNT`)
   - **Namespace**: configurable via `MARKOVD_JOB_NAMESPACE` (default: same namespace as markovd)
   - **Volumes**:
     - ConfigMap with workflow YAML mounted at `/etc/markov/workflow.yaml`
     - Any shared PVCs (issues, workspace, artifacts) if configured
   - **Env vars**:
     - `req.Vars` as individual env vars (prefixed, e.g., `MARKOV_VAR_key=value`), or passed as `--var key=value` args
     - Callback URL/token: `--callback http://markovd:8080/api/v1/events --callback-header Authorization=Bearer<token>`
   - **Secrets**: mount or inject any secrets the workflow jobs will need (configurable via `MARKOVD_JOB_SECRETS`, comma-separated Secret names)
   - **Labels**: `app=markov`, `markov/run-id=<id>`, `markov/workflow=<name>`
3. Return the Job name as `runID`

### Cancel()

Delete the Job with `propagationPolicy: Background` to cascade-delete the pod.

### Cleanup

Set `ttlSecondsAfterFinished` on Jobs (default 86400) so completed jobs are garbage-collected. Also clean up the workflow ConfigMap on job completion.

## Secrets Propagation

When markov runs as a Job and creates child Jobs via `k8s_job` executor, those child Jobs need credentials too. Two patterns are supported:

### 1. Inherited Secrets (env-based)

markovd injects Secret references into the parent Job via `envFrom`. The workflow YAML templates these into child job steps:

```yaml
step_types:
  my_agent:
    base: k8s_job
    job:
      image: my-agent:latest
      secrets:
        - pipeline-credentials   # Secret name, injected as envFrom
```

This already works in markov — `buildEnvFrom()` in `k8s_job.go` handles the `secrets` param.

### 2. Mounted Secrets (volume-based)

For secrets that need to be files (TLS certs, kubeconfigs), the workflow declares volumes:

```yaml
step_types:
  my_agent:
    base: k8s_job
    job:
      image: my-agent:latest
      volumes:
        - name: tls-certs
          secret: my-tls-secret
          mount: /etc/tls
          read_only: true
```

This also already works in markov — `buildVolumes()` handles Secret volume sources.

### What's Needed

markovd's KubernetesRunner needs a way to specify which Secrets should be available to the parent markov Job. Those secrets then propagate to child jobs via the workflow YAML (using the mechanisms above).

Config via `MARKOVD_JOB_SECRETS`:

```
MARKOVD_JOB_SECRETS=pipeline-credentials,jira-token,gcp-service-account
```

These get added as `envFrom` entries on the markov Job pod.

## markovd Dependencies to Add

markovd's `go.mod` needs:

```
k8s.io/api
k8s.io/apimachinery
k8s.io/client-go
```

Use `rest.InClusterConfig()` for auth when `MARKOVD_RUNNER=kubernetes`.

## Files to Create/Modify

| File | Change |
|------|--------|
| `internal/runner/k8s.go` | New — KubernetesRunner implementation |
| `cmd/markovd/main.go` | Read `MARKOVD_RUNNER` env var, instantiate correct runner |
| `go.mod` | Add k8s.io/client-go, k8s.io/api, k8s.io/apimachinery |

## Environment Variables Summary

| Variable | Default | Description |
|----------|---------|-------------|
| `MARKOVD_RUNNER` | `shell` | Runner backend: `shell` or `kubernetes` |
| `MARKOVD_MARKOV_IMAGE` | (required when runner=kubernetes) | Container image for markov CLI |
| `MARKOVD_JOB_NAMESPACE` | (current namespace) | Namespace for spawned Jobs |
| `MARKOVD_JOB_SERVICE_ACCOUNT` | `pipeline-agent` | ServiceAccount for spawned Jobs |
| `MARKOVD_JOB_SECRETS` | (none) | Comma-separated Secret names to inject into Jobs |

## Verification

1. Set `MARKOVD_RUNNER=kubernetes` and trigger a workflow run via the API
2. Confirm a Job appears in the target namespace with correct labels and SA
3. Confirm the markov pod starts, mounts the workflow ConfigMap, and runs the workflow
4. Confirm callback events flow back to markovd and appear in the dashboard
5. Confirm child Jobs created by k8s_job steps use `pipeline-agent` SA
6. Confirm `Cancel()` deletes the Job and its pod
7. Confirm the ConfigMap is cleaned up after job completion
