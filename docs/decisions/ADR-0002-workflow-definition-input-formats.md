# ADR-0002: Support File and Directory Workflow Definitions

## Status

Proposed

## Context

Markov now supports two workflow input forms:

- A single self-contained YAML workflow file.
- A workflow directory containing `meta.yaml`, `vars.yaml`, `rules.yaml`,
  `step_types.yaml`, and `workflows/*.yaml`.

markovd currently models workflows as a single `yaml` text blob. That assumption
exists in the database schema, API payloads, runner interfaces, Kubernetes
ConfigMap mounting, project imports, diagram generation, and the UI.

If markovd keeps treating workflows as one YAML string, it cannot faithfully
manage directory-based Markov workflows.

## Decision

Introduce a workflow definition model that supports both input forms:

```go
type WorkflowDefinition struct {
    Kind  string
    Files []WorkflowDefinitionFile
}

type WorkflowDefinitionFile struct {
    Path    string
    Content string
}
```

`Kind` is one of:

- `file`
- `directory`

Single-file workflows are represented as one file. Directory workflows are
represented as a set of relative paths and file contents rooted at the workflow
directory.

Keep the existing `yaml` field during migration for backward compatibility with
current API clients and UI code, but make the explicit definition model the
canonical representation for new behavior.

## Rationale

This keeps markovd aligned with Markov's execution contract: Markov accepts an
input path, and that path may be either a file or a directory. A file-set model
lets markovd materialize either form for validation, shell execution,
Kubernetes execution, project import, and diagram generation.

Storing directory workflows as a synthetic merged YAML string was rejected
because it would lose source file boundaries, make project re-sync harder, and
hide the layout users actually maintain in Git.

Treating directory workflows only as project-sourced paths was rejected because
manual upload should also support directory workflows.

## Consequences

Positive:

- markovd can upload, import, validate, diagram, and run both Markov input forms.
- Existing single-file workflows can remain compatible.
- Project-sourced directory workflows can preserve source-relative file paths.
- The UI can present directory workflows as file trees instead of opaque merged
  YAML.

Negative:

- Workflow persistence and API responses become more complex.
- Runners must materialize multiple files safely.
- Kubernetes ConfigMap mounting needs explicit key-to-path handling.
- Diagram generation must stop assuming one YAML document.

## Implementation Notes

- Add persistence fields such as `definition_kind` and `definition_json`.
- Backfill existing workflows as `file` definitions.
- Validate relative paths and reject absolute paths or path traversal.
- Materialize definitions to a temporary file or directory and run
  `markov validate <path>` before storing or running.
- In Kubernetes runner mode, mount directory definitions at a workflow root path
  such as `/etc/markov/workflow` and run `markov run /etc/markov/workflow`.
- Keep legacy `{ name, yaml }` create/update payloads working while the UI and
  clients migrate.

## Related

- [Workflow Input Formats Plan](../plans/001-workflow-input-formats.md)
- [M1: Workflow Input Formats](../milestones/M1-workflow-input-formats.md)
