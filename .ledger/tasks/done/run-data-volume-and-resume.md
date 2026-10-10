# Task: Keep runs on a data volume, resume runs, show step descriptions

## Status

Done (2026-10-10)

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
- **Step errors on re-run:** a step that runs again or completes now drops
  the error from an earlier attempt. Before, a resumed step showed
  "completed" next to its first attempt's error.
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

## End-to-end check (2026-10-10)

Deployed with Breadboard's `deploy/k8s/15-markovd.yaml` (`markovd-data`
PVC). Breadboard's `var/demos/markov-feature-check` was run through the API
(no model calls), with three attempts:

1. **Start:** failed at the fan-out as designed. Item `b` failed, and `c`
   did not start. The run directory held the workflow and
   `markov-state.db`; no ConfigMap was created.
2. **Resume with `allow_b=true`:** only `b` ran. The collect step saw all
   three results, and the run went to **paused** at the gate. Resuming
   with no vars was refused with 400.
3. **Resume with `approved=true`:** completed.

Found and fixed along the way:

- `markov resume` didn't accept `--verbose` (Markov `3520707`);
- resumed numbers printed as `2.000000` (Markov `88190a4`);
- stale step errors were kept after a re-run (`3186ce0`).

The reconcile followed the resume Job (`runs.job_name`). Importing right
after a markovd restart, with no new sync, worked, because the clones are
now on the volume.

## Follow-up (2026-10-10): templated sub-workflow names

Markov `d9b2b8e` lets `workflow:` be a template, resolved when the step
runs. The diagram used to fail on such a name, as an undefined workflow. It
now shows the call with its template (`submit-{{ test.arm }}`) and doesn't
expand it (`0b3ebc7`, `TestGenerateDiagramLeavesTemplatedWorkflowUnexpanded`).
Checked on Breadboard's strat-workflow diagram: 37 nodes, with the templated
call shown.
