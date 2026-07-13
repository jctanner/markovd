# Task: Classify Standalone Project Workflow YAML

## Goal

Filter project import candidates so standalone `.yaml` and `.yml` files are
listed only when their structure identifies them as likely Markov workflows.

## Context

Directory workflow roots are already detected and their descendants excluded.
Every other YAML file is currently returned to the Projects page, including
Kubernetes, CI, and application configuration.

The classification contract and implementation phases are defined in:

- [ADR-0004](../../decisions/ADR-0004-classify-standalone-markov-workflows.md)
- [Plan 004](../../plans/004-standalone-workflow-classification.md)

## Acceptance Criteria

- [x] Add a pure structural classifier with stable result reasons.
- [x] Require one mapping document, a scalar entrypoint, a non-empty workflows
      sequence, a named workflow with a steps sequence, and an entrypoint match.
- [x] Reject malformed, multi-document, ambiguous duplicate-key, and
      structurally non-matching YAML without failing project listing.
- [x] Keep optional Markov sections optional and allow unknown top-level keys.
- [x] Filter standalone discovery without invoking the Markov binary.
- [x] Preserve directory workflow detection and descendant exclusion.
- [x] Preserve deterministic listing order and useful filesystem errors.
- [x] Preserve authoritative Markov validation during direct and UI imports.
- [x] Add classifier table tests and mixed-repository discovery tests for both
      `.yaml` and `.yml`.
- [x] Verify the Projects page renders only the server-returned workflow
      candidates for a representative mixed repository.

## Files Likely Involved

- `internal/projects/git.go`
- `internal/projects/git_test.go`
- `internal/api/projects.go`
- `internal/api/projects_test.go`, if endpoint coverage is needed
- `ui/src/pages/Projects.tsx`, only if the API contract changes

## Status

Done

## Notes

The existing project-file response shape should remain unchanged. Classification
is a discovery filter, not a replacement for import-time Markov validation.

Implemented:

- Added a package-private, pure `yaml.Node` classifier with stable reason codes.
- Filtered non-directory YAML candidates during project discovery and sorted
  accepted definitions by path.
- Changed the project-files endpoint to return genuine discovery errors instead
  of treating them as an empty successful result.
- Preserved the existing project import call to
  `workflowdef.ValidateWithMarkov`.

Verification performed on 2026-07-12:

- `env GOCACHE=/tmp/go-build-cache go test ./internal/projects ./internal/api`
- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `env GOCACHE=/tmp/go-build-cache go vet ./internal/projects ./internal/api`
- `cd ui && npm run build`
- `git diff --check`
- Classifier table tests cover minimal and optional valid workflows, empty and
  malformed input, multiple documents, wrong node kinds, missing and duplicate
  signature keys, invalid workflow items, entrypoint mismatch, Kubernetes, and
  CI YAML.
- Mixed-repository discovery covers `.yaml`, `.yml`, filtered invalid files,
  stable path ordering, directory-root detection, and descendant exclusion.
- Playwright rendered a representative project response with exactly two
  classified files and one directory workflow, preserving imported state,
  badges, selection controls, and a layout without horizontal overflow.
