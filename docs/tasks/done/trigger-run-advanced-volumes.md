# Task: Collapse Trigger Run Volume Selectors

## Goal

Move the PVC volume and secret volume selectors on the Trigger Run page behind a
collapsed Advanced section so the default form is shorter.

## Acceptance Criteria

- [x] Trigger Run hides PVC and secret volume lists by default.
- [x] Users can expand Advanced to select PVC and secret volume mounts.
- [x] Existing selected defaults and submitted run payloads keep working.
- [x] Verification is recorded before moving this task to done.

## Files Likely Involved

- `ui/src/pages/TriggerRun.tsx`
- `ui/src/index.css`

## Verification

- `npm run build` passed in `ui/`.
- `npx eslint src/pages/TriggerRun.tsx` passed in `ui/`.
- `npm run lint` was also run in `ui/`; it still fails on pre-existing
  unrelated lint errors in files such as `src/auth.tsx`,
  `src/components/GanttChart.tsx`, and `src/pages/RunDetail.tsx`.

## Status

Done
