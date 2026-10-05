# Task: Show claude step icon and live step_progress in markovd

## Status

Done

## Summary

Markov's `claude` step emits `step_progress` callback events (assistant text, tool calls and results, running turn/token usage, limit breaches). markovd already stored every callback event but ignored the new type and had no way to read them.

## Changes

- `processEvent` handles `step_progress` explicitly as events-table only; no steps row changes.
- `GET /api/v1/runs/{runID}/progress?step=&workflow=&fork=&after=&limit=` (JWT) returns a step's progress events oldest first. `after` is the last seen event id so clients poll incrementally; `limit` defaults to 500, max 2000. Fork events match by `<root>-<fork>` and by root prefix, excluding other runs that merely share a prefix.
- `idx_events_run_id` index on `events(run_id, id)`.
- `DeleteRun` now also deletes fork events (`<root>-*`); before it removed only events whose run_id equalled the root, leaving fork events behind.
- UI: a Transcript section in the step detail modal (text, tool calls, tool results, limit and result lines, with running turns, tokens, and cost), polled every 2s until a terminal event; the `events` output array is summarized rather than dumped. A `sparkles` icon for `claude` in both graph components. Helpers live in `ui/src/components/stepProgress.ts` with tests.

## Verification

- `go vet`, `go test ./...`, `tsc -b`, `npm test` pass. `internal/db` tests (skipped unless `MARKOVD_TEST_DATABASE_URL` is set) ran against Postgres 16: step/fork/prefix/after/limit filtering and fork-event deletion.
- End to end with the shell runner: markovd + real markov + real `claude` (OAuth). A run's progress was visible while it ran and complete at the end (init, tool_use, usage, tool_result, text, usage, result); `after=<last id>` returned nothing; a missing `step` returned 400.
- Not verified in a browser: the transcript rendering and the new icon were type-checked and unit-tested only.
- `eslint` reports no new errors from these changes; the existing baseline (see the lint bug) remains.
