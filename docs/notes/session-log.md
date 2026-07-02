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
