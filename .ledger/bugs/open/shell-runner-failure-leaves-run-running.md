---
title: Run stays "running" when markov exits before emitting events
severity: medium
component: runner
---

## Problem

With the shell runner, if the `markov` process exits with an error before it sends any callback event, markovd keeps the run in `running` indefinitely. The error is only visible in the server log.

## Reproduction

1. Start markovd with `MARKOVD_RUNNER=shell` and leave `TMPDIR` at `/tmp` on a machine where `/tmp` contains a socket (for example `/tmp/.ICE-unix/*`).
2. Create and trigger any workflow.
3. markov fails at startup: `fingerprinting workflow source: ... unsupported workflow source entry "/tmp/.ICE-unix/2944"`.
4. `GET /api/v1/runs/{id}` stays `running` (observed for 2 minutes) with no steps.

## Expected

The run is marked `failed` with the process error when the runner process exits non-zero and no terminal event was received.

## Actual

Status never leaves `running`.

## Notes

Found while verifying [claude step progress](../../tasks/done/claude-step-progress-ui.md). The trigger here is the shell runner materializing the workflow into the shared temp directory; the missing failure propagation is the bug. A separate improvement would be to materialize workflows into a private directory.
