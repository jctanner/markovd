# Bug: Project import treats `meta.yaml` directory roots as files

## Summary

Project import should detect a directory containing `meta.yaml` as a
directory-based workflow root. Instead, a path like `var/demos/end-to-end` is
shown as individual YAML subfiles rather than one `directory` workflow entry.

## Impact

Users can accidentally import component YAML files one by one instead of the
intended directory workflow definition.

## Evidence

- Reported against running instance `https://markovd.local`.
- Project: `https://github.com/jctanner/ai-first-pipeline`.
- Expected project path: `var/demos/end-to-end`.
- Expected behavior: `var/demos/end-to-end` appears once as a `directory`
  workflow and its internal YAML files are hidden from the import list.

## Status

Fixed

## Resolution

Project workflow discovery now treats any directory containing a `meta.yaml`
file as a directory workflow root. Once detected, the discovery walk skips that
directory's children so internal YAML files do not appear as separate file
workflows in the import list.

## Verification

- Added regression coverage for `var/demos/end-to-end`-style directory roots
  with `meta.yaml` and nested YAML files.
- `env GOCACHE=/tmp/go-build-cache go test ./internal/projects` passed.
- `env GOCACHE=/tmp/go-build-cache go test ./...` passed.
