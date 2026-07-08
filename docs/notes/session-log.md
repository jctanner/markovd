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

## 2026-07-08

Agent: codex

Completed:
- Implemented Phase 1 of the Run Graph Workflow Boundaries plan.
- Added non-interactive workflow boundary nodes behind nested Run Detail graph
  steps, derived from exact `fork_id` workflow scopes.
- Styled workflow boundaries with quiet dashed containers and compact labels.
- Recorded screenshot evidence in
  `docs/plans/run-graph-workflow-boundaries.png`.

Verified:
- Playwright against `http://127.0.0.1:5173/runs/markov-run-b09c12f9`
  rendered 9 workflow boundary nodes for the end-to-end nested workflow run.
- Playwright containment checks confirmed representative child steps were
  inside `run_pipeline`, nested `run_pipeline-*`, and `reset_github`
  boundaries.
- Playwright confirmed step clicks still open `StepDetailModal`.
- Playwright against `markov-run-a25ff450` confirmed a step with top-level
  `output_json.job_name` still shows a Logs section in the modal.
- `npm run build` in `ui/`.
- `npx eslint src/components/WorkflowGraph.tsx` in `ui/`.

## 2026-07-08

Agent: codex

Completed:
- Added `docs/fixtures/workflows/graph-boundary-noop/`, a fast no-op directory
  workflow for reproducing Run Detail graph boundary label and edge crowding
  issues without running the full ai-first-pipeline end-to-end demo.
- Updated the open graph boundary overlap bug to use this fixture as the
  preferred reproducer.

Verified:
- `../markov/bin/markov validate docs/fixtures/workflows/graph-boundary-noop`
- `../markov/bin/markov run docs/fixtures/workflows/graph-boundary-noop --state-store /tmp/markovd-graph-boundary-noop-2.db --run-id graph-boundary-noop-smoke-2 --verbose`

## 2026-07-08

Agent: codex

Completed:
- Fixed the Run Detail workflow boundary label overlap bug using the fast no-op
  graph fixture instead of the slow ai-first-pipeline end-to-end demo.
- Stopped rendering workflow boundary containers around expanded `for_each`
  branch paths while preserving boundaries around actual workflow calls.
- Added clearer workflow boundary label separation and more header padding.
- Moved the bug record to `docs/bugs/fixed/` with Playwright screenshot
  evidence.

Verified:
- `../markov/bin/markov validate docs/fixtures/workflows/graph-boundary-noop`
- `../markov/bin/markov run docs/fixtures/workflows/graph-boundary-noop --state-store /tmp/markovd-graph-boundary-noop-3.db --run-id graph-boundary-noop-smoke-3 --verbose`
- Imported `graph-boundary-noop` into `markovd.local` and triggered
  `markov-run-a8dc339b`, which completed with 34 steps.
- Playwright rendered `markov-run-a8dc339b` on the Graph tab through the local
  Vite frontend and found no workflow boundary label overlaps or console
  errors.
- `npm run build` in `ui/`.
- `npx eslint src/components/WorkflowGraph.tsx` in `ui/`.

## 2026-07-08

Agent: codex

Completed:
- Added `docs/plans/003-markovd-cli.md`, a plan for `./bin/markovd-cli` with
  full API CRUD coverage and primary flows for project sync/wait and workflow
  trigger/wait.
- Added `docs/tasks/pending/markovd-cli.md` so the implementation has a ledger
  task.
- Linked the plan and pending task from `PLAN.md`.
- Tightened the CLI plan and task requirements for easy username/password
  login, `--password-stdin`, self-signed certificate support, and custom CA
  trust.
- Added project-local TOML config support to the CLI plan:
  `./.markovd-cli-config.toml` can hold instance URL, credentials, TLS
  verification behavior, custom CA path, and command defaults.
- Corrected `PLAN.md` so the CLI task is listed as pending, matching its
  filesystem ledger location.

Verified:
- Reviewed current API routes in `internal/api/router.go`,
  `internal/api/projects.go`, `internal/api/runs.go`, and
  `internal/api/workflows.go` to map CLI commands to existing endpoints.

## 2026-07-08

Agent: codex

Completed:
- Implemented `./bin/markovd-cli` as a wrapper around the Go command in
  `cmd/markovd-cli`.
- Added a markovd API client with token injection, username/password login,
  JSON error decoding, TLS verification by default, self-signed certificate
  opt-out, and custom CA support.
- Added TOML config loading for `./.markovd-cli-config.toml`, explicit
  `--config`, and user config with precedence below environment variables and
  CLI flags.
- Implemented project, workflow, run, PVC, Secret, and preferences commands
  from the CLI plan, including `projects sync --wait` and
  `runs create --workflow ... --wait`.
- Added strict PVC and Secret mount parsing for `NAME:/absolute/path`.
- Added `.markovd-cli-config.toml` to `.gitignore` to reduce accidental
  credential commits.
- Moved the CLI task to `docs/tasks/done/markovd-cli.md`.

Verified:
- `env GOCACHE=/tmp/go-build-cache go test ./cmd/markovd-cli`
- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli` printed CLI usage.
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --output json health`
  returned healthy.
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --username admin --password admin --output json projects list`
  authenticated and listed projects.
- `printf '%s\n' admin | env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --username admin --password-stdin --output json auth login`
  authenticated through standard input.
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --username admin --password admin --timeout 2m --poll-interval 2s runs create graph-boundary-noop --workflow pipeline --wait`
  triggered `markov-run-cfb29a83` and waited until it completed.

## 2026-07-08

Agent: codex

Completed:
- Added `docs/reference/markovd-cli.md`, a full user reference for
  `./bin/markovd-cli`.
- Documented authentication, environment variables, self-signed TLS handling,
  local TOML config, output formats, project/workflow/run commands, volume
  mounts, wait behavior, exit codes, and common recipes.
- Linked the CLI reference from `PLAN.md`.
- Updated `preferences set` so JSON-looking values are sent as JSON arrays or
  objects, matching the documented default volume example.

Verified:
- `env GOCACHE=/tmp/go-build-cache go test ./cmd/markovd-cli`
- `git diff --check`

## 2026-07-08

Agent: codex

Completed:
- Fixed `./bin/markovd-cli --help`, `./bin/markovd-cli -h`, and
  `./bin/markovd-cli help` so they print usage and exit successfully.
- Added a regression test for all three help forms.

Verified:
- `env GOCACHE=/tmp/go-build-cache go test ./cmd/markovd-cli`
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --help`
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli -h`
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli help`
