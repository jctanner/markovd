# Session Log

## 2026-07-01

Agent: codex

Completed:
- Adopted the agent work ledger layout for repository docs.
- Moved the project overview to `docs/plans/000-overview.md`.
- Moved operational notes to `docs/notes/`.
- Moved open and fixed bugs into `docs/bugs/open/` and `docs/bugs/fixed/`.
- Added `PLAN.md` and `AGENTS.md` as root navigation and agent instruction files.

Next:
- Convert future work into task files under `docs/tasks/pending/`.

## 2026-07-02

Agent: codex

Completed:
- Analyzed markovd assumptions around single self-contained workflow YAML files.
- Added a plan for supporting both single-file and directory workflow definitions.
- Added ADR-0002 for the file/directory workflow definition decision.
- Added a milestone and pending task files for persistence, validation, runners,
  project import, API/UX, and diagrams.
- Added the compose verification prerequisite to refresh `./bin/markov` from
  `../markov/bin/markov` before stack startup.
- Expanded the plan with testing strategy, migration/rollback details, validation
  endpoint design, and a dedicated end-to-end verification task.

Discovered:
- Workflow storage, project import, runner materialization, diagram generation,
  and the UI all currently assume a single YAML string.

Next:
- Start with the workflow definition model task. This task later moved to
  `docs/tasks/done/workflow-definition-model.md`.

## 2026-07-02

Agent: codex

Completed:
- Implemented workflow definitions with `file` and `directory` kinds across
  persistence, API, runners, project import, diagrams, and UI.
- Added path-safe workflow definition normalization and materialization helpers.
- Added validation endpoint `POST /api/v1/workflows/validate`.
- Updated shell runner lifecycle so workflow runs outlive the create-run request
  context and complete through callbacks.
- Added runtime `git` to the API image so local project repository sync works.

Verified:
- `go test ./...`
- `npm run build`
- `cp ../markov/bin/markov ./bin/markov`
- `./bin/markov validate ../markov/examples/dir-based-hello-world`
- `./bin/markov validate ../markov/examples/k8s-job-test.yaml`
- `uv tool run podman-compose build`
- `uv tool run podman-compose up -d --force-recreate api`
- API health returned `{"status":"healthy"}`.
- Single-file workflow `single-file-smoke` validated, diagrammed, ran as
  `markov-run-43cea868`, and completed.
- Manual directory workflow `directory-smoke` validated, diagrammed, ran as
  `markov-run-20926252`, and completed.
- Invalid directory workflow validation failed before storage with missing
  `vars.yaml`.
- Project directory workflow `pipeline` imported from project `1`, diagrammed,
  ran as `markov-run-6b9c3df9`, and completed.
- Project re-sync updated `pipeline` from `project-directory-v1` to
  `project-directory-v2`; re-run `markov-run-6b6cdbd7` completed with stdout
  `project-directory-v2`.

Next:
- Commit the implementation.

## 2026-07-07

Agent: codex

Completed:
- Collapsed the Trigger Run PVC and secret volume selectors behind an Advanced
  toggle while preserving selected defaults and submission payload construction.
- Added compact selected-count text to the collapsed Advanced control.

Verified:
- `npm run build` in `ui/`.
- `npx eslint src/pages/TriggerRun.tsx` in `ui/`.

Notes:
- Full `npm run lint` in `ui/` still fails on pre-existing unrelated React lint
  errors outside the edited Trigger Run page.

## 2026-07-07

Agent: codex

Completed:
- Added an optional Trigger Run workflow entrypoint override field that maps to
  Markov's `--workflow` run flag.
- Trimmed and omitted blank workflow entrypoint overrides in the frontend API
  helper and API handler.
- Passed non-blank overrides through shell and Kubernetes runners as
  `--workflow <name>`.
- Added datalist suggestions for likely workflow names from the selected
  workflow definition. Directory definitions use `workflows/*.yaml` top-level
  `name:` fields; standalone definitions use conservative top-level
  `workflows:` entry parsing.

Verified:
- Read `../markov/docs/reference/cli.md` and
  `../markov/docs/reference/workflow-file.md`.
- `env GOCACHE=/tmp/go-build-cache go test ./internal/runner`
- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `npm run build` in `ui/`
- `npx eslint src/pages/TriggerRun.tsx` in `ui/`

## 2026-07-07

Agent: codex

Completed:
- Fixed project workflow discovery so a directory containing `meta.yaml` is
  treated as a directory workflow root.
- Added regression coverage for `var/demos/end-to-end`-style roots so internal
  YAML files under the detected root are not listed as separate importable file
  workflows.

Verified:
- `env GOCACHE=/tmp/go-build-cache go test ./internal/projects`
- `env GOCACHE=/tmp/go-build-cache go test ./...`

## 2026-07-07

Agent: codex

Completed:
- Reproduced the `var/demos/end-to-end` import failure through the running
  `https://markovd.local` API. The backend rejected the directory with
  `missing required directory workflow file: step_types.yaml`.
- Updated workflow definition normalization to accept `step_types/*.yaml` as the
  directory workflow step type source when `step_types.yaml` is absent.
- Added runtime compatibility materialization for Markov binaries that still
  require `step_types.yaml`: Markovd generates a merged `step_types.yaml` from
  `step_types/*.yaml` for validation and runs, and omits the original
  `step_types/` files from the runtime copy.
- Improved the Projects page import failure message to include backend
  per-path error details.

Verified:
- `env GOCACHE=/tmp/go-build-cache go test ./internal/workflowdef ./internal/projects`
- `env GOCACHE=/tmp/go-build-cache go test ./internal/workflowdef ./internal/projects ./internal/runner`
- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `npm run build` in `ui/`

Notes:
- `npx eslint src/pages/Projects.tsx` still fails on pre-existing React hook
  lint errors in that file.

## 2026-07-07

Agent: codex

Completed:
- Fixed the Run Detail graph layout so nested workflow calls render as nested
  lanes instead of being flattened into the parent workflow chain.
- Preserved existing branch expansion and summary behavior for `for_each`
  forks while treating exact child `fork_id` groups as sub-workflows.
- Moved the graph flattening bug record to `docs/bugs/fixed/` with Playwright
  coordinate evidence and a screenshot artifact.

Verified:
- Playwright against `http://127.0.0.1:5173/runs/markov-run-b09c12f9` showed
  `main` at `x=0`, `run_pipeline` children at `x=340`, and nested `run-skill`
  internals at `x=680`.
- `npm run build` in `ui/`.
- `npx eslint src/components/WorkflowGraph.tsx` in `ui/`.
