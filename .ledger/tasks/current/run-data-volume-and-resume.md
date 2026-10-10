# Task: Keep runs on a data volume, resume runs, show step descriptions

## Status

Current: built and unit-tested; deployment and an end-to-end check follow.

## Summary

Gaps found while running Breadboard's strat-workflow benchmark from markovd:

1. **No resume.** The API had start, cancel and delete. Markov can resume a
   failed or paused run from its state database, but that database lived
   only inside the runner pod and was lost when the pod ended. Gates were
   unusable from markovd, and a long run that failed had to start over.
2. **Run files in a ConfigMap,** capped at about 1 MiB.
3. **Project clones without a volume.** After every restart, imports failed
   with "no such file" until the project was synced again.
4. **Step descriptions** (Markov's `description` field) were not shown.

## Changes

- **Data volume** (`MARKOVD_DATA_PVC`, mounted in markovd at
  `MARKOVD_DATA_DIR`):
  - each run gets `runs/<run-id>/` holding its workflow files and Markov's
    state (`MARKOV_STATE_STORE=/run-data/markov-state.db`);
  - the runner pod mounts that directory at `/run-data` (subPath), so there
    is no ConfigMap and no size cap;
  - without the setting, the ConfigMap path is unchanged.
- **Resume:** `POST /api/v1/runs/{id}/resume {vars}`:
  - only paused or failed runs can be resumed, and a paused run needs at
    least one var;
  - starts a `<run-id>-resume-<hex>` Job running `markov resume <run-id>`
    with the same run directory, PVCs and secrets;
  - `runs.job_name` records the current Job, so the active-job reconcile
    follows the resume Job rather than the finished first one;
  - the shell runner returns `ErrResumeUnsupported`.
- **Events:** `run_paused` now sets the run to `paused` (it was ignored, so a
  paused run stayed "running"), and `run_resumed` sets it to `running`.
- **Cancel and delete:** Cancel deletes every Job labelled with the run ID
  (the first run and any resumes). Delete also removes the run's directory.
- **Projects:** `MARKOVD_PROJECTS_DIR=/data/projects` on the same volume, so
  clones survive restarts.
- **UI:**
  - a Resume button and dialog for paused and failed runs, and a `paused`
    badge;
  - step descriptions as a tooltip on structure-graph nodes and as a
    paragraph in the step detail dialog (`for_each` items and
    rescue/always steps use their step's description);
  - the diagram API carries `description`.
- **Deployment** (Breadboard `deploy/k8s/15-markovd.yaml`): a `markovd-data`
  PVC, mounted at `/data`, with the three env settings.

## Verification

- `go build ./...`, `go vet`, `go test ./...` pass.
- `ui`: `tsc --noEmit`, `npm test`, `npm run build` pass.
- New tests in `internal/runner/k8s_data_test.go`:
  - a start with the data volume writes the run's files and mounts its
    directory with the state-store env, and creates no ConfigMap;
  - resume refuses a run with no saved state, then starts a
    `markov resume` Job on the same subPath with its vars;
  - delete removes every Job and the run's directory;
  - resume without a data volume is unsupported;
  - an escaping file path is rejected.
- Existing cancel tests still pass: Cancel now lists the run's Jobs and
  deletes each, and still errors when there is none.
