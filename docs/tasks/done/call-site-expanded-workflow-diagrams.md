# Task: Expand Workflow Diagrams by Call Site

## Goal

Render each sub-workflow call as a distinct invocation group and show explicit
call and return/join flow in workflow definition diagrams.

## Context

The current generator deduplicates groups by workflow definition name. Repeated
calls therefore converge on one group, and a direct caller sequence edge
bypasses child completion.

The required behavior is defined in:

- [ADR-0005](../../decisions/ADR-0005-expand-definition-diagrams-by-call-site.md)
- [Plan 005](../../plans/005-call-site-expanded-workflow-diagrams.md)

## Acceptance Criteria

- [x] Build a stable, call-site-based invocation tree.
- [x] Render a distinct child group for every reachable call site.
- [x] Emit semantic `sequence`, `call`, and `return` edges.
- [x] Do not emit sequence edges that bypass child invocations.
- [x] Propagate child exits through final-step calls.
- [x] Keep `for_each` as one static child template with an explicit join.
- [x] Render connected synthetic nodes for empty workflows and recursion.
- [x] Enforce deterministic graph expansion limits with useful errors.
- [x] Keep IDs deterministic and collision-free for repeated and duplicate step
      names.
- [x] Preserve file and directory definition support.
- [x] Add exact backend topology coverage for critical edge cases.
- [x] Render edge relations and synthetic nodes clearly in React Flow.
- [x] Verify realistic normal, fullscreen, desktop, and mobile layouts without
      incoherent overlap or blank canvases.

## Files Likely Involved

- `internal/api/diagram.go`
- `internal/api/diagram_test.go`
- `ui/src/api.ts`
- `ui/src/components/WorkflowStructureGraph.tsx`
- `ui/src/index.css`

## Status

Done

## Notes

Runtime branch cardinality remains the responsibility of the run graph. The
definition diagram represents one static template for a `for_each` invocation.

Implemented:

- Replaced workflow-name deduplication with a measured invocation tree keyed by
  escaped call-site paths.
- Added bottom-up entry/exit propagation and semantic `sequence`, `call`, and
  `return` edges.
- Added connected empty-workflow and recursive-reference nodes, unresolved
  reference errors, and deterministic invocation/node limits.
- Added distinct React Flow styling and arrows for call and return edges,
  caller context on repeated groups, and synthetic-node rendering.
- Increased group header space after Playwright identified caller subtitles
  touching the first child node.

Verification performed on 2026-07-12:

- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `env GOCACHE=/tmp/go-build-cache go vet ./internal/api`
- `cd ui && npm test`
- `cd ui && npm run build`
- `cd ui && npx eslint src/api.ts src/components/WorkflowStructureGraph.tsx`
- `git diff --check`
- Backend tests assert repeated call-site groups, call/return relations, no
  bypass edges, nested final exits, `for_each` joins, file and directory input,
  empty and direct/indirect recursive workflows, unresolved references,
  delimiter-safe duplicate IDs, and both expansion limits.
- The current 27-file `var/demos/end-to-end` definition generated 99 nodes and
  74 edges, including 24 groups and 8 separate `run-skill` groups.
- Playwright confirmed all 74 edge paths were nonzero; call, return, and
  sequence edges had distinct colors; 99 nodes and 24 groups had no peer
  overlaps or clipped labels; and all 23 caller subtitles cleared their first
  child nodes.
- Normal and fullscreen views were nonblank and correctly framed. Zoomed views
  showed readable call and return direction. At `390x844`, the 342px graph had
  no local overflow or control overlap. The separate global navigation overflow
  is recorded in `docs/bugs/open/mobile-navigation-horizontal-overflow.md`.
- A synthetic recursion fixture rendered its reference node, caller subtitle,
  and nonzero purple call/green return paths.
