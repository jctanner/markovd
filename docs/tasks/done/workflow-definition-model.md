# Task: Define Workflow Definition Model

## Goal

Replace the single `yaml` workflow assumption with a model that can represent
either one workflow file or a directory workflow file set.

## Context

Current persistence and API models store `workflows.yaml` as the only canonical
definition. Markov now accepts a directory input path as well as a file path.

## Acceptance Criteria

- [x] Add `WorkflowDefinition` and `WorkflowDefinitionFile` types.
- [x] Add database columns for `definition_kind`, `definition_json`,
      `source_kind`, and `source_root`.
- [x] Backfill existing rows as file definitions.
- [x] Keep legacy `yaml` response compatibility for file workflows.
- [x] Define source semantics for manual file, manual directory, project file,
      and project directory workflows.
- [x] Update DB helpers in `internal/db/workflows.go` and project import helpers.
- [x] Add tests for migration/backfill behavior where practical.

## Files Likely Involved

- `internal/models/models.go`
- `internal/db/db.go`
- `internal/db/workflows.go`
- `internal/db/projects.go`
- `internal/api/workflows.go`
- `ui/src/api.ts`

## Status

Done

## Notes

Plan reference: `docs/plans/001-workflow-input-formats.md`.

Implementation started:
- Added workflow definition model fields to `internal/models/models.go`.
- Added workflow definition storage/backfill migrations.
- Updated workflow and project DB helpers to read/write `definition_kind`,
  `definition_json`, `source_kind`, and `source_root`.
- Added shared definition normalization/materialization helpers in
  `internal/workflowdef`.
- Preserved legacy `yaml` compatibility for file workflows.

Verification so far:
- `go test ./...`
- `npm run build`
