# Task: Update Workflow API and UI for File and Directory Definitions

## Goal

Expose workflow definition kind and files through the API and make the UI usable
for both single-file and directory workflows.

## Context

The current UI has a YAML textarea upload modal and a single YAML viewer/editor
on the workflow detail page.

## Acceptance Criteria

- [x] Extend API request/response types with `definition_kind` and `files`.
- [x] Keep legacy `{ name, yaml }` create/update payloads working.
- [x] Add API client support for `POST /api/v1/workflows/validate`.
- [x] Add workflow type badges to the Workflows list.
- [x] Add a manual upload mode switch for single file vs directory.
- [x] Add a directory file editor/viewer with relative paths.
- [x] Show server validation errors in upload/edit forms.
- [x] Update Project import UI labels from "Workflow Files" to "Workflow Definitions".
- [x] Show workflow type/source on Trigger Run and Rerun surfaces.
- [x] End-to-end UI verification runs against a compose stack started after
      copying `../markov/bin/markov` to `./bin/markov`.

## Files Likely Involved

- `internal/api/workflows.go`
- `ui/src/api.ts`
- `ui/src/pages/Workflows.tsx`
- `ui/src/pages/WorkflowDetail.tsx`
- `ui/src/pages/Projects.tsx`
- `ui/src/pages/TriggerRun.tsx`
- `ui/src/components/RerunModal.tsx`
- `ui/src/index.css`

## Status

Done

## Notes

Avoid making directory upload depend only on browser directory picker support;
provide manual file add/edit as the fallback.
