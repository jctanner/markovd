# Task: Define Workflow Definition Model

## Goal

Replace the single `yaml` workflow assumption with a model that can represent
either one workflow file or a directory workflow file set.

## Context

Current persistence and API models store `workflows.yaml` as the only canonical
definition. Markov now accepts a directory input path as well as a file path.

## Acceptance Criteria

- [ ] Add `WorkflowDefinition` and `WorkflowDefinitionFile` types.
- [ ] Add database columns for `definition_kind`, `definition_json`,
      `source_kind`, and `source_root`.
- [ ] Backfill existing rows as file definitions.
- [ ] Keep legacy `yaml` response compatibility for file workflows.
- [ ] Define source semantics for manual file, manual directory, project file,
      and project directory workflows.
- [ ] Update DB helpers in `internal/db/workflows.go` and project import helpers.
- [ ] Add tests for migration/backfill behavior where practical.

## Files Likely Involved

- `internal/models/models.go`
- `internal/db/db.go`
- `internal/db/workflows.go`
- `internal/db/projects.go`
- `internal/api/workflows.go`
- `ui/src/api.ts`

## Status

Pending

## Notes

Plan reference: `docs/plans/001-workflow-input-formats.md`.
