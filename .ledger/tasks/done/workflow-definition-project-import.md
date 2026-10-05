# Task: Import Directory Workflows From Projects

## Goal

Let project sync/import discover and import both standalone workflow files and
directory workflow roots.

## Context

Project import currently lists YAML files and reads one file per workflow. A
directory workflow should be imported by its root directory and all required
files inside that root.

## Acceptance Criteria

- [x] Replace `ListYAMLFiles` with workflow definition discovery.
- [x] Detect directory workflows by Markov's conventional required files.
- [x] Hide or disable internal files of a detected directory workflow so users do
      not import partial definitions by accident.
- [x] Import directory definitions by reading all required and workflow files.
- [x] Re-sync project-sourced directory workflows when the source repository is
      synced.
- [x] Preserve existing standalone YAML file import behavior.
- [x] Keep the legacy `{ "files": [...] }` import payload working for file-only
      imports.
- [x] Add a new `{ "definitions": [{ "path": "...", "kind": "..." }] }` import
      payload for file and directory imports.
- [x] Add tests for project discovery and safe path handling.

## Files Likely Involved

- `internal/projects/git.go`
- `internal/api/projects.go`
- `internal/db/projects.go`
- `internal/models/models.go`
- `ui/src/pages/Projects.tsx`
- `ui/src/api.ts`

## Status

Done

## Notes

The API should return `kind`, `path`, `name`, and `imported` for each discovered
definition.
