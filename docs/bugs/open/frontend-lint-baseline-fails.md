# Bug: Frontend Lint Baseline Fails

## Summary

Running `npm run lint` in `ui/` reports existing React Hooks and TypeScript
lint failures across multiple frontend modules.

## Reproduction

1. Change to `ui/`.
2. Run `npm run lint`.
3. Observe 18 errors and one warning.

## Expected

The repository-wide frontend lint command exits successfully so it can serve as
a regression gate for UI changes.

## Actual

ESLint reports failures in existing files including `auth.tsx`,
`GanttChart.tsx`, `RerunModal.tsx`, `RunLogs.tsx`, `StepDetailModal.tsx`,
`Jobs.tsx`, `Projects.tsx`, `RunDetail.tsx`, `Runs.tsx`, and `Workflows.tsx`.
The failures include synchronous state changes in effects, render purity,
immutability, and constructing JSX inside `try`/`catch`.

## Impact

Medium. Repository-wide lint cannot currently distinguish regressions in a
focused UI change from the existing baseline. Changed files must be linted
directly until the baseline is repaired.

## Evidence

Observed on 2026-07-12 with ESLint 10.2.1:

- `npm run lint`: failed with 18 errors and one warning.
- `npx eslint src/components/WorkflowGraph.tsx src/components/workflowGraphFollow.ts src/components/workflowGraphFollow.test.ts`:
  passed for the run-graph follow-mode changes.

## Related Tasks

- `docs/tasks/done/run-graph-follow-running-step.md`
