# Task: Improve Workflow Definition Edge Readability

## Goal

Make sequence, call, and return connections in large workflow definition
diagrams understandable without requiring users to manually trace overlapping
lines across the entire graph.

## Context

The call-site-expanded diagram introduced by ADR-0005 represents workflow
topology accurately, but realistic definitions can produce many long call and
return connections. In `var/demos/end-to-end`, fit view renders 99 nodes and 74
edges at approximately 0.096 zoom. At that scale, edge arrowheads and relation
styles are difficult to distinguish. After zooming and panning, edges may enter
from outside the viewport without enough context to identify their source.

The current purple, green, and gray styles distinguish call, return, and
sequence edges in code, but the diagram does not explain that visual language
or help a user isolate one invocation path.

## Acceptance Criteria

- [x] Add a compact, accessible legend for `sequence`, `call`, and `return`
      connections without obscuring graph content or controls.
- [x] Expose the relation and connected endpoints for an edge through a label,
      tooltip, selection detail, or similarly discoverable interaction.
- [x] Selecting or focusing a node or edge emphasizes its complete relevant
      caller-to-sub-workflow-to-return path and visually de-emphasizes unrelated
      edges.
- [x] Provide a clear way to restore the unfiltered graph after path emphasis.
- [x] Route call and return connections through predictable lanes or otherwise
      reduce overlapping parallel and crossing paths between invocation groups.
- [x] Make return-to-caller-next-step semantics visually distinguishable from
      an ordinary forward sequence connection.
- [x] Preserve usable pan, wheel zoom, control zoom, fit-view, minimap, jump,
      and fullscreen behavior.
- [x] Preserve keyboard accessibility and expose selection state and controls
      with appropriate accessible names and state.
- [x] Keep edge arrowheads and emphasized paths perceivable in both light and
      dark themes.
- [x] Avoid node, group, legend, minimap, and control overlap at supported
      desktop and mobile viewport sizes.
- [x] Add focused automated coverage for any path-selection and edge-relation
      logic extracted from the component.
- [x] Verify the deployed `var/demos/end-to-end` diagram with Playwright at fit
      view and practical working zoom levels, including zooming, panning,
      selecting a repeated `run-skill` invocation, tracing its return, clearing
      selection, and entering fullscreen.

## Design Questions

- Whether edge semantics should be shown persistently, on hover, on selection,
  or through a combination of those states.
- Whether invocation groups should also become collapsible to improve the
  overview, or whether path emphasis and improved routing are sufficient.
- Whether lane assignment belongs in the backend layout output or can remain a
  deterministic frontend presentation concern.

## Files Likely Involved

- `internal/api/diagram.go`
- `internal/api/diagram_test.go`
- `ui/src/components/WorkflowStructureGraph.tsx`
- `ui/src/index.css`
- Frontend test files or test configuration, if added

## Status

Done

## Verification Baseline

Observed with Playwright on 2026-07-13:

- The deployed `var/demos/end-to-end` diagram rendered 99 nodes and 74 edges.
- Fit view used `scale(0.0960826)`, where arrowheads and relation colors were
  difficult to interpret.
- Wheel zoom and pointer-drag panning worked; a working view at approximately
  `scale(0.669158)` made individual nodes readable.
- At working zoom, only part of a long call/return route was visible and several
  connections entered from outside the viewport, leaving their origin unclear.
- Call, return, and sequence paths were purple, green, and gray respectively,
  but no visible legend explained those meanings.

## Implementation

- Added relation-specific backend handle metadata and matching React Flow side
  handles. Calls use an outbound lane and returns use a separate inbound lane;
  sequence edges remain vertical.
- Added a persistent semantic legend and accessible edge labels. Selecting an
  edge shows its relation and endpoint labels.
- Added invocation-path focus resolution. Node or edge selection emphasizes
  the selected invocation subtree plus its caller and return boundary while
  dimming unrelated nodes and edges.
- Added pointer and keyboard selection, a live selection summary, pane clearing,
  and an explicit clear-selection icon control.
- Kept invocation groups expanded. Playwright verification showed focused-path
  isolation was sufficient without adding another graph state.
- Kept semantic handle assignment in the backend diagram contract and added a
  frontend relation-derived fallback for rolling upgrades.

## Verification

Performed on 2026-07-13:

- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `env GOCACHE=/tmp/go-build-cache go vet ./internal/api`
- `cd ui && npm test`
- `cd ui && npm run build`
- `cd ui && npx eslint src/api.ts src/components/WorkflowStructureGraph.tsx src/components/workflowStructureFocus.ts src/components/workflowStructureFocus.test.ts`
- `git diff --check`
- Backend coverage verifies deterministic sequence, call, and return handle
  assignment. Frontend coverage verifies invocation subtree focus, caller and
  return boundaries, repeated-call isolation, and unresolved selection.
- Playwright rendered the 99-node, 74-edge `var/demos/end-to-end` definition
  with separate outbound-call and inbound-return lanes and a non-overlapping
  semantic legend.
- Selecting a repeated `run-skill` invocation emphasized five nodes and its
  three-edge call/sequence/return path while dimming 94 nodes and 71 edges.
- Pointer edge selection exposed relation and endpoint labels. Keyboard Enter
  and Space activated node and edge paths, and the clear control restored the
  complete graph.
- Wheel zoom, pointer-drag pan, jump-to-bottom, fit view, minimap, normal view,
  and fullscreen remained functional.
- Dark and light themes rendered distinct semantic strokes and matching arrow
  markers. Desktop and `390x844` checks found no graph-local node, group,
  legend, minimap, or control overlap. The previously recorded global mobile
  navigation overflow remains outside this task's graph boundary.
