# Task: Support Diagrams for Directory Workflow Definitions

## Goal

Generate workflow structure diagrams for both single-file and directory workflow
definitions.

## Context

`internal/api/diagram.go` currently unmarshals one single-file YAML document
with a top-level `workflows` list.

## Acceptance Criteria

- [ ] Add diagram generation from `WorkflowDefinition`.
- [ ] Preserve existing diagrams for single-file workflows.
- [ ] Support directory definitions by resolving/merging the directory files into
      the diagram schema or by delegating to Markov.
- [ ] Return useful errors for invalid definitions instead of a generic internal
      error.
- [ ] Add tests for file and directory diagram generation.
- [ ] Confirm diagram generation uses the same path validation and
      materialization helper as validation and runners.

## Files Likely Involved

- `internal/api/diagram.go`
- `internal/api/workflows.go`
- New workflow definition materializer/loader helper

## Status

Pending

## Notes

Long-term preference is to use Markov's resolved workflow model rather than
duplicating parser behavior in markovd.
