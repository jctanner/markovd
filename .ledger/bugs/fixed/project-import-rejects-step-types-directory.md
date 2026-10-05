# Bug: Project import rejects directory workflows with `step_types/`

## Summary

Importing `var/demos/end-to-end` from the `ai-first-pipeline` project fails.
The API returns `missing required directory workflow file: step_types.yaml`.

Markov's directory workflow format allows step types to be defined either in a
single `step_types.yaml` file or as YAML files under a `step_types/` directory.
Markovd's workflow definition normalization only accepts `step_types.yaml`.

## Impact

Valid directory workflows that use `step_types/` cannot be imported into
Markovd. The frontend also hides the detailed per-path import error and only
reports the path.

## Evidence

- Running instance: `https://markovd.local`
- API import request for project `1`, path `var/demos/end-to-end`, kind
  `directory`
- API response:
  `missing required directory workflow file: step_types.yaml`
- After allowing `step_types/` in Markovd normalization, the running Markov
  validator still failed with:
  `reading "/tmp/markov-workflow-.../step_types.yaml": open .../step_types.yaml: no such file or directory`

## Status

Fixed

## Resolution

Workflow definition normalization now accepts directory workflows that provide
step types via `step_types/*.yaml` when `step_types.yaml` is absent. This matches
Markov's documented directory workflow format.

For compatibility with Markov binaries that still require `step_types.yaml`,
Markovd now builds a runtime-compatible directory definition when validating or
running workflows: it merges `step_types/*.yaml` into a generated
`step_types.yaml` and omits the `step_types/` files from the runtime copy to
avoid duplicate step type loads in newer Markov binaries.

The Projects page import error now includes the backend's per-path error detail
instead of only listing failed paths.

## Verification

- Reproduced against `https://markovd.local` with `POST
  /api/v1/projects/1/import` for `var/demos/end-to-end`; API returned
  `missing required directory workflow file: step_types.yaml`.
- Added workflow definition normalization coverage for `step_types/*.yaml`.
- Added runtime compatibility coverage for generated `step_types.yaml`.
- Added project import read coverage for a directory workflow using
  `step_types/`.
- Added Kubernetes runner coverage proving generated runtime ConfigMaps contain
  `step_types.yaml` and omit `step_types/*.yaml`.
- `env GOCACHE=/tmp/go-build-cache go test ./internal/workflowdef ./internal/projects`
  passed.
- `env GOCACHE=/tmp/go-build-cache go test ./internal/workflowdef ./internal/projects ./internal/runner`
  passed.
- `env GOCACHE=/tmp/go-build-cache go test ./...` passed.
- `npm run build` passed in `ui/`.
- `npx eslint src/pages/Projects.tsx` was run; it still fails on pre-existing
  React hook lint errors in `Projects.tsx`.
