# Bug: Run graph workflow boundary labels overlap fan-out branches

## Summary

The Run Detail graph workflow boundary containers help show workflow scope, but
their labels and nested boxes become noisy or overlap in fan-out and nested
workflow regions.

## Evidence

Screenshots:

- `docs/screenshots/markovd-ui-bug-1.png`
- `docs/screenshots/markovd-ui-bug-2.png`
- `docs/screenshots/markovd-ui-bug-3.png`

Fast reproducer:

- `docs/fixtures/workflows/graph-boundary-noop/`

This fixture is a complete directory workflow with shell no-op steps. It avoids
the slow `ai-first-pipeline` end-to-end dependencies while preserving the graph
shape that triggers the UI issues:

- a sequential parent workflow
- a `fanout-parent` workflow with long `for_each_key` branch names
- an `import-repo` branch workflow with stacked no-op steps
- a `seed-fixture` child workflow for edge/header crowding
- a `pipeline` workflow with repeated nested `run-skill` calls

Observed issues:

- In expanded fan-out branches, multiple long boundary labels render along the
  same horizontal band and visually collide.
- Nested branch boundary boxes sit very close together, making it difficult to
  distinguish branch ownership.
- Labels such as
  `reset_github-process_repos-agent-eval-harn...` are too long for the
  available group header area.
- Parent workflow containers can become very large when they include nested
  child workflow boxes, creating large empty regions and making scope harder to
  scan.
- The path label and workflow name label are adjacent with little visual
  separation, for example `run_pipelinerun-pipeline` in screenshots.

## Expected Behavior

Workflow boundaries should clarify scope without creating label collisions or
large confusing empty containers. Fan-out branches should remain readable when
expanded, and long workflow paths should truncate or be summarized cleanly.

## Possible Fixes

- Render shorter labels for fan-out branch boundaries, such as the branch key
  rather than the full `fork_id` path.
- Hide or de-emphasize labels for deeply nested branch groups unless selected
  or hovered.
- Add stronger label separators between `fork_id` path and `workflow_name`.
- Consider not drawing decorative boundaries around expanded fan-out branch
  internals when a fork summary is not collapsed.
- Cap or tune parent boundary sizing so nested child workflow boxes do not
  create excessive blank space.
- Revisit Phase 2 true React Flow sub-flow parenting only after label density is
  under control.

## Resolution

2026-07-08:

- Expanded `for_each` branch paths are no longer rendered as workflow boundary
  containers. The branch steps remain visible as normal expanded fork columns,
  which removes the long branch boundary labels from the crowded fan-out band.
- Workflow boundary labels now render the branch key and workflow name with a
  visible `/` separator instead of adjacent text.
- Workflow boundary top and bottom padding were increased so labels have more
  room away from child step cards and connection arrows.
- Added `docs/fixtures/workflows/graph-boundary-noop/` as the fast no-op
  reproducer for this class of graph bugs.

Resolution screenshots:

- `docs/bugs/fixed/run-graph-boundary-labels-fixed-fitview.png`
- `docs/bugs/fixed/run-graph-boundary-labels-fixed-fullscreen.png`
- `docs/bugs/fixed/run-graph-boundary-labels-fixed-graph.png`
- `docs/bugs/fixed/run-graph-boundary-labels-fixed-zoom.png`

## Verification

2026-07-08:

- `../markov/bin/markov validate docs/fixtures/workflows/graph-boundary-noop`
  returned `valid`.
- `../markov/bin/markov run docs/fixtures/workflows/graph-boundary-noop --state-store /tmp/markovd-graph-boundary-noop-2.db --run-id graph-boundary-noop-smoke-2 --verbose`
  completed successfully.
- Imported the fixture into the running `markovd.local` instance as workflow
  `graph-boundary-noop`.
- Triggered `markov-run-a8dc339b`; the API reported `status: completed` with
  34 steps.
- Playwright rendered
  `http://127.0.0.1:5173/runs/markov-run-a8dc339b` on the Graph tab and found
  eight workflow boundary labels:
  `setup`, `fanout_parent / fanout-parent`, `seed_fixture / seed-fixture`,
  `pipeline`, `rfe_speedrun / run-skill`, `rfe_submit / run-skill`,
  `strat_create / run-skill`, and `strat_refine / run-skill`.
- Playwright found no workflow boundary label bounding-box overlaps and no
  browser console errors while rendering the fixed graph.
- `npm run build` in `ui/` completed successfully.
- `npx eslint src/components/WorkflowGraph.tsx` in `ui/` completed
  successfully.

## Status

Fixed
