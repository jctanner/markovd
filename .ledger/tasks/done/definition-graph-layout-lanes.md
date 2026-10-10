# Task: Definition graph layout: adjacent returns, edge lanes, alignment, spacing

## Status

Done (2026-10-10)

## Summary

Problems seen on Breadboard's `var-benchmarks-strat-workflow` page:

- call and return lines overlapped in each gap between columns;
- a deep return crossed four columns and the `wipe` group;
- fit-to-view zoomed out to 21%.

Changes follow [ADR-0006](../../decisions/ADR-0006-adjacent-column-returns-and-edge-lanes.md).

## Verification

- `go test ./...` passes.
  `TestGenerateDiagramPropagatesFinalNestedCallExit` is replaced by
  `TestGenerateDiagramReturnsStayBetweenAdjacentColumns`. New tests on a
  benchmark-shaped diagram:
  - `TestDiagramLanesKeepEdgesApart`: every call and return edge is a
    `lane` edge, its lane lies inside its gap, and no two share a vertical
    segment;
  - `TestDiagramAlignsChildEntryWithCaller`: entries line up with callers
    unless blocked, and no groups overlap in a column.
- `ui`: `tsc --noEmit`, `npm test` and `npm run build` pass.
- Before and after screenshots of the deployed page are in the session
  notes. See Breadboard's plan observations, 2026-10-10.

## Follow-up: templated calls drawn as alternatives

See the addendum in ADR-0006. Tests:
- `TestTemplatedCallExpandsPatternMatchesAsAlternatives` (pattern matching,
  alternative and lane data on edges, non-matching workflows left out);
- `TestTemplatedCallUsesWorkflowNames`;
- `TestFullyDynamicTemplateWithoutNamesIsUnresolved`.

They replace `TestGenerateDiagramLeavesTemplatedWorkflowUnexpanded`.

## Follow-up: the run graph's for_each branches overlapped

The run graph (`WorkflowGraph.tsx`, laid out in the browser) had its own
problems:

- **Fixed spacing:** it placed a `for_each`'s branches `FORK_GAP_X` (280px)
  apart around the parent, ignoring how wide each branch's own fan-out or
  sub-workflow boxes were. With nested fan-outs (strat-workflow's rounds →
  tests → variants), round 0's S2 column was drawn exactly on round 1's S1
  column.
- **Too close:** cards are 294px wide, so even plain neighbouring branches
  overlapped by 14px.

`runGraphLayout.ts` now measures each chain's left and right extent first
(sub-workflow chains with their box padding, nested fan-outs, collapsed
fan-outs adding nothing). It places sibling branches side by side by those
widths, 40px apart, centred under the parent.

Tests (`runGraphLayout.test.ts`):
- the benchmark's shape has no shared columns;
- a branch with its sub-workflow box fits between its neighbours;
- branches are centred, and plain neighbours no longer overlap;
- collapsed fan-outs add no width.

`LARGE_RUN_THRESHOLD`, which opened runs on Logs and asked before drawing
the graph, went from 25 steps to 500; a single benchmark run already has
46.
