---
title: markovd uses shell runner instead of kubernetes runner in k8s deployment
severity: high
component: deployment
---

## Problem

markovd is deployed to Kubernetes but runs markov workflows as local subprocesses inside the markovd container (shell runner) instead of spawning them as Kubernetes Jobs (kubernetes runner).

The Go code already supports both runner backends, selected by the `MARKOVD_RUNNER` environment variable (default: `shell`). The Kubernetes runner (`internal/runner/k8s.go`) creates a Job with a ConfigMap-mounted workflow, proper ServiceAccount, and callback configuration. However, the k8s deployment manifest (`deploy/k8s/15-markovd.yaml`) does not set the required environment variables to activate it.

## Evidence

Pod logs show `[markov]` output inline with markovd logs, confirming subprocess execution. No Jobs are created in the namespace (`kubectl get jobs -n ai-pipeline` returns empty).

## Required Environment Variables

From `cmd/markovd/main.go`:

| Variable | Required Value | Description |
|----------|---------------|-------------|
| `MARKOVD_RUNNER` | `kubernetes` | Switch from shell to k8s job runner |
| `MARKOVD_MARKOV_IMAGE` | `markov:latest` | Container image for job pods |
| `MARKOVD_JOB_NAMESPACE` | (auto-detected from pod) | Namespace to create jobs in |
| `MARKOVD_JOB_SERVICE_ACCOUNT` | `pipeline-agent` (default) | ServiceAccount for job pods |
| `MARKOVD_JOB_SECRETS` | comma-separated secret names | Secrets to inject into job pods via `envFrom` |

## Fix

Add to the markovd container env in `deploy/k8s/15-markovd.yaml`:

```yaml
- name: MARKOVD_RUNNER
  value: "kubernetes"
- name: MARKOVD_MARKOV_IMAGE
  value: "markov:latest"
```

`MARKOVD_JOB_NAMESPACE` will auto-detect from the mounted ServiceAccount. `MARKOVD_JOB_SERVICE_ACCOUNT` defaults to `pipeline-agent`, which already exists and is bound to the `pipeline-orchestrator` Role.
