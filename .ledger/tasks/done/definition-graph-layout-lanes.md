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
