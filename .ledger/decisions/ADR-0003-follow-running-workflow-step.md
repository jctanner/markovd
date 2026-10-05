# ADR-0003: Follow the Running Workflow Step in the Graph

## Status

Accepted

## Context

The run detail page includes a React Flow graph that is refreshed as run and
step state is polled from the API. Its existing downward-arrow control moves
the viewport to the lowest rendered node. It is a one-time action and does not
continue moving the viewport as the workflow progresses.

For workflows that extend beyond the visible viewport, users must repeatedly
pan the graph or activate the jump control to keep the active work in view.
Forked workflows complicate automatic tracking because more than one step may
be running at once, and sufficiently large forks are represented by a collapsed
summary node rather than individual step nodes.

## Decision

Add an explicit follow mode to the run graph. A toggle button adjacent to the
existing jump control will enable or disable the mode. While enabled, the graph
will animate the viewport to the latest running task whenever polling produces
a new follow target.

Select the follow target using these rules:

1. Consider steps with status `running`.
2. Prefer the running step with the most recent `updated_at` value.
3. Use a deterministic step identifier as a tie-breaker so unchanged polling
   results do not move the viewport repeatedly.
4. When the selected step belongs to a collapsed fork, target that fork's
   rendered summary node.
5. If no step is running, retain the current viewport rather than jumping to a
   pending, completed, or failed step.

Following will preserve the user's current zoom level. Re-centering will occur
only when the resolved target changes, not on every render or polling response.
Follow mode will remain under explicit user control; ordinary graph updates
will not enable it automatically.

The existing downward-arrow control will retain its current one-time "jump to
bottom" behavior. Follow mode is a separate action with a distinct icon,
tooltip, pressed state, and accessible label.

## Rationale

The frontend already receives the information needed to follow execution, and
React Flow exposes viewport APIs such as `setCenter` and `getZoom`. Keeping the
feature in the graph component avoids backend state, additional API traffic,
and changes to run execution semantics.

Using `updated_at` gives concurrent branches a predictable definition of
"latest" that matches observed workflow progress. Tracking the resolved node
identifier prevents polling updates that do not change the active task from
continually interrupting manual inspection.

Preserving zoom avoids a jarring scale change each time execution advances.
Leaving the viewport unchanged when no task is running also avoids surprising
movement during transitions between steps or after a run finishes.

## Consequences

Positive:

- Users can monitor long-running workflows without repeatedly navigating the
  graph.
- The behavior requires no backend or persistence changes.
- Follow behavior remains opt-in and does not alter the existing jump action.
- Concurrent and collapsed fork behavior is defined rather than dependent on
  render order.

Negative:

- The graph builder must retain enough step metadata to map the newest running
  step to a rendered node or collapsed summary.
- In highly concurrent workflows, following the newest update means the view
  may move between branches as they report progress.
- Animated viewport changes can conflict with a user who pans while follow mode
  remains enabled; the enabled state must therefore be visually unambiguous.

## Implementation Notes

- Implement the follow controller inside the React Flow provider boundary so it
  can use `useReactFlow()`.
- Add `updated_at` and any required source-step identity to graph node metadata,
  without exposing them as visible node content.
- Store the last followed target in a ref or equivalent non-rendering state.
- Use the current React Flow zoom when calling `setCenter`.
- Resolve running steps inside collapsed forks to their summary node.
- Add focused tests for target selection, unchanged polling results, concurrent
  running steps, collapsed forks, and toggling follow mode.

## Related

- [Implement continuous follow mode for the run graph](../tasks/done/run-graph-follow-running-step.md)
