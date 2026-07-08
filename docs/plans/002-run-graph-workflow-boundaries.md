# Run Graph Workflow Boundaries Plan

## Summary

Use React Flow sub-flow/grouping primitives to draw visual boundaries around
steps that belong to the same Markov workflow invocation. The goal is to make
the Run Detail graph show workflow ownership explicitly, not just through
horizontal lane offsets and fork labels.

## Execution Status

Implemented Phase 1 on 2026-07-08. The Run Detail graph now renders
non-interactive workflow boundary nodes behind nested workflow step nodes. The
implementation keeps the existing absolute lane layout and does not yet convert
step nodes to true React Flow parent-relative sub-flows.

## Motivation

The current Run Detail graph separates nested workflow calls into lanes, but
workflow scope is still implicit. In realistic runs such as
`var/demos/end-to-end`, a user needs to quickly see:

- which steps belong to `main`
- which steps belong to child workflows such as `reset-jira`,
  `reset-github`, `seed-rfe`, and `run-pipeline`
- which `run-skill` steps belong under each parent pipeline step
- where repeated helper workflows start and end

React Flow supports grouped and nested nodes with `type: "group"`,
`parentId`, and `extent: "parent"`. Those primitives can be used to render
workflow scope as labeled boxes around related step nodes.

## Current Behavior

`ui/src/components/WorkflowGraph.tsx` currently builds a flat React Flow node
list from callback-backed step rows. Nested workflow calls are inferred from
`fork_id` path segments and laid out into progressively deeper horizontal
lanes.

This fixed the earlier flattening problem, but the boundary between workflow
scopes remains visual convention instead of an explicit graph element.

## Desired Behavior

The graph should render labeled workflow containers around nested workflow
steps. For example, a run of `var/demos/end-to-end` should show:

- top-level `main` steps outside or inside a root-level `main` boundary
- a `reset-jira` boundary around its reset and verify steps
- a `reset-github` boundary around its bootstrap, repo setup, and import work
- a `run-pipeline` boundary around pipeline phases
- nested `run-pipeline-rfe_speedrun`, `run-pipeline-strat_create`,
  `run-pipeline-strat_refine`, and `run-pipeline-strat_review` boundaries
  around each `run-skill` invocation

The boundary should help users answer "what workflow am I looking at?" without
making the graph harder to scan or click.

## React Flow Approach

React Flow grouping supports:

- group nodes using `type: "group"`
- child ownership using `parentId`
- constraining child movement using `extent: "parent"`
- nested groups through group nodes that also have a `parentId`

The feature should use these primitives where they help, but the first
implementation does not need draggable editing behavior. The run graph is a
read-only visualization.

## Implementation Strategy

### Phase 1: Decorative Workflow Boundary Nodes

Keep the existing absolute lane layout. After step nodes are positioned,
calculate a bounding rectangle for each workflow scope and add a background
group-like node behind the step nodes.

Requirements:

- add boundary nodes for non-root workflow scopes derived from exact `fork_id`
  groups
- label each boundary with a readable workflow path or workflow name
- size each boundary from child node bounds plus padding
- keep boundary nodes non-draggable and non-selectable if practical
- ensure boundary nodes render behind step nodes
- preserve all existing step click behavior and log modal behavior
- preserve existing fork summary behavior for large fan-outs

This phase is lower risk because child step positions remain absolute and
existing edge routing should continue to work with minimal changes.

### Phase 2: True Parent-Child Sub-Flows

If Phase 1 is visually successful, convert workflow scopes to true React Flow
sub-flows:

- set `parentId` on step nodes that belong to a workflow boundary
- convert child node positions from absolute coordinates to parent-relative
  coordinates
- consider `extent: "parent"` only if it improves read-only graph behavior
- support nested workflow groups with parent-relative group positions
- verify edge routing between parent and child groups remains readable

This phase is more invasive because React Flow interprets child node positions
relative to the parent group.

## Data Model Notes

The grouping source should come from Markov execution lineage already available
to the frontend:

- `fork_id` identifies a workflow invocation path
- `workflow_name` identifies the workflow currently executing a step
- `step_name` identifies the parent step that invoked a child workflow when
  combined with the path convention used by Markov

Group IDs should be stable and separate from step node IDs, for example:

```text
group::run-pipeline
group::run-pipeline-rfe_speedrun
group::run-pipeline-strat_create
```

Do not infer groups from display labels alone.

## UX Guidelines

- Boundary styling should be quiet: subtle border, low-contrast background,
  compact label.
- Boundaries should not look like clickable cards if they do not open details.
- Step nodes must remain the primary visual target.
- Long workflow paths should truncate or wrap safely without overlapping step
  nodes.
- Boundaries should not obscure status colors, job names, errors, or selected
  step modal interactions.
- The graph should remain usable on desktop widths where the current graph is
  usable; mobile can preserve the existing pan/zoom behavior.

## Risks

- Boundary boxes can make large runs feel visually cluttered.
- Parent-relative positioning can break existing edge routing if introduced too
  early.
- Large fan-outs may create oversized or overlapping groups.
- React Flow z-index behavior for nested groups and edges may require tuning.
- Group labels may become noisy for deeply nested helper workflows.

## Verification Plan

Use the existing completed nested workflow run as a regression target:

- run URL: `/runs/markov-run-b09c12f9`
- workflow shape: `var/demos/end-to-end`

Verify with Playwright:

- graph renders without console errors
- top-level `main` lane remains readable
- `run-pipeline` child steps are visually inside a `run-pipeline` boundary
- repeated `run-skill` invocations each have distinct boundaries
- step clicks still open `StepDetailModal`
- job log sections still appear for steps with `output_json.job_name`
- screenshot evidence shows boundaries without text overlap

Run frontend checks:

```bash
cd ui
npm run build
npx eslint src/components/WorkflowGraph.tsx
```

## Verification Results

Verified on 2026-07-08 against the local dev UI at
`http://127.0.0.1:5173`.

- `markov-run-b09c12f9` rendered 9 workflow boundary nodes:
  `reset_jira`, `reset_github`, `reset_services`, `seed_rfe`,
  `run_pipeline`, and four nested `run_pipeline-*` `run-skill` scopes.
- Playwright containment checks confirmed representative child steps render
  inside their expected boundaries:
  - `rfe_speedrun` and `strat_create` inside `run_pipeline`
  - `submit` inside `run_pipeline-rfe_speedrun`
  - `wait_for_completion` inside `run_pipeline-strat_create`
  - `process_repos` inside `reset_github`
- Step click behavior still opens `StepDetailModal`.
- `markov-run-a25ff450` verified that a graph step with top-level
  `output_json.job_name` still opens a details modal with a Logs section.
- Screenshot evidence:
  `docs/plans/run-graph-workflow-boundaries.png`
- Browser verification reported 0 console errors.
- `npm run build` passed in `ui/`.
- `npx eslint src/components/WorkflowGraph.tsx` passed in `ui/`.

## Open Questions

- Should `main` get a visible root boundary, or should only nested workflows be
  grouped?
- Should group labels use `fork_id`, `workflow_name`, or both?
- Should collapsed fan-out summaries live inside the parent workflow boundary
  or get their own fan-out boundary?
- Should clicking a boundary filter/highlight steps in that workflow scope?

## Non-Goals

- Do not change backend callback processing.
- Do not change Markov workflow semantics.
- Do not add graph editing or drag-and-drop behavior.
- Do not replace the existing Gantt or table views.
- Do not solve run graph memory growth for very large runs in this plan.
