# Task: Implement Continuous Follow Mode for the Run Graph

## Goal

Add an opt-in React Flow control that continually centers the run graph on the
latest running task as workflow execution progresses.

## Context

The existing downward-arrow control in `WorkflowGraph.tsx` performs a one-time
jump to the lowest rendered node. Run data is already refreshed by polling in
`RunDetail.tsx`, so follow behavior can react to updated step data without a
backend change.

The behavior and target-selection rules are defined by
[ADR-0003](../../decisions/ADR-0003-follow-running-workflow-step.md).

## Acceptance Criteria

- [x] Add a separate follow-mode toggle adjacent to the existing jump control.
- [x] Give the control a distinct icon, tooltip, accessible label, and visible
      enabled/pressed state.
- [x] When enabled, center the viewport when the resolved running target
      changes as refreshed step data arrives.
- [x] Select the running step with the newest `updated_at`, using a stable
      deterministic tie-breaker.
- [x] Preserve the current zoom level while following.
- [x] Do not recenter for polling updates when the resolved target is unchanged.
- [x] Leave the viewport unchanged when no step is running.
- [x] Focus the appropriate summary node when the selected running step is
      represented by a collapsed fork.
- [x] Keep the existing one-time jump-to-bottom behavior unchanged.
- [x] Add focused automated coverage for target selection and follow-mode state,
      or document why the current frontend test setup cannot support it.
- [x] Verify the control and viewport behavior in both normal and fullscreen
      graph layouts.

## Files Likely Involved

- `ui/src/components/WorkflowGraph.tsx`
- `ui/src/index.css`
- Frontend test files or test configuration, if added

## Status

Done

## Notes

No backend or database changes are expected. Manual panning does not
automatically disable follow mode under ADR-0003; the active toggle state must
make subsequent automatic movement clear to the user.

Implemented:

- Added a pure latest-running-step resolver with `updated_at` ordering and step
  ID tie-breaking.
- Added a graph-build mapping from every step ID to its visible step node or
  collapsed fork summary.
- Added an accessible pressed-state follow control that retains current zoom
  and remembers the last followed node.
- Added Node-native unit tests without introducing a new test dependency.

Verification performed on 2026-07-12:

- `npm test` passed the follow-target test suite, covering newest update,
  deterministic ties, no running target, and collapsed-summary resolution.
- `npm run build` passed TypeScript and the Vite production build.
- `npx eslint src/components/WorkflowGraph.tsx src/components/workflowGraphFollow.ts src/components/workflowGraphFollow.test.ts`
  passed. Repository-wide `npm run lint` remains blocked by the pre-existing
  failures recorded in `docs/bugs/open/frontend-lint-baseline-fails.md`.
- `git diff --check` passed.
- Playwright fixture verification confirmed that enabling follow changed the
  viewport transform while preserving zoom (`scale(1.13469)`), and a subsequent
  newer step moved focus from an ordinary step to a collapsed fork summary.
- The collapsed summary center was within 20 pixels horizontally and 5 pixels
  vertically of the graph center after following.
- Playwright confirmed an unchanged polling result did not alter the viewport,
  disabling follow set `aria-pressed="false"` and prevented movement, and the
  control remained active and non-overlapping in fullscreen.
