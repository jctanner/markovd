---
title: Kubernetes runner does not set ImagePullPolicy on job pods
severity: high
component: runner
---

## Problem

The Kubernetes runner (`internal/runner/k8s.go:116`) creates job containers without setting `ImagePullPolicy`. When the image tag is `:latest`, Kubernetes defaults to `Always`, causing the kubelet to attempt a pull from a remote registry. In environments where the image is locally imported (e.g., `docker save | k3s ctr images import`), this results in `ErrImagePull` and the job never starts.

## Evidence

```
$ kubectl get pods -n ai-pipeline -l app=markov
NAME                        READY   STATUS         RESTARTS   AGE
markov-run-3aa9930c-ngqmt   0/1     ErrImagePull   0          36s
```

## Fix

Add `ImagePullPolicy: corev1.PullNever` (or make it configurable via env var) to the container spec in `internal/runner/k8s.go`:

```go
Containers: []corev1.Container{
    {
        Name:            "markov",
        Image:           r.image,
        ImagePullPolicy: corev1.PullNever,
        Command:         []string{"markov"},
        Args:            args,
        EnvFrom:         envFrom,
        ...
    },
},
```

A configurable approach would be preferable for production use where images may come from a registry:

```go
// In KubernetesRunner struct
imagePullPolicy corev1.PullPolicy

// Env var: MARKOVD_JOB_IMAGE_PULL_POLICY (default: "Never")
```

## Resolution

Added `imagePullPolicy` as a configurable field on `KubernetesRunner`. The policy is set on the markov container in every Job the runner creates.

**Env var**: `MARKOVD_JOB_IMAGE_PULL_POLICY` — accepts `Never`, `IfNotPresent`, or `Always`. Defaults to `IfNotPresent` when unset.

**Files changed**:
- `internal/runner/k8s.go` — added `imagePullPolicy` field to struct, new parameter on `NewKubernetesRunner()`, applied to container spec at line 125
- `cmd/markovd/main.go` — reads `MARKOVD_JOB_IMAGE_PULL_POLICY` env var and passes it to the constructor
- `internal/runner/k8s_test.go` — added `TestImagePullPolicy` covering both `IfNotPresent` default and `Never` override

**For locally-imported images** (e.g., `k3s ctr images import`), set:
```
MARKOVD_JOB_IMAGE_PULL_POLICY=Never
```
