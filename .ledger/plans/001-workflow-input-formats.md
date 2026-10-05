# Workflow Input Formats Plan

## Goal

Support both Markov workflow input forms throughout markovd:

- Single self-contained YAML workflow file.
- Directory workflow project using Markov's conventional layout:
  `meta.yaml`, `vars.yaml`, `rules.yaml`, `step_types.yaml`, and `workflows/*.yaml`.

The user-facing workflow catalog should treat both forms as first-class runnable
workflow definitions.

Decision record: [ADR-0002: Support File and Directory Workflow Definitions](../decisions/ADR-0002-workflow-definition-input-formats.md)

## Current Single-File Assumptions

### Persistence

- `workflows.yaml` stores exactly one text blob.
- `models.Workflow` exposes `yaml` as the only workflow definition payload.
- Project imports store one `source_path` pointing to one YAML file.

### API

- `POST /api/v1/workflows` accepts `{ name, yaml }`.
- `PUT /api/v1/workflows/{name}` accepts `{ yaml }`.
- `GET /api/v1/workflows/{name}` returns one `yaml` string.
- `GET /api/v1/workflows/{name}/diagram` calls `generateDiagramFromYAML(wf.YAML)`.
- `POST /api/v1/runs` loads a workflow by name and passes `wf.YAML` into the runner.
- Project APIs list/import individual YAML files only.

### Runner

- `runner.RunRequest` has `WorkflowYAML string`.
- `ShellRunner` writes one temporary `markov-workflow-*.yaml` file and runs it.
- `KubernetesRunner` creates one ConfigMap key, `workflow.yaml`, mounts it at
  `/etc/markov`, and runs `markov run /etc/markov/workflow.yaml`.

### UX

- Workflows page upload modal is a name plus YAML textarea.
- Workflow detail page displays and edits one YAML string.
- Projects page says "Workflow Files" and imports selected YAML files.
- Trigger Run only needs a workflow name; it cannot show whether the selected
  definition is file-backed or directory-backed.

## Target Model

Introduce a workflow definition abstraction instead of assuming one YAML string:

```go
type WorkflowDefinition struct {
    Kind  WorkflowDefinitionKind
    Files []WorkflowDefinitionFile
}

type WorkflowDefinitionFile struct {
    Path    string
    Content string
}

type WorkflowDefinitionKind string

const (
    WorkflowDefinitionFileKind WorkflowDefinitionKind = "file"
    WorkflowDefinitionDirectoryKind WorkflowDefinitionKind = "directory"
)
```

For single-file workflows, store one file with a stable path such as
`workflow.yaml`. For directory workflows, store the directory files using paths
relative to the workflow root.

Keep `workflows.yaml` during the transition for backward compatibility, but add
new columns that make the canonical shape explicit:

- `definition_kind TEXT NOT NULL DEFAULT 'file'`
- `definition_json JSONB NOT NULL DEFAULT '[]'`
- `source_kind TEXT NOT NULL DEFAULT 'manual'`
- `source_root TEXT NOT NULL DEFAULT ''`

Backfill existing rows as:

- `definition_kind = 'file'`
- `definition_json = [{"path":"workflow.yaml","content":yaml}]`

## Storage and Migration Design

### Schema

Add these columns to `workflows`:

```sql
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS definition_kind TEXT NOT NULL DEFAULT 'file';
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS definition_json JSONB NOT NULL DEFAULT '[]';
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS source_kind TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS source_root TEXT NOT NULL DEFAULT '';
```

Keep `yaml TEXT NOT NULL` for compatibility during the first migration. For
directory workflows, store an empty string or a generated read-only summary in
`yaml`; API clients must use `definition_kind` and `files` for canonical data.

### Backfill

After adding the columns, backfill any row whose `definition_json` is empty:

```json
[
  {
    "path": "workflow.yaml",
    "content": "<existing workflows.yaml value>"
  }
]
```

Backfill must be idempotent because `migrate()` runs at application startup.

### Name and Source Semantics

- `name` remains the workflow catalog key and must stay unique.
- `source_path` continues to point at the imported source root for both file and
  directory definitions.
- `source_root` stores the repository-relative directory root for directory
  workflows; for file workflows it can match `source_path`.
- Project deletion keeps existing behavior: imported workflows become manual or
  detached according to the current `ON DELETE SET NULL` semantics.

### Compatibility Window

- Existing clients can keep sending `{ name, yaml }` for file workflows.
- New clients should send `{ name, definition_kind, files }`.
- Once all UI/API clients are migrated, a later ADR can decide whether to remove
  or deprecate `yaml`.

## API Design

