# Task: Materialize and Validate Workflow Definitions

## Goal

Validate both file and directory workflow definitions before storing or running
them.

## Context

markovd currently accepts any non-empty YAML string. Directory support needs a
path-safe file set and should use Markov validation so markovd does not drift
from the engine's schema.

## Acceptance Criteria

- [x] Reject absolute paths, empty paths, and paths that escape the workflow root.
- [x] Validate file definitions contain exactly one YAML file.
- [x] Validate directory definitions contain Markov's required category files and
      at least one `workflows/*.yaml` file.
- [x] Materialize definitions to a temp file or directory.
- [x] Add `POST /api/v1/workflows/validate` for UI preflight validation.
- [x] Run `markov validate <path>` before create, update, import, and run.
- [x] Return validation errors to API clients with actionable messages.
- [x] Add unit tests for valid definitions, missing required files, duplicate or
      unsafe paths, and malformed YAML.
- [x] For compose verification, copy `../markov/bin/markov` to `./bin/markov`
      before starting the stack so server-side validation uses the new Markov
      binary.

## Files Likely Involved

- `internal/api/workflows.go`
- `internal/api/projects.go`
- `internal/runner/runner.go`
- New helper under `internal/workflows/` or `internal/definition/`

## Status

Done

## Notes

This should centralize materialization so runners, validation, and diagrams use
the same path rules.
