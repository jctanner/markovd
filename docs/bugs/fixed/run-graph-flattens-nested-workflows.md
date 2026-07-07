# Bug: Run graph flattens nested workflow calls

## Summary

The Run Detail graph renders nested workflow execution as a mostly linear chain
instead of preserving the parent-child workflow structure.

## Evidence

- Screenshot: `/tmp/markovd-graph-bug.png`
- Example run URL in screenshot: `/runs/markov-run-5214ed0e`
- Related completed workflow shape from `var-demos-end-to-end`:
  - `main`
    - `reset_jira`
    - `reset_github`
    - `reset_services`
    - `seed_rfe`
    - `run_pipeline`
      - `rfe_speedrun`
        - `submit`
        - `wait_for_completion`
      - `add_strat_label`
      - `strat_create`
        - `submit`
        - `wait_for_completion`
      - `discover_strat_key`
      - `set_strat_issue`
      - `strat_refine`
        - `submit`
        - `wait_for_completion`
      - `strat_review`
        - `submit`
        - `wait_for_completion`

## Actual Behavior

The graph mixes parent workflow steps and child workflow internals into one
vertical path. For example, `submit` and `wait_for_completion` from the
`run-skill` child workflow appear at the same visual level as parent workflow
steps, and the graph does not make it clear that they belong under the
`rfe_speedrun`, `strat_create`, `strat_refine`, or `strat_review` calls.

## Expected Behavior

The graph should preserve nested workflow calls as hierarchy or grouped
subgraphs. Parent workflow steps should remain visually distinct from the steps
executed inside child workflows, and repeated child workflows should appear
under the parent step that invoked them.

## Development and Test Plan

1. Start a local Markovd stack with `./venv/bin/podman-compose`.
2. Import or create a directory workflow with nested workflow calls comparable
   to `var/demos/end-to-end`.
3. Trigger a run, or seed fixture run/step data if a full workflow run is too
   slow for layout iteration.
4. Open the run detail page with Playwright and capture:
   - a browser snapshot of the graph controls and node labels
   - a full-page screenshot of the graph
5. Inspect the graph data transformation code, especially how parent workflow
   steps, child workflow steps, `fork_id`, and `workflow_name` are converted
   into nodes and edges.
6. Update the graph model so sub-workflow execution is represented as grouped
   hierarchy or as clearly nested lanes under the invoking parent step.
7. Verify with Playwright that:
   - top-level `main` steps remain visually distinct
   - child workflow steps appear under the parent step that invoked them
   - repeated `run-skill` child workflows are not merged into one ambiguous
     vertical chain
   - node labels and statuses remain readable at desktop viewport sizes
8. Run frontend build checks and any focused graph/unit tests available.
9. Record screenshots, commands, and verification results in this bug file
   before moving it to `docs/bugs/fixed/`.

## Impact

The graph is difficult to use for real workflow debugging because execution
lineage is ambiguous. Users can see step names and statuses, but not the
workflow nesting that explains why those steps ran.

## Resolution

Updated `ui/src/components/WorkflowGraph.tsx` so the graph layout keeps each
exact `fork_id` group as its own lane instead of recursively expanding child
workflow steps into the parent chain. Nested workflow calls now render at the
next horizontal level, with dashed child and join edges back to the parent
workflow chain.

The layout still preserves existing `for_each` branch handling. Direct child
workflow groups are rendered as nested lanes, while fan-out branches remain
expanded or summarized according to the existing branch threshold.

## Verification

- Started the local UI dev server against the running API:
  `VITE_API_PROXY=http://markovd.local npm run dev -- --host 127.0.0.1 --port 5173`
- Opened `http://127.0.0.1:5173/runs/markov-run-b09c12f9` with Playwright,
  selected the Graph tab, and accepted the large-run render confirmation.
- Captured screenshot evidence:
  `docs/bugs/fixed/run-graph-nested-workflows-fixed.png`
- Verified React Flow node transforms for the nested workflow:
  - top-level `main` steps render at `x=0`, including `run_pipeline`
  - `run_pipeline` child steps render at `x=340`, including `rfe_speedrun`,
    `strat_create`, `strat_refine`, and `strat_review`
  - repeated child `run-skill` internals render at `x=680`, including distinct
    `submit` and `wait_for_completion` nodes for
    `run_pipeline-rfe_speedrun`, `run_pipeline-strat_create`,
    `run_pipeline-strat_refine`, and `run_pipeline-strat_review`
- Ran `npm run build` in `ui/`.
- Ran `npx eslint src/components/WorkflowGraph.tsx` in `ui/`.

## Status

Fixed
