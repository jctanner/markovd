# Project Plan

`PLAN.md` is the navigation index for project state. Detailed plans, tasks,
bugs, decisions, and operational notes live under `docs/`.

## Overview

- [Project overview](docs/plans/000-overview.md)
- [Agent work ledger decision](docs/decisions/ADR-0001-agent-work-ledger.md)

## Active Tasks

No task files are currently in `docs/tasks/current/`.

## Pending Tasks

No task files are currently in `docs/tasks/pending/`.

## Open Bugs

- [Verify or implement `/api/v1/health`](docs/bugs/open/health-endpoint-unverified.md)
- [Use Kubernetes runner in k8s deployment](docs/bugs/open/runner-bug-1.md)
- [Markov job does not send callbacks](docs/bugs/open/runner-bug-3-callbacks-silent.md)
- [PVC artifact loader cannot read after job completion](docs/bugs/open/runner-bug-6-artifact-loader-pvc.md)
- [RunDetail UI memory growth on large runs](docs/bugs/open/ui-memory-leak-large-runs.md)

## Fixed Bugs

- [Configurable Kubernetes job image pull policy](docs/bugs/fixed/runner-bug-2-imagepullpolicy.md)
- [Run ID mismatch between markovd and markov](docs/bugs/fixed/runner-bug-4-run-id-mismatch.md)
- [`rootRunID()` truncates `markov-run-*` IDs](docs/bugs/fixed/runner-bug-5-rootrunid-truncation.md)

## Operational Notes

- [Accessing a running instance](docs/notes/accessing-running-instance.md)
- [Kubernetes admin credentials](docs/notes/k8s-admin-credentials.md)
- [Kubernetes job support requirements](docs/notes/k8s-job-support.md)
- [Kubernetes service account](docs/notes/k8s-service-account.md)
- [Session log](docs/notes/session-log.md)
