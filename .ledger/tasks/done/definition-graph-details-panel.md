# Task: Show workflow and step descriptions in the definition graph

## Status

Done (2026-10-10)

## Summary

Markov workflows and steps carry `description` fields, but the workflow
definition graph didn't show them. Step cards are fixed at 260×72 and laid
out on the server, so long text doesn't belong in them.

## Changes

- **Details panel:** the legend's selection box grew into one.
  - Selecting a step shows its description in full, wrapped and scrollable,
    plus its type and path, and any of `when`, `for_each`, `for_each_when`,
    the called workflow (templated names as written), `rules`,
    `failed_when` and `ignore_errors`.
  - Selecting a workflow group shows the workflow's description.
- **Cards:** a notes icon on steps that have a description; the hover
  tooltip stays.
- **Group headers:** the workflow description on one truncated line, with
  the full text on hover. `groupPadTop` went from 72 to 90 to make room, so
  groups are 18px taller.
- **Diagram API:** carries the workflow `description`, plus `forEachWhen`,
  `ignoreErrors` and `failedWhen` on step nodes.

## Verification

- `go test ./...` passes, including the new
  `TestGenerateDiagramCarriesDescriptionsAndStepDetails`.
- `ui`: `tsc --noEmit`, `npm test` (14 pass) and `npm run build` pass.
- Checked in a browser on the deployed `var-benchmarks-strat-workflow` page:
  - selecting `wait_for_run` showed its description and
    `ignore_errors: true`;
  - selecting the `run-test` group showed the workflow description;
  - `describe_run` has the notes icon;
  - the `run-test` header shows its one-line description above the first
    card.
