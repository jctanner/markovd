# Task: Run Directory Workflows From Shell and Kubernetes Runners

## Goal

Allow both runner backends to execute workflow definitions represented as either
a single file or a directory file set.

## Context

`ShellRunner` writes one temp YAML file. `KubernetesRunner` creates one ConfigMap
key named `workflow.yaml` and runs `/etc/markov/workflow.yaml`.

## Acceptance Criteria

- [ ] Change `runner.RunRequest` to carry a workflow definition instead of only
      `WorkflowYAML`.
- [ ] Shell runner writes a temp file for file workflows and a temp directory for
      directory workflows.
- [ ] Kubernetes runner mounts a file workflow at the existing path.
- [ ] Kubernetes runner mounts a directory workflow at `/etc/markov/workflow` and
      runs `markov run /etc/markov/workflow`.
- [ ] ConfigMap key/path handling preserves nested relative paths safely.
- [ ] Existing runner tests continue to cover single-file behavior.
- [ ] Add runner tests for directory ConfigMap materialization and command args.
- [ ] Before compose-based runner testing, copy `../markov/bin/markov` to
      `./bin/markov`, rebuild/start the stack, and record the command and run
      IDs used for single-file and directory workflow verification.

## Files Likely Involved

- `internal/runner/runner.go`
- `internal/runner/shell.go`
- `internal/runner/k8s.go`
- `internal/runner/k8s_test.go`
- `internal/api/runs.go`

## Status

Pending

## Notes

Kubernetes ConfigMap `items` should be used to map stored keys to desired
relative paths.
