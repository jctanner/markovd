---
title: Run ID mismatch between markovd job name and markov internal run ID
severity: high
component: runner, markov
---

## Problem

When markovd creates a k8s Job, it uses the Job name (e.g., `markov-run-e11b2157`) as the run ID and inserts a `runs` row with that ID. However, markov generates its own 8-char internal run ID (e.g., `f8b27a8a`) and uses that in all callback events. The result is two unlinked run records in the database:

| Row | run_id | status | source |
|-----|--------|--------|--------|
| 14 | `markov-run-e11b2157` | running (forever) | Created by markovd at job launch |
| 15 | `f8b27a8a` | completed | Created by callback events |

The UI fetches the run by the job name (`markov-run-e11b2157`), which is stuck at "running" with no steps. The actual step and event data is under `f8b27a8a`, which the UI never queries.

## Evidence

```sql
SELECT id, run_id, workflow_name, status FROM runs ORDER BY id DESC LIMIT 5;

 id |       run_id        | workflow_name |  status
----+---------------------+---------------+-----------
 15 | f8b27a8a            | hello         | completed
 14 | markov-run-e11b2157 | hello-world   | running
```

```sql
SELECT id, run_id, step_name, status FROM steps WHERE run_id = 'f8b27a8a';

 id |  run_id  | step_name |  status
----+----------+-----------+-----------
 25 | f8b27a8a | combine   | completed
 23 | f8b27a8a | show-date | completed
 21 | f8b27a8a | say-hello | completed
```

Steps for `markov-run-e11b2157`: none.

## Root Cause

1. `internal/runner/k8s.go:generateRunID()` creates a Job name like `markov-run-e11b2157`
2. `internal/api/runs.go:handleCreateRun()` inserts a `runs` row with that Job name as `run_id`
3. markov internally generates its own 8-char run ID and uses it in all `--callback` event payloads
4. `internal/api/events.go:processEvent()` calls `UpsertRunFromEvent()` which creates a *new* run row for the unknown callback run ID
5. The two records are never linked

## Possible Fixes

### Option A: Pass run ID from markovd to markov

Add a `--run-id` flag to markov's `run` command so markovd can force the run ID:

```go
// k8s.go
args = append(args, "--run-id", runID)
```

markov would use this as its run ID instead of generating one, so all callbacks use the same ID that markovd stored.

### Option B: Map callback run ID back to job run ID in markovd

Store the job name → markov run ID mapping when the first callback arrives, and redirect all queries. More complex and fragile.

### Option C: Use the job name as a label/var and correlate

Pass the job name as a `--var` and have markov include it in callback payloads. markovd then updates the original run record. Also fragile.

**Option A is the cleanest fix** — a single `--run-id` flag keeps both sides in sync with no mapping layer.

## Resolution

Implemented Option A. Both runners now generate the run ID upfront and pass it to markov via `--run-id`, so callbacks use the same ID that markovd stores in the database.

### markovd changes

**`internal/runner/k8s.go`** — `Start()` already generated the run ID before creating the Job. Added `"--run-id", runID` to the container args so the K8s Job passes it through to markov.

**`internal/runner/shell.go`** — `Start()` now generates the run ID upfront with `generateRunID()` and passes `"--run-id", runID` in the command args. Removed the previous stdout regex parsing that tried to extract markov's self-generated run ID, which was the source of the mismatch.

**`internal/runner/k8s_test.go`** — `TestStartCreatesConfigMapAndJob` updated to verify `--run-id` appears in the Job container args with the correct value.

### markov changes (separate repo)

markov's `run` command needs a `--run-id` flag that, when provided, uses the given ID instead of generating its own. This is being implemented separately.
