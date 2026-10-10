# ADR-0006: Keep Call and Return Edges Between Adjacent Columns, Routed in Lanes

## Status

Accepted. Amends the "Invocation Endpoints" section of
[ADR-0005](ADR-0005-expand-definition-diagrams-by-call-site.md).

## Context

ADR-0005 made an invocation's exit its last step's *effective* exit: when the
last step calls a workflow, the exit is that child's exit, recursively. In a
chain of workflows that each end in a call (Breadboard's strat-workflow:
`rounds` → `round` → `test-variants` → `run-test`), the return to `summarize`
started four columns away. It crossed the gaps and groups in between,
including the `wipe` group's cards.

Call and return edges were React Flow `smoothstep` edges. They put the
vertical segment midway between the two columns, so every edge crossing one
gap used the same x, and edges with overlapping vertical extents lay on top
of each other.

## Decision

1. **An invocation's exit is its own last step.** Effective exits are no
   longer propagated.
2. **Every call returns, in the adjacent gap.** The return goes to the next
   step of the caller, as before; when the call is the caller's last step, it
   goes to the calling step itself. That step completes when its child does,
   and its own workflow's return then leaves from it. No call or return edge
   spans more than one gap.
3. **Lanes.** The server gives each call and return edge a lane, the x of its
   vertical segment, within the gap it crosses (`data.laneX`, edge type
   `lane`):
   - interval partitioning means no two edges whose vertical extents overlap
     share a lane;
   - lane order in a gap is chosen to minimise crossings (exhaustively, up to
     seven lanes);
   - a gap widens beyond the 80px minimum to fit its lanes.
4. **Separate handles.** Calls meet a card 25px from its top and returns at
   47px (35% and 65% of the 72px card the layout assumes). A step that is
   both a caller and a return target doesn't merge the two lines, and taller
   cards don't shift the handles away from the routing.
5. **Alignment.** A child group is placed so that its first step is level
   with its caller, which makes the call a straight line, unless an earlier
   group in the same column is in the way.
6. **Spacing.** The vertical gap between steps went from 60px to 36px.

## Consequences

- The meaning of ADR-0005's edges is unchanged: calls enter the child, and
  returns join the caller's sequence.
- A deep chain now reads as one return per level instead of one long edge.
- Crossings can remain where they can't be avoided; overlapping segments
  can't.
- Layout logic lives in `internal/api/diagram.go` (`assignEdgeLanes`,
  `bestLaneOrder`). The UI only draws the path through `laneX`
  (`LaneEdge`).

## Addendum (2026-10-10): templated calls

Markov resolves a templated sub-workflow name (`submit-{{ test.arm }}`) when
the step runs. The diagram draws its possible targets as alternatives:

- **Which targets:**
  - the step's `workflow_names` (Markov `2caa232`) when it has them;
  - otherwise the workflows whose names fit the template, with each
    `{{ }}`/`{% %}` read as a wildcard;
  - a template with no fixed text fits nothing, rather than every workflow,
    and the step continues by sequence as before.
- **Layout:** each target is a group in the next column, stacked one above
  the other. The call and return edges are marked `data.alternative` and
  drawn fainter, because only one target runs. They are routed in lanes like
  any other edge.
- **Labels:**
  - the step node lists its candidates (`workflowNames`) and where they came
    from (`candidatesFrom`: `workflow_names` or `pattern`);
  - each target group carries `alternativeOf` and shows "one possible
    target" in its header.