### Workflow Create/Update

Support both legacy and new payloads.

Legacy file upload remains valid:

```json
{
  "name": "hello",
  "yaml": "entrypoint: main\n..."
}
```

New file upload:

```json
{
  "name": "hello",
  "definition_kind": "file",
  "files": [
    { "path": "workflow.yaml", "content": "entrypoint: main\n..." }
  ]
}
```

New directory upload:

```json
{
  "name": "hello-dir",
  "definition_kind": "directory",
  "files": [
    { "path": "meta.yaml", "content": "entrypoint: main\n" },
    { "path": "vars.yaml", "content": "greeting: hello\n" },
    { "path": "rules.yaml", "content": "[]\n" },
    { "path": "step_types.yaml", "content": "echo_local:\n  base: shell_exec\n" },
    { "path": "workflows/main.yaml", "content": "name: main\nsteps: []\n" }
  ]
}
```

Validation rules:

- `definition_kind` must be `file` or `directory`.
- File definitions must contain exactly one `.yaml` or `.yml` file.
- Directory definitions must contain the required Markov category files.
- All file paths must be relative, clean, non-empty, and must not escape the root.
- Duplicate file paths after cleaning are rejected.
- Directory definitions must include at least one `workflows/*.yaml` file.
- The server should materialize the definition to a temp directory and run
  `markov validate <path>` before accepting create/update/import.

### Workflow Validate

Add an explicit validation endpoint so the UI can validate before saving:

```http
POST /api/v1/workflows/validate
```

Request body uses the same definition payload as create/update. Response:

```json
{
  "valid": false,
  "error": "missing required directory workflow file: rules.yaml"
}
```

The create/update/import endpoints must still validate server-side even if the
UI calls this endpoint first.

### Workflow Read/List

Return both legacy and new fields during migration:

```json
{
  "id": 1,
  "name": "hello-dir",
  "definition_kind": "directory",
  "source_kind": "manual",
  "source_root": "",
  "files": [
    { "path": "meta.yaml", "content": "entrypoint: main\n" }
  ],
  "yaml": ""
}
```

For file workflows, `yaml` may continue to mirror the single file content so
older UI code remains compatible until it is migrated.

The list endpoint may omit file contents later for payload size, but the first
implementation can keep the existing full workflow shape. If payload size
becomes a problem, add `GET /workflows/{name}/files` or a `?include_files=false`
query instead of overloading the meaning of `files`.

### Project Import

Replace "list YAML files" with "discover workflow definitions":

```json
[
  {
    "path": "pipelines/hello.yaml",
    "kind": "file",
    "name": "pipelines-hello",
    "imported": false
  },
  {
    "path": "pipelines/hello-dir",
    "kind": "directory",
    "name": "pipelines-hello-dir",
    "imported": false
  }
]
```

Import request:

```json
{
  "definitions": [
    { "path": "pipelines/hello.yaml", "kind": "file" },
    { "path": "pipelines/hello-dir", "kind": "directory" }
  ]
}
```

For compatibility, keep accepting the current `{ "files": ["path.yaml"] }`
payload as file-only imports.

Discovery should:

- Continue finding standalone `.yaml` and `.yml` files.
- Detect directory workflows by checking for `meta.yaml`, `vars.yaml`,
  `rules.yaml`, `step_types.yaml`, and `workflows/`.
- Avoid listing internal files of a detected workflow directory as separate
  standalone imports.
- Import by root path and kind, not just file path.
- Derive default catalog names from the source root, but allow a future UI/API
  override if name collisions become common.

### Diagram

Do not keep a separate single-file-only parser in markovd long term. Prefer one
of these approaches:

1. Use Markov as the source of truth by adding/using a Markov command or library
   that resolves either file or directory input and emits diagram JSON.
2. Short-term: materialize the definition and add a markovd loader that merges a
   directory into the existing in-memory diagram schema before rendering.

The short-term loader must match Markov's directory contract exactly.

### Runs

`POST /api/v1/runs` should keep accepting:

```json
{
  "workflow_name": "hello-dir",
  "vars": {},
  "debug": false
}
```

The run API does not need a new field because the workflow catalog entry already
contains the definition kind and file set.

## Runner Design

Change `RunRequest` from one YAML string to a workflow source:

```go
type RunRequest struct {
    Workflow WorkflowDefinition
    Vars map[string]string
    ...
}
```

### Shell Runner

- For `file`, write a temp `.yaml` file and run `markov run <temp-file>`.
- For `directory`, create a temp directory, write all relative files, and run
  `markov run <temp-dir>`.
- Remove temp files/directories after the process exits.

