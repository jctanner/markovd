# Project Plan

`PLAN.md` is the navigation index for project state. Detailed plans, tasks,
bugs, decisions, and operational notes live under `docs/`.

## Overview

- [Project overview](plans/000-overview.md)
- [Workflow input formats plan](plans/001-workflow-input-formats.md)
- [Run graph workflow boundaries plan](plans/002-run-graph-workflow-boundaries.md)
- [Markovd CLI plan](plans/003-markovd-cli.md)
- [Standalone Markov workflow classification plan](plans/004-standalone-workflow-classification.md)
- [Call-site-expanded workflow diagram plan](plans/005-call-site-expanded-workflow-diagrams.md)
- [Agent work ledger decision](decisions/ADR-0001-agent-work-ledger.md)
- [Workflow definition input formats decision](decisions/ADR-0002-workflow-definition-input-formats.md)
- [Run graph follow-mode decision](decisions/ADR-0003-follow-running-workflow-step.md)
- [Standalone Markov workflow classification decision](decisions/ADR-0004-classify-standalone-markov-workflows.md)
- [Call-site-expanded workflow diagram decision](decisions/ADR-0005-expand-definition-diagrams-by-call-site.md)

## Milestones

- [M1: Workflow Input Formats](milestones/M1-workflow-input-formats.md)

## Active Tasks

None.

## Pending Tasks

None.

## Done Tasks

- [Show workflow and step descriptions in the definition graph](tasks/done/definition-graph-details-panel.md)
- [Keep runs on a data volume, resume runs, show step descriptions](tasks/done/run-data-volume-and-resume.md)
- [Keep all files on project import and record the source commit](tasks/done/project-import-all-files-and-source-commit.md)
- [Improve workflow definition edge readability](tasks/done/improve-workflow-definition-edge-readability.md)
- [Expand workflow diagrams by call site](tasks/done/call-site-expanded-workflow-diagrams.md)
- [Classify standalone project workflow YAML](tasks/done/standalone-workflow-classification.md)
- [Implement continuous follow mode for the run graph](tasks/done/run-graph-follow-running-step.md)
- [Add Trigger Run workflow entrypoint override](tasks/done/trigger-run-workflow-entrypoint.md)
- [Collapse Trigger Run volume selectors](tasks/done/trigger-run-advanced-volumes.md)
- [Define workflow definition model](tasks/done/workflow-definition-model.md)
- [Materialize and validate workflow definitions](tasks/done/workflow-definition-validation.md)
- [Run directory workflows from shell and Kubernetes runners](tasks/done/workflow-definition-runners.md)
- [Import directory workflows from projects](tasks/done/workflow-definition-project-import.md)
- [Update workflow API and UI for file and directory definitions](tasks/done/workflow-definition-api-ux.md)
- [Support diagrams for directory workflow definitions](tasks/done/workflow-definition-diagrams.md)
- [Verify workflow definition formats end to end](tasks/done/workflow-definition-e2e-verification.md)
- [Add Markovd API CLI](tasks/done/markovd-cli.md)
- [Show claude step icon and live step_progress](tasks/done/claude-step-progress-ui.md)

## Open Bugs

- [Mobile navigation causes horizontal page overflow](bugs/open/mobile-navigation-horizontal-overflow.md)
- [Frontend lint baseline fails](bugs/open/frontend-lint-baseline-fails.md)
- [Verify or implement `/api/v1/health`](bugs/open/health-endpoint-unverified.md)
- [Use Kubernetes runner in k8s deployment](bugs/open/runner-bug-1.md)
- [Markov job does not send callbacks](bugs/open/runner-bug-3-callbacks-silent.md)
- [PVC artifact loader cannot read after job completion](bugs/open/runner-bug-6-artifact-loader-pvc.md)
- [RunDetail UI memory growth on large runs](bugs/open/ui-memory-leak-large-runs.md)
- [Run stays running when markov exits before emitting events](bugs/open/shell-runner-failure-leaves-run-running.md)

## Fixed Bugs

- [Run graph flattens nested workflow calls](bugs/fixed/run-graph-flattens-nested-workflows.md)
- [Run graph workflow boundary labels overlap fan-out branches](bugs/fixed/run-graph-workflow-boundary-labels-overlap.md)
- [Project import rejects directory workflows with `step_types/`](bugs/fixed/project-import-rejects-step-types-directory.md)
- [Project import treats `meta.yaml` directory roots as files](bugs/fixed/project-import-meta-root-detected-as-files.md)
- [Configurable Kubernetes job image pull policy](bugs/fixed/runner-bug-2-imagepullpolicy.md)
- [Run ID mismatch between markovd and markov](bugs/fixed/runner-bug-4-run-id-mismatch.md)
- [`rootRunID()` truncates `markov-run-*` IDs](bugs/fixed/runner-bug-5-rootrunid-truncation.md)
- [Kubernetes directory workflows mount files at the wrong path](bugs/fixed/runner-bug-7-directory-workflow-k8s-mount.md)

## Operational Notes

- [Markovd CLI reference](../docs/reference/markovd-cli.md)
- [Accessing a running instance](notes/accessing-running-instance.md)
- [Kubernetes admin credentials](notes/k8s-admin-credentials.md)
- [Kubernetes job support requirements](notes/k8s-job-support.md)
- [Kubernetes service account](notes/k8s-service-account.md)
- [Session log](notes/session-log.md)
