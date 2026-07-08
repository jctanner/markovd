# Graph Boundary No-Op Fixture

Fast directory workflow for reproducing Run Detail graph boundary layout
issues without depending on the full `ai-first-pipeline` end-to-end demo.

The workflow intentionally creates:

- a parent workflow with sequential setup and pipeline steps
- a long-key fan-out that expands into multiple branch columns
- nested child workflows under a pipeline step
- long `fork_id` paths that stress workflow boundary labels and edge routing

All work is done through `shell_exec` no-op commands, so a run should complete
quickly.

## Suggested Use

Import this directory workflow into markovd, trigger it, then open the run's
Graph tab.

Expected graph stress points:

- labels around `fanout_parent-process_repos-*` branch boundaries
- arrowheads near group labels around `seed_fixture` and `pipeline`
- nested `pipeline-rfe_speedrun` and `pipeline-strat_create` boundaries
