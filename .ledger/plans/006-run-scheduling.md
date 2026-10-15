# Persistent run scheduling

Support one-time timestamps and five-field cron schedules with IANA time zones.
The existing Trigger Run inputs apply to schedules: workflow, entrypoint, variables,
debug, PVC mounts and Secret references. Freeze the imported definition and source
commit when saving; editing preserves that snapshot unless the workflow changes.

Use PostgreSQL row locks to claim a due occurrence and record its run ID before
calling the runner. Never blindly retry an uncertain launch. Recurring schedules
do not overlap their own active run; missed times coalesce into one launch after
recovery. Store occurrence history, expose errors, and allow edit/pause/resume/delete.

Verify cron/DST and validation, PostgreSQL concurrent claiming, restart recovery,
payload preservation, API authentication, UI builds and harmless deployed runs.
No model calls are needed to test scheduling.
