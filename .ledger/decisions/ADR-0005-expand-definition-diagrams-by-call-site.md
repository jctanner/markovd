# ADR-0005: Expand Definition Diagrams by Sub-Workflow Call Site

## Status

Accepted

## Context

Workflow definition diagrams show workflow groups, sequential steps, and dotted
edges from sub-workflow steps to the referenced workflow. The generator
currently renders each workflow definition at most once. It uses workflow name
as both the deduplication key and the placement lookup key.

That representation is compact, but it conflates a reusable definition with an
invocation. In `var/demos/end-to-end`, five distinct steps (`rfe_speedrun`,
`rfe_submit`, `strat_create`, `strat_refine`, and `strat_review`) invoke
`run-skill`. The definition diagram points all five call sites at one shared
`run-skill` group. The run graph correctly shows five separate invocation
boundaries.

The current edge model also retains the caller's direct sequential edge while
adding a call edge into the child. For a caller sequence `A, B, C`, where `B`
invokes a child, the diagram contains both `B -> C` and `B -> child entry`. It
does not show that the child must finish before `C` begins.

The definition diagram should remain a static execution template. Runtime
cardinality for `for_each`, condition outcomes, skipped steps, and timing cannot
be known from the definition and belong in the run graph.

## Decision

Render one workflow group per reachable sub-workflow call site, not one group
per workflow definition name.

Each rendered group represents a workflow invocation template and has a stable
invocation path derived from its parent invocation path and caller step. The
root invocation starts with the selected entrypoint. Repeated references to the
same workflow definition therefore produce distinct groups with distinct node
and edge IDs.

For example:

```text
main/run_pipeline/rfe/rfe_speedrun/run-skill
main/run_pipeline/rfe/rfe_submit/run-skill
main/run_pipeline/strategy/strat_create/run-skill
```

The visible group label remains the workflow definition name. The call-site
step and invocation path provide the surrounding identity; the UI may include
the caller step in the group subtitle when needed to distinguish repeated
groups.

### Invocation Endpoints

The generator will calculate an entry and exit node for every invocation:

- Entry is the first step node in the invocation.
- Exit is the last step's effective exit.
- For an ordinary last step, effective exit is that step node.
- When the last step invokes a sub-workflow, effective exit is the child
  invocation's effective exit.
- Empty workflows require an explicit synthetic pass-through node so call and
  return edges remain well-defined.

### Edge Semantics

Diagram edges will carry a semantic relation separate from the React Flow
routing type:

- `sequence`: ordinary progression within one workflow invocation.
- `call`: progression from a sub-workflow step to that call site's child entry.
- `return`: progression from the child exit to the next step in the caller.

For a sub-workflow call that has a following caller step:

```text
caller step --call--> child entry
child exit --return/join--> next caller step
```

The generator will not also emit a direct sequence edge from the caller step to
the next caller step. For a sub-workflow call that is the caller's final step,
the child's exit becomes the caller invocation's exit and participates in the
parent's return edge.

For a `for_each` sub-workflow call, the child group remains one static template.
Its `return` edge represents the join after all runtime branches finish; the
definition diagram does not duplicate a branch count that is only known at run
time.

The frontend will render `call` and `return` edges distinctly but consistently:
call edges enter child groups, while return edges visually communicate a join
back into the caller sequence. Styling alone is not the semantic contract; the
relation is included in the API response and is testable.

### Recursion And Expansion Limits

Call-site expansion must not recurse without bound. If a target workflow name
already exists in the current invocation ancestry, render a synthetic recursive
reference node for that call site instead of expanding the workflow again. The
reference node acts as both entry and exit, allowing the caller's return edge to
remain explicit.

Apply a deterministic maximum invocation/node budget as a second guard against
pathological call graphs. Exceeding the budget returns a useful diagram error
rather than a silently truncated or partially connected graph.

## Rationale

An invocation tree matches the mental model used by the run graph and removes
the misleading convergence of unrelated call sites. Explicit entry and exit
endpoints make nested returns composable, including when a workflow ends by
calling another workflow.

Adding a semantic edge relation avoids encoding behavior only in stroke styles.
It also lets backend tests verify topology without depending on frontend CSS.

Expanding runtime `for_each` branches was rejected because variables and data
determine cardinality. The definition diagram should show one branch template
and an all-branches join; the run graph remains the source of truth for actual
branches.

Keeping a shared group per workflow definition was rejected because it depicts
reuse accurately at the catalog level but inaccurately at the execution-flow
level. A separate dependency or definition-reference view could use that model
in the future.

## Consequences

Positive:

- Every sub-workflow call has an unambiguous child group.
- The diagram shows that child execution completes before the caller advances.
- Repeated `run-skill` calls align structurally with separate run-graph
  invocation boundaries.
- Nested and final-step calls have testable return semantics.
- Recursive definitions remain renderable without infinite expansion.

Negative:

- Large workflows become wider and contain repeated copies of reusable
  definitions.
- Node and edge IDs can no longer be generated or looked up solely by workflow
  name.
- Layout must account for invocation subtrees, return edges, and repeated
  groups.
- The diagram API and frontend edge mapping gain a semantic relation field and
  synthetic reference/pass-through nodes.

## Implementation Notes

- Replace the global `rendered[workflowName]` set with an invocation tree whose
  nodes carry definition name, invocation path, caller step, depth, and
  children.
- Escape path, call, and occurrence delimiters inside names, and add an
  occurrence suffix when step names are not unique within a workflow so IDs
  remain deterministic and collision-free.
- Build topology from invocation entry/exit endpoints before finalizing edges.
- Do not emit the ordinary sequence edge across a sub-workflow call.
- Extend `DiagramEdge` and the frontend `DiagramEdge` type with a relation such
  as `sequence`, `call`, or `return`.
- Add node metadata/type support for empty-workflow pass-through and recursive
  reference nodes.
- Layout complete invocation subtrees so cloned groups and return edges do not
  overlap unrelated groups.
- Preserve current conditional, gate, and `for_each` badges on caller steps.
- Compare `var/demos/end-to-end` with a completed run: repeated `run-skill`
  calls should have separate groups, while runtime branch counts remain only in
  the run graph.

## Related

- [Call-site-expanded workflow diagram plan](../plans/005-call-site-expanded-workflow-diagrams.md)
- [Call-site-expanded workflow diagram task](../tasks/done/call-site-expanded-workflow-diagrams.md)
- [Workflow input formats decision](ADR-0002-workflow-definition-input-formats.md)
- [Run graph workflow boundaries plan](../plans/002-run-graph-workflow-boundaries.md)
