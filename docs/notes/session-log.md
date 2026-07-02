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
- Start with `docs/tasks/pending/workflow-definition-model.md`.