### Kubernetes Runner

- For `file`, preserve the existing ConfigMap key `workflow.yaml` and command
  path `/etc/markov/workflow.yaml`.
- For `directory`, create a ConfigMap containing every definition file and use
  ConfigMap `items` with per-key paths, or create key names from sanitized paths
  and map each key back to its relative file path.
- Mount the ConfigMap at `/etc/markov/workflow`.
- Run `markov run /etc/markov/workflow`.
- Keep labels and cleanup behavior unchanged.

Kubernetes path handling must be explicit because ConfigMap keys cannot always
be used as arbitrary nested paths without `items`.

Recommended ConfigMap shape for directory workflows:

- Store each content value under a safe key such as `f-000`, `f-001`, ...
- Set `items[].key` to that safe key.
- Set `items[].path` to the validated relative workflow path.

This avoids depending on Kubernetes ConfigMap key rules for nested workflow file
paths while preserving the directory layout inside the mounted volume.

## Validation and Materialization

Add one shared helper used by API validation, runners, project import, and
diagram generation.

Responsibilities:

- Normalize and validate definition kind.
- Clean and validate relative paths.
- Reject absolute paths, empty paths, `..`, duplicate paths, and directory paths.
- Validate required directory workflow files.
- Materialize a definition to a temp file or temp directory.
- Return the path to pass to `markov validate`, `markov run`, or diagram loading.
- Clean up temp files/directories reliably.

Materialization should not mutate content or merge files. Markov remains the
source of truth for workflow schema validation.

## UX Design

### Workflows List

- Add a Type column: `File` or `Directory`.
- Show Source as `Manual` or `Project`, and include `source_path` when present.
- Preserve existing delete behavior.

### Manual Upload

Use a mode switch:

- `Single file`: current YAML textarea.
- `Directory`: file set editor.

Directory upload should support:

- Browser directory selection where available.
- Manual add/edit/remove of relative file paths.
- Inline required-file checklist.
- Server-side validation errors surfaced in the modal.

### Workflow Detail

- Replace "YAML Definition" with "Definition".
- File workflows keep a single editor.
- Directory workflows show a file tree and editor/viewer for the selected file.
- Disable editing for project-sourced workflows as today.
- Show type, source path/root, and file count.

### Projects

- Rename "Workflow Files" to "Workflow Definitions".
- Show kind badges for file vs directory.
- Allow importing directory roots.
- Disable selecting internal files that belong to an already discovered directory
  workflow.

### Trigger Run / Rerun

- Display the selected workflow type and source.
- No new user input is required to run a directory workflow after it is imported
  or uploaded; `POST /runs` still accepts `workflow_name`.

## Testing Strategy

### Unit Tests

- Definition validation:
  - valid file definition
  - valid directory definition
  - missing `meta.yaml`, `vars.yaml`, `rules.yaml`, `step_types.yaml`
  - missing `workflows/*.yaml`
  - absolute path rejection
  - `..` path traversal rejection
  - duplicate normalized paths rejection
- Database:
  - migration/backfill existing `yaml` rows
  - create/update/list/get file definitions
  - create/update/list/get directory definitions
  - project import rows preserve `source_path`, `source_root`, and kind
- Project discovery:
  - standalone YAML files are discovered
  - directory workflow roots are discovered
  - internal files of directory workflows are not presented as separate imports
- Runner:
  - shell runner passes temp file path for file definitions
  - shell runner passes temp directory path for directory definitions
  - Kubernetes runner preserves current single-file ConfigMap behavior
  - Kubernetes runner maps directory file keys to nested mounted paths
- Diagram:
  - file definition diagram remains unchanged
  - directory definition diagram includes workflows from `workflows/*.yaml`

### API Tests

- Legacy `{ name, yaml }` create/update still works.
- New file and directory create/update payloads work.
- Invalid definitions return HTTP 400 with actionable errors.
- Project import accepts legacy `files` and new `definitions` payloads.
- `POST /runs` works for both workflow kinds without a request shape change.

### UI Tests

- Workflows list shows type/source.
- Manual upload can create a file workflow.
- Manual upload can create a directory workflow.
- Workflow detail displays a file workflow as one editable definition.
- Workflow detail displays a directory workflow as a file tree/editor.
- Projects import UI distinguishes file and directory definitions.
- Trigger Run and Rerun show workflow type/source and submit unchanged run
  payloads.

### End-to-End Compose Verification

Before compose verification, refresh the Markov binary used by markovd:

```bash
cp ../markov/bin/markov ./bin/markov
```

Then rebuild/start the stack:

```bash
make compose-build
make compose-up
```

