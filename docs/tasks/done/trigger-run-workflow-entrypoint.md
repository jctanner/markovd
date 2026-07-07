# Task: Add Trigger Run Workflow Entrypoint Override

## Goal

Allow Trigger Run users to optionally pass Markov's `--workflow` override for
file and directory workflow definitions. When the override is blank, omit the
flag and let Markov use the workflow definition entrypoint.

## Context

The Markov CLI reference documents `markov run <file.yaml|directory>
--workflow name` for running a specific workflow by its `name:` field instead
of the file or directory default entrypoint.

## Acceptance Criteria

- [x] Trigger Run has an optional workflow entrypoint override field.
- [x] Blank override values are omitted from the API payload and runner CLI
      invocation.
- [x] Non-blank override values pass through to shell and Kubernetes runners as
      `--workflow <name>`.
- [x] Verification is recorded before moving this task to done.

## Files Likely Involved

- `internal/api/runs.go`
- `internal/runner/runner.go`
- `internal/runner/shell.go`
- `internal/runner/k8s.go`
- `ui/src/api.ts`
- `ui/src/pages/TriggerRun.tsx`

## Verification

- Read Markov reference docs:
  - `../markov/docs/reference/cli.md`
  - `../markov/docs/reference/workflow-file.md`
- `env GOCACHE=/tmp/go-build-cache go test ./internal/runner` passed.
- `env GOCACHE=/tmp/go-build-cache go test ./...` passed.
- `npm run build` passed in `ui/`.
- `npx eslint src/pages/TriggerRun.tsx` passed in `ui/`.

## Status

Done
