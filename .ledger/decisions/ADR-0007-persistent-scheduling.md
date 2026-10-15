# Persistent scheduling with conservative dispatch recovery

Status: accepted.

Schedules live in PostgreSQL and store frozen workflow definitions and launch
inputs. Row locks with SKIP LOCKED claim each due occurrence. The transaction
records its stable run ID before the external runner is invoked. Both runners
accept this ID. Callback credentials are injected at dispatch, not saved in inputs.

Cron uses five fields and an explicit IANA time zone, with embedded Go tzdata.
Missed times coalesce, and a schedule never overlaps its own pending/running/paused
run. Pausing or deleting affects future launches, not already dispatched work.

External launch and PostgreSQL cannot share a transaction. If launch is uncertain,
or a process crashes before recording the result, recovery inspects persisted run
status and stops the schedule if no callback/dispatch evidence exists. It does not
automatically launch a second potentially paid job. This favors visible missed
work over duplicate effects; exactly-once external execution is not promised.
