# Project Plan

`PLAN.md` is the navigation index for project state. Detailed plans, tasks,
bugs, decisions, and operational notes live under `docs/`.

## Overview

- [Project overview](docs/plans/000-overview.md)
- [Workflow input formats plan](docs/plans/001-workflow-input-formats.md)
- [Run graph workflow boundaries plan](docs/plans/002-run-graph-workflow-boundaries.md)
- [Markovd CLI plan](docs/plans/003-markovd-cli.md)
- [Standalone Markov workflow classification plan](docs/plans/004-standalone-workflow-classification.md)
- [Agent work ledger decision](docs/decisions/ADR-0001-agent-work-ledger.md)
- [Workflow definition input formats decision](docs/decisions/ADR-0002-workflow-definition-input-formats.md)
- [Run graph follow-mode decision](docs/decisions/ADR-0003-follow-running-workflow-step.md)
- [Standalone Markov workflow classification decision](docs/decisions/ADR-0004-classify-standalone-markov-workflows.md)

## Milestones

- [M1: Workflow Input Formats](docs/milestones/M1-workflow-input-formats.md)

## Active Tasks

None.

## Pending Tasks

None.

## Done Tasks

- [Classify standalone project workflow YAML](docs/tasks/done/standalone-workflow-classification.md)
- [Implement continuous follow mode for the run graph](docs/tasks/done/run-graph-follow-running-step.md)
- [Add Trigger Run workflow entrypoint override](docs/tasks/done/trigger-run-workflow-entrypoint.md)
- [Collapse Trigger Run volume selectors](docs/tasks/done/trigger-run-advanced-volumes.md)
- [Define workflow definition model](docs/tasks/done/workflow-definition-model.md)
- [Materialize and validate workflow definitions](docs/tasks/done/workflow-definition-validation.md)
- [Run directory workflows from shell and Kubernetes runners](docs/tasks/done/workflow-definition-runners.md)
- [Import directory workflows from projects](docs/tasks/done/workflow-definition-project-import.md)
- [Update workflow API and UI for file and directory definitions](docs/tasks/done/workflow-definition-api-ux.md)
- [Support diagrams for directory workflow definitions](docs/tasks/done/workflow-definition-diagrams.md)
- [Verify workflow definition formats end to end](docs/tasks/done/workflow-definition-e2e-verification.md)
- [Add Markovd API CLI](docs/tasks/done/markovd-cli.md)

## Open Bugs

- [Frontend lint baseline fails](docs/bugs/open/frontend-lint-baseline-fails.md)
- [Verify or implement `/api/v1/health`](docs/bugs/open/health-endpoint-unverified.md)
- [Use Kubernetes runner in k8s deployment](docs/bugs/open/runner-bug-1.md)
- [Markov job does not send callbacks](docs/bugs/open/runner-bug-3-callbacks-silent.md)
- [PVC artifact loader cannot read after job completion](docs/bugs/open/runner-bug-6-artifact-loader-pvc.md)
- [RunDetail UI memory growth on large runs](docs/bugs/open/ui-memory-leak-large-runs.md)

## Fixed Bugs

- [Run graph flattens nested workflow calls](docs/bugs/fixed/run-graph-flattens-nested-workflows.md)
- [Run graph workflow boundary labels overlap fan-out branches](docs/bugs/fixed/run-graph-workflow-boundary-labels-overlap.md)
- [Project import rejects directory workflows with `step_types/`](docs/bugs/fixed/project-import-rejects-step-types-directory.md)
- [Project import treats `meta.yaml` directory roots as files](docs/bugs/fixed/project-import-meta-root-detected-as-files.md)
- [Configurable Kubernetes job image pull policy](docs/bugs/fixed/runner-bug-2-imagepullpolicy.md)
- [Run ID mismatch between markovd and markov](docs/bugs/fixed/runner-bug-4-run-id-mismatch.md)
- [`rootRunID()` truncates `markov-run-*` IDs](docs/bugs/fixed/runner-bug-5-rootrunid-truncation.md)
- [Kubernetes directory workflows mount files at the wrong path](docs/bugs/fixed/runner-bug-7-directory-workflow-k8s-mount.md)

## Operational Notes

- [Markovd CLI reference](docs/reference/markovd-cli.md)
- [Accessing a running instance](docs/notes/accessing-running-instance.md)
- [Kubernetes admin credentials](docs/notes/k8s-admin-credentials.md)
- [Kubernetes job support requirements](docs/notes/k8s-job-support.md)
- [Kubernetes service account](docs/notes/k8s-service-account.md)
- [Session log](docs/notes/session-log.md)
