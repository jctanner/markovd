# Task: Keep all files on project import and record the source commit

## Status

Current

## Summary

Two gaps in importing directory workflows from git projects, found while
planning Breadboard's strat-workflow benchmark
(`.ledger/plans/strat-workflow-eval-plan.md` in Breadboard):

1. **Non-YAML files were dropped.** `ReadWorkflowDefinition` in
   `internal/projects/git.go` kept only `.yaml`/`.yml` files under a
   directory workflow root. `scripts/` (read by `script_exec` `path:`), data
   files and READMEs never reached the runner. The CLI's directory upload
   (`readWorkflowDirectory` in `cmd/markovd-cli/main.go`) keeps every file,
   and plan 001 already said "Update import and re-sync to read all files
   under directory workflow roots".
2. **No record of the source commit.** An imported workflow stored its
   project and path but not the commit, so a run could not be traced back
   to the exact workflow, scripts and data that produced it.

## Changes

- `ReadWorkflowDefinition` (directory kind) keeps every file under the
  root and skips hidden directories such as `.git`, matching the CLI upload.
- `projects.HeadCommit(repoPath)` returns the clone's HEAD commit.
- `workflows.source_commit` and `runs.workflow_source_commit` columns,
  added by an idempotent migration in `internal/db/db.go`.
- Project import and re-sync store the HEAD commit on each imported
  workflow. A manual upload or an edit through the API clears it, because
  the content no longer matches a commit.
- `CreateRun` copies the workflow's `source_commit` into the run, so a later
  re-import does not change what an earlier run reports.
- Starting a run passes the workflow's commit to it as the
  `workflow_source_commit` var (unless the caller set that var), so a
  workflow can pin what it fetches, such as job scripts cloned from the
  same repository, to the commit it was imported from.
- API JSON: `source_commit` on workflows, `workflow_source_commit` on runs.
- UI: the workflow page shows the commit next to "Source"; the run page has a
  "Source commit" card when the run has one.

## Verification

- `go build ./...`, `go vet ./internal/...`, `go test ./...`: pass.
- New tests: `TestReadWorkflowDefinitionDirectoryKeepsNonYAMLFiles`
  (scripts, data and README kept; hidden directory skipped) and
  `TestHeadCommit`, and `TestWithSourceCommit` for the run var.
- `npx tsc --noEmit` in `ui/`: pass.
- Deployed with Breadboard's `make host-rebuild-markovd` (2026-10-09): pod
  2/2 Running, startup log clean, `/api/v1/health` 200. Postgres has
  `workflows.source_commit` and `runs.workflow_source_commit`. Existing
  imported workflows have an empty commit until their project's next sync.
- End-to-end, 2026-10-10, after `make host-rebuild-markovd` with `d923fe5`:
  synced project 1 (`ai-first-pipeline`, Breadboard) and imported
  `var/benchmarks/strat-workflow`. The workflow
  `var-benchmarks-strat-workflow` has `source_commit` `921cce9…` (Breadboard
  HEAD) and all 30 files, including `scripts/*.py`, `scripts/*.sh`, `data/`
  and `README.md`.
- Still to see: a run started from markovd carrying `workflow_source_commit`
  in its vars (the first benchmark run will show it).
