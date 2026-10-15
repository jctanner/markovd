# Scheduling workflow runs

In **Trigger**, choose **Once at a specific time** or **Recurring cron schedule**.
The normal workflow, entrypoint, variables, debug flag, PVC and Secret selections
also apply to scheduled runs. One-time dates use the browser's local time zone;
the form shows both local and UTC times before saving. Cron uses five fields
and an IANA time zone such as `America/New_York`.

The **Schedules** page shows the next launch, state, last run, errors and recent
dispatch history. Expand a schedule's name to inspect its saved inputs and commit.
Active and paused schedules can be edited. A past one-time date must be changed
before resuming. Pause or delete prevents future launches and leaves an already
launched run under the normal Runs controls. Delete removes schedule history,
but keeps the runs themselves.

## API

The existing JWT authentication applies to all schedule endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| GET / POST | `/api/v1/schedules` | List / create |
| GET / PUT / DELETE | `/api/v1/schedules/{id}` | Inspect / edit / delete |
| POST | `/api/v1/schedules/{id}/pause` | Pause future dispatches |
| POST | `/api/v1/schedules/{id}/resume` | Resume future dispatches |
| GET | `/api/v1/schedules/{id}/occurrences` | Most recent 100 dispatches |

Example one-time request (choose a future date):

```json
{
  "name": "Overnight batch",
  "kind": "once",
  "timezone": "America/New_York",
  "run_at": "2027-01-15T00:30:00-05:00",
  "cron": "",
  "request": {
    "workflow_name": "my-workflow",
    "workflow_entrypoint": "main",
    "vars": {"model": "example", "repeats": "1"},
    "debug": false,
    "volumes": [{"name": "artifacts", "pvc": "artifacts", "mount_path": "/out"}],
    "secret_volumes": [{"name": "auth", "secret": "my-auth", "mount_path": "/auth", "read_only": true}]
  }
}
```

For cron, use `"kind": "cron"`, `"run_at": null`, and e.g.
`"cron": "0 */6 * * *"`. Only five-field expressions are accepted, with minute
as the first field. The time zone is separate; inline `TZ` and `@every` are not
accepted. Cron follows daylight saving rules: skipped wall times do not run;
repeated wall times can occur twice. The no-overlap rule still applies.

## Execution and recovery

- Saving freezes the workflow's files and imported source commit. Later imports
  do not alter that schedule. Editing timing or inputs retains the snapshot;
  selecting a different workflow snapshots its current definition. Create a new
  schedule to adopt a new revision of the same workflow.
- PostgreSQL stores the schedule, occurrence and run identity before launch.
  Multiple server replicas use row locks to claim an occurrence once.
- Recurring schedules wait while their previous run is pending, running or
  paused. Different schedules can run concurrently. This does not impose global
  resource limits or make workflow side effects idempotent.
- After downtime, a missed one-time launch runs once. Multiple missed cron
  intervals coalesce into one launch, with the next time calculated from now.
- A schedule's `completed` state means its one-time dispatch completed. Inspect
  the linked run for the workflow's actual result.
- If launch fails ambiguously, scheduling stops with an error and a run link.
  After an interrupted dispatch, recovery waits two minutes and checks persisted
  run evidence. It never blindly retries an external launch. Inspect the run and
  runner before creating a replacement. Exactly-once external effects are not
  guaranteed by a database claim.
- Store Secret names as mount references. Avoid putting credentials directly in
  variables, since authenticated users can inspect saved launch inputs. Callback
  credentials come from the current server configuration at dispatch time.
