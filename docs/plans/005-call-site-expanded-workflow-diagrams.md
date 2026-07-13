# Call-Site-Expanded Workflow Diagram Plan

## Goal

Make workflow definition diagrams represent sub-workflow execution topology by
rendering a separate child group for every call site and an explicit return/join
from each child invocation into the caller's next step.

## Execution Status

Implemented on 2026-07-12. Definition diagrams now expand reusable workflows by
call site, calculate invocation entry/exit topology, and render semantic call
and return/join edges with recursion and graph-growth safeguards.

The governing behavior is defined in
[ADR-0005](../decisions/ADR-0005-expand-definition-diagrams-by-call-site.md).

## Current Behavior

`internal/api/diagram.go` collects reachable workflows with a global
`rendered[workflowName]` set. Placements are later stored in a map keyed by
workflow name. Consequently:

- a reusable workflow is rendered once regardless of call count
- every call edge to that workflow converges on the same group
- the caller retains a direct edge to its next step
- the diagram has no child-exit-to-caller return edge

In `var/demos/end-to-end`, five `run-skill` call sites converge on one group.
The latest completed run, `markov-run-cc8a1a27`, shows separate invocation
boundaries for each call. It also shows runtime-only fan-out: five
`seed_projects` branches, eight `process_repos` branches, four `run_codegen`
branches, and zero `run_investigations` branches.

## Desired Behavior

For each reachable call site, the definition diagram should show:

- the caller's sub-workflow step
- a unique child workflow group for that call site
- a `call` edge from the caller step to the child entry
- a `return` edge from the child exit to the caller's next step
- no direct caller-step-to-next-step edge that bypasses the child

Repeated references to one definition become repeated invocation groups.
`for_each` calls still render one child template because runtime cardinality is
unknown; the return edge represents the all-branches join.

## API Model

Extend diagram edges with semantic relation metadata:

```json
{
  "id": "...",
  "source": "...",
  "target": "...",
  "type": "smoothstep",
  "relation": "return"
}
```

Supported relations:

- `sequence`
- `call`
- `return`

Add sufficient node metadata for:

- invocation path and caller step
- workflow definition name
- synthetic empty-workflow pass-through nodes
- synthetic recursive-reference nodes

Keep new JSON fields additive so older clients can continue rendering nodes and
edges with their existing defaults.

## Implementation Phases

### Phase 1: Invocation Tree

- Replace workflow-name deduplication with recursive call-site collection.
- Assign every invocation a stable path and deterministic ID namespace.
- Track caller invocation, caller step index, definition name, and depth.
- Escape internal ID delimiters and add occurrence suffixes for duplicate step
  names.
- Stop recursive expansion when the target definition is already in the current
  ancestry and create a recursive-reference leaf.
- Enforce deterministic invocation and node budgets with useful errors.

### Phase 2: Entry, Exit, And Edges

- Create all step and synthetic nodes for each invocation.
- Calculate invocation entry and effective exit nodes bottom-up.
- Emit ordinary `sequence` edges only across non-call steps.
- Emit `call` edges into each child invocation.
- Emit `return` edges from child exits to following caller steps.
- Propagate a final child exit as the parent exit when a call is the last step.
- Treat `for_each` child exits as static all-branches joins without expanding
  runtime items.

### Phase 3: Layout

- Measure each invocation subtree before assigning final coordinates.
- Align a child group with its caller step where practical.
- Reserve vertical space for each cloned subtree so sibling invocations do not
  overlap.
- Route return edges so their direction back to the caller is legible and does
  not obscure sequence edges or group labels.
- Keep stable dimensions for step, group, recursive-reference, and pass-through
  nodes.

### Phase 4: Frontend Rendering

- Add the edge relation to `ui/src/api.ts`.
- Map `sequence`, `call`, and `return` to distinct, theme-compatible styles in
  `WorkflowStructureGraph.tsx`.
- Use arrow markers and routing that make call direction and join direction
  readable at normal and fullscreen sizes.
- Render recursive-reference and empty-workflow nodes as compact structural
  nodes, not ordinary executable steps.
- Preserve fullscreen, fit-view, minimap, zoom, and jump controls.

### Phase 5: Verification

- Add backend topology tests that assert exact node and edge relations rather
  than only node counts.
- Cover a repeated child definition invoked by multiple caller steps.
- Cover nested calls, a call followed by another step, and a call as the final
  step.
- Cover `for_each` calls, empty workflows, unresolved references, direct and
  indirect recursion, duplicate step names, and expansion-budget failures.
- Preserve file and directory definition diagram coverage.
- Inspect `var/demos/end-to-end` in Playwright at desktop and mobile widths and
  in fullscreen.
- Compare call-site groups against the latest completed run without expecting
  the static diagram to reproduce runtime branch cardinality or conditions.
- Check screenshots and node bounding boxes for overlap, clipped labels,
  unreadable return edges, and blank/incorrectly framed canvases.

## Acceptance Criteria

- [x] Every reachable sub-workflow call site has its own rendered child group.
- [x] Repeated references to one workflow definition do not share a rendered
      invocation group.
- [x] Call steps connect to their child entry with a semantic `call` edge.
- [x] Child exits connect to the next caller step with a semantic `return` edge.
- [x] No sequence edge bypasses a rendered sub-workflow invocation.
- [x] A final sub-workflow call's child exit becomes the parent invocation exit.
- [x] `for_each` calls retain one static child template and an explicit
      all-branches join without inventing runtime cardinality.
- [x] Empty workflows and recursive calls remain connected through explicit
      synthetic nodes.
- [x] Expansion limits fail deterministically with a useful error instead of
      returning a partial graph.
- [x] Node and edge IDs are stable and collision-free for duplicate names and
      repeated calls.
- [x] Existing file and directory workflow diagrams remain supported.
- [x] Backend tests assert topology and relation metadata for all critical
      cases.
- [x] React Flow renders call and return direction clearly in normal and
      fullscreen layouts without incoherent overlap.
- [x] `var/demos/end-to-end` displays separate groups for all repeated
      `run-skill` call sites and remains usable at supported viewport sizes.

## Files Likely Involved

- `internal/api/diagram.go`
- `internal/api/diagram_test.go`
- `ui/src/api.ts`
- `ui/src/components/WorkflowStructureGraph.tsx`
- `ui/src/index.css`

## Risks And Mitigations

Graph growth:

Call-site expansion duplicates reusable definitions. Enforce a deterministic
budget, keep runtime fan-out collapsed to one template, and return a clear error
when a definition is too large to render safely.

Edge readability:

Return edges travel from deeper columns back into caller flow. Measure subtree
layout first, reserve routing space, and validate screenshots and bounding boxes
on realistic nested workflows.

Recursive definitions:

Ancestry-aware expansion terminates recursion at a synthetic reference node.
The diagram stays connected without claiming that recursive runtime depth is
known statically.

Compatibility:

Add relation and invocation metadata without removing existing fields. Keep
frontend defaults for responses that omit the new relation.

## Deliverables

- Call-site invocation-tree diagram generator.
- Explicit semantic call and return/join edges.
- Recursion and expansion safeguards.
- Updated React Flow edge and synthetic-node rendering.
- Backend topology tests and Playwright verification evidence.
