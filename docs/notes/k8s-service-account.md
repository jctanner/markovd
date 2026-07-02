# Kubernetes Service Account

markovd needs a Kubernetes ServiceAccount with RBAC permissions to orchestrate workloads — creating Jobs, reading Pod logs, managing Secrets, and provisioning PersistentVolumeClaims.

## Current Permissions

The ServiceAccount `markovd` is bound to the `pipeline-orchestrator` Role, which grants:

| API Group | Resource | Verbs |
|-----------|----------|-------|
| `batch` | `jobs` | get, list, watch, create, delete |
| `""` (core) | `pods` | get, list, watch |
| `""` (core) | `pods/log` | get |

This is sufficient for submitting jobs and tailing their logs, but not for managing volumes or secrets.

## Required Permissions

To fully manage workflow execution, markovd needs these additional RBAC rules:

### Secrets (for injecting credentials into Jobs)

```yaml
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "create", "update", "delete"]
```

Use cases:
- Create per-job Secrets with API tokens, callback credentials
- Mount secrets into Job pods as env vars or volumes
- Clean up secrets when jobs complete

### PersistentVolumeClaims (for job workspace storage)

```yaml
- apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "list", "create", "delete"]
```

Use cases:
- Provision scratch volumes for job working directories
- Mount shared PVCs (issues, workspace, artifacts) into jobs
- Clean up ephemeral PVCs after job completion

### ConfigMaps (for job configuration)

```yaml
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "create", "update", "delete"]
```

Use cases:
- Inject workflow definitions and skill configs into jobs
- Store job templates as ConfigMaps

## Full Role Definition

The complete Role covering all orchestration needs:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pipeline-orchestrator
  namespace: ai-pipeline
rules:
# Job management
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch", "create", "delete"]
# Pod access (for logs and status)
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get"]
# Secret management (for job credentials)
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "create", "update", "delete"]
# PVC management (for job storage)
- apiGroups: [""]
  resources: ["persistentvolumeclaims"]
  verbs: ["get", "list", "create", "delete"]
# ConfigMap management (for job config)
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "create", "update", "delete"]
```

## Deployment Setup

The ServiceAccount is defined in `15-markovd.yaml` and bound to the Role in `16-pipeline-rbac.yaml`:

```yaml
# 15-markovd.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: markovd
  namespace: ai-pipeline
```

```yaml
# 16-pipeline-rbac.yaml (RoleBinding subjects)
subjects:
- kind: ServiceAccount
  name: pipeline-dashboard
  namespace: ai-pipeline
- kind: ServiceAccount
  name: markovd
  namespace: ai-pipeline
```

The deployment pod spec references it:

```yaml
spec:
  serviceAccountName: markovd
```

## Job ServiceAccount (`pipeline-agent`)

Jobs spawned by markovd need their own ServiceAccount so they can interact with the Kubernetes API — particularly when a job needs to spawn sub-jobs (e.g., a workflow step that fans out work into child jobs).

The `pipeline-agent` ServiceAccount is defined in `16-pipeline-rbac.yaml` and bound to the same `pipeline-orchestrator` Role. markovd sets `serviceAccountName: pipeline-agent` on every Job it creates. Child jobs inherit the same SA automatically.

```
markovd (SA: markovd)
  └─ creates Job A (SA: pipeline-agent)
       └─ creates Job B (SA: pipeline-agent)
            └─ ...
```

All three ServiceAccounts (`pipeline-dashboard`, `markovd`, `pipeline-agent`) share the same Role, so they have identical permissions. The separation exists for auditability — `kubectl` audit logs show which SA performed each action.

### Why not reuse the `markovd` ServiceAccount?

You could, but separating them lets you:
- Tighten permissions independently (e.g., jobs don't need `delete` on secrets)
- Trace API calls back to their origin (markovd vs. job pod) in audit logs
- Revoke job permissions without affecting the markovd control plane

## Accessing the API from Go

The Kubernetes client auto-discovers credentials from the mounted ServiceAccount token:

```go
import "k8s.io/client-go/rest"

config, err := rest.InClusterConfig()
```

This reads the token from `/var/run/secrets/kubernetes.io/serviceaccount/token`, which is automatically mounted by Kubernetes when `serviceAccountName` is set.

## Notes

- The Role is namespace-scoped (`ai-pipeline`), so markovd cannot access resources in other namespaces.
- The `pipeline-dashboard` ServiceAccount shares the same Role — permission changes affect both services.
- If markovd and the dashboard need divergent permissions in the future, create a separate `markovd-orchestrator` Role.
