# M1: Workflow Input Formats

## Goal

Make markovd support both single-file and directory-based Markov workflow
definitions across API, persistence, runners, project import, diagrams, and UI.

Detailed plan: [Workflow Input Formats Plan](../plans/001-workflow-input-formats.md)

Decision record: [ADR-0002: Support File and Directory Workflow Definitions](../decisions/ADR-0002-workflow-definition-input-formats.md)

## Tasks

- [Define workflow definition model](../tasks/done/workflow-definition-model.md)
- [Materialize and validate workflow definitions](../tasks/done/workflow-definition-validation.md)
- [Run directory workflows from shell and Kubernetes runners](../tasks/done/workflow-definition-runners.md)
- [Import directory workflows from projects](../tasks/done/workflow-definition-project-import.md)
- [Update workflow API and UI for file and directory definitions](../tasks/done/workflow-definition-api-ux.md)
- [Support diagrams for directory workflow definitions](../tasks/done/workflow-definition-diagrams.md)
- [Verify workflow definition formats end to end](../tasks/done/workflow-definition-e2e-verification.md)

## Success Criteria

- Single-file workflows keep current behavior.
- Directory workflows can be uploaded or imported, viewed, diagrammed, run, and
  re-synced.
- The implementation rejects invalid or unsafe file paths.
- Compose verification refreshes `./bin/markov` from `../markov/bin/markov`
  before the stack is built or started.
