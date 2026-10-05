---
title: Kubernetes directory workflows mount files at the wrong path
severity: high
component: runner/kubernetes
---

## Problem

Directory workflow runs launched through the Kubernetes runner failed before
workflow execution because Markov could not find the directory path it was asked
to run.

Observed run:

- `markov-run-2a0c4953`
- URL: `http://markovd.local/runs/markov-run-2a0c4953`

Error:

```text
Error: stat workflow path: stat /etc/markov/workflow: no such file or directory
Usage:
  markov run <file.yaml|directory> [flags]
```

## Root Cause

The Kubernetes runner passed `/etc/markov/workflow` for directory workflow
runs, but mounted the workflow ConfigMap at `/etc/markov`.

That produced files like:

```text
/etc/markov/meta.yaml
/etc/markov/vars.yaml
/etc/markov/rules.yaml
/etc/markov/step_types.yaml
/etc/markov/workflows/main.yaml
```

Markov was invoked with:

```text
markov run /etc/markov/workflow
```

So the path did not exist in the container.

## Fix

Directory workflow ConfigMaps now mount at `/etc/markov/workflow`, matching the
path passed to Markov. Single-file workflows continue to mount at `/etc/markov`
and run `/etc/markov/workflow.yaml`.

Added a Kubernetes runner unit test assertion that directory workflow jobs mount
the workflow volume at `/etc/markov/workflow`.

## Verification

```bash
GOCACHE=/tmp/markovd-go-cache go test ./internal/runner
GOCACHE=/tmp/markovd-go-cache go test ./...
```

Fixed in commit:

```text
29d09f5 Fix Kubernetes directory workflow mount path
```

Kubernetes verification run succeeded:

- `markov-run-6bda3960`
- `http://markovd.local/runs/markov-run-6bda3960`
