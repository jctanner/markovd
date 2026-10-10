# Task: Cancelling a run stops the Jobs its steps are attached to

## Status

Done (2026-10-10)

## Problem

Cancelling (or deleting) a run deleted its Markov runner Job only. Jobs the
run had started or was waiting on kept running:

- dashboard jobs submitted over HTTP and watched with `k8s_job_wait`;
- Jobs Markov created with `k8s_job`.

In Breadboard's strat-workflow benchmark, a cancelled run left an S2 job
running. The next run's preflight check refused to start until that job
was stopped by hand.

## Change

`stopStepJobs` runs on cancel (`POST /runs/{id}/cancel`, and `/jobs/cancel`
with kind `run`) and on delete, after the runner is stopped so nothing new
is submitted:

- for each running or pending step of the run with a `job_name`
  (`DB.ListActiveStepJobs`), it deletes that Job with the same call the
  step cancel already used;
- it marks the step cancelled.

This works now that Markov's `k8s_job_wait` reports its Job through
`job_created` (Markov `ea133d5`). A Job submitted in the last instant before
its wait step reports it can still be missed.

## Verification

- `go build`, `go vet` and `go test ./...` pass.
- On the cluster:
  - created a `sleep 600` Job (`cancel-test-job`) by hand;
  - started a one-step workflow whose `k8s_job_wait` watched it, and
    waited for the step to report `job_name: cancel-test-job`;
  - cancelled the run.

  Both the runner Job and `cancel-test-job` were deleted, the step became
  `cancelled`, and markovd logged "Stopped job cancel-test-job of step
  main/wait_for_external". The test run and workflow were deleted
  afterwards.