Verify at least:

1. Upload or import a single-file workflow, diagram it, run it, and inspect run
   detail.
2. Upload or import a directory workflow, diagram it, run it, and inspect run
   detail.
3. Re-sync a project containing a directory workflow and confirm the catalog
   definition updates.
4. Submit an invalid directory definition and confirm validation fails before a
   workflow is stored or run.

Record the copied Markov binary path, compose commands, workflow names, run IDs,
and validation failures in the relevant completed task notes.

## Implementation Phases

### Phase 1: Domain and Persistence

- Add workflow definition structs.
- Add migrations for `definition_kind`, `definition_json`, `source_kind`, and
  `source_root`.
- Backfill existing workflows from `yaml`.
- Update DB create/update/import/list/get helpers.
- Keep legacy `yaml` responses for compatibility.

### Phase 2: Validation and Materialization

- Add path validation and a materializer helper.
- Add `POST /api/v1/workflows/validate`.
- Add server-side `markov validate` integration for both file and directory
  definitions.
- Add unit tests for valid/invalid definitions and path traversal rejection.

### Phase 3: Runner Support

- Change `RunRequest` to carry `WorkflowDefinition`.
- Update shell and Kubernetes runners.
- Add tests for shell temp directory behavior and Kubernetes ConfigMap item
  mapping.

### Phase 4: Project Discovery and Import

- Replace `ListYAMLFiles` with `ListWorkflowDefinitions`.
- Add directory workflow detection.
- Update import and re-sync to read all files under directory workflow roots.
- Keep existing file imports working.
- Keep legacy `{ files: [...] }` imports working for file-only workflows.

### Phase 5: Diagram Support

- Short term: add a definition-aware diagram loader.
- Long term: move diagram generation to Markov's resolved workflow model so
  markovd does not duplicate parser behavior.

### Phase 6: UI Updates

- Update API types.
- Add workflow type badges and definition viewers/editors.
- Add directory upload/import UX.
- Update Trigger Run and Rerun surfaces to show definition type.

### Phase 7: Documentation and Compatibility

- Update README and operational notes.
- Document API compatibility and migration.
- Add release notes for legacy `yaml` behavior.

### Phase 8: End-to-End Verification

- Build or otherwise obtain a Markov binary that includes directory workflow
  support in the sibling Markov repo.
- Copy the updated binary into markovd before starting the compose stack:

```bash
cp ../markov/bin/markov ./bin/markov
```

- Rebuild and start the markovd compose stack after the copy so containers use
  the updated Markov binary.
- Verify both a single-file workflow and a directory workflow can be uploaded or
  imported, validated, diagrammed, run, and inspected through the UI/API.
- Record the copied Markov binary source, compose command, test workflow names,
  run IDs, and observed results in the completed task notes.

## Rollout and Rollback

Rollout:

- Ship the schema migration with backward-compatible API responses first.
- Keep legacy file workflow upload/update paths operational.
- Update backend runner/API paths before enabling directory upload controls in
  the UI.
- Gate UI directory upload behind the presence of backend `definition_kind`
  support.

Rollback:

- Existing single-file workflows remain runnable through the legacy `yaml`
  column.
- Directory workflows may become unreadable by older markovd versions; document
  this as a forward migration limitation.
- Avoid deleting or rewriting the legacy `yaml` column until after directory
  support has been validated in production-like compose and Kubernetes runs.

## Open Questions

- Should directory workflow file contents be returned in `GET /workflows`, or
  should list responses omit contents and detail responses include them?
- Should markovd call `./bin/markov validate` directly, or should the Markov
  binary path be configurable for validation just like runner execution?
- Should workflow definition files be stored in `workflows.definition_json` or a
  normalized child table if large directory workflows become common?
- Should diagram generation delegate to Markov once Markov exposes a resolved
  workflow or diagram command that supports directory input?

## Acceptance Criteria

- Existing single-file workflows continue to upload, edit, import, diagram, run,
  rerun, and delete.
- Directory workflows can be uploaded manually and imported from a project.
- Directory workflows validate before being stored.
- Directory workflows run successfully in shell and Kubernetes runner modes.
- Workflow diagrams work for both definition kinds.
- Project re-sync updates directory workflow definitions when any contained file
  changes.
- The UI clearly distinguishes file and directory definitions.
- Path traversal and absolute-path inputs are rejected.
- Unit, API, runner, project discovery, diagram, and UI tests cover both workflow
  definition kinds.
- Compose-based verification copies `../markov/bin/markov` to `./bin/markov`
  before building or starting the stack, so markovd tests use a Markov binary
  with directory workflow support.
