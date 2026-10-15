# Persistent scheduling in API and UI

Implement [the plan](../../plans/006-run-scheduling.md), deploy to markovd.local,
and verify without model calls. Record test evidence before moving to done.

Completed on feature branch `feat/run-scheduling`. API/UI support one-time and
five-field cron schedules with IANA time zones, complete launch inputs, pinned
definitions, edit/pause/resume/delete, recent occurrences and linked runs. Claims
persist before runner launch. No blind retry after uncertainty; missed cron times
coalesce and a schedule does not overlap its own active run.

Verification (2026-10-15):

- `go test -race ./...` passed with the dedicated PostgreSQL database
  `markovd_schedule_test_20261015`; includes API auth/validation, full payload and
  mount preservation, definition snapshot after workflow changes, callback race,
  64 concurrent claim attempts, reopen/recovery and no-overlap/uncertainty tests.
- `npm run build` and `npm test` passed (five test files); new tests cover Eastern
  date conversion, past-date rejection and DST gaps.
- Deployed with `make host-rebuild-markovd`, final image
  `sha256:b5355b72fd7f650350bfccc0aeaca82e5bf482330f9b8aaee3d5e2965a70a86a`.
- Live server restart preserved one-time and cron schedules. Each dispatched one
  run: `markov-run-cadf3549c380168d639a5d4e` and
  `markov-run-526218bcf3e26975eccf3813`, both completed, no model calls. CLI logs
  and run records preserve test variables. The initial fixture used an invalid
  template prefix, so that assertion was corrected to check the CLI inputs rather
  than claim interpolated output.
- Playwright verified one-time/cron form creation, New York zone label, edit,
  pause/resume and delete; no desktop overflow. Blocking the GET response proves
  the entire form stays disabled until loading finishes (bug recorded and fixed).
- `git diff --check` passed. Vite reports its existing large-bundle warning.
