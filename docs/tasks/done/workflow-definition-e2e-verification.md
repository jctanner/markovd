# Task: Verify Workflow Definition Formats End to End

## Goal

Prove that both single-file and directory workflow definitions work through the
full markovd stack.

## Context

Directory workflow support depends on a newer Markov binary than the one
currently checked into `./bin/markov`. Compose verification must refresh that
binary from the sibling Markov checkout before the stack is built or started.

## Acceptance Criteria

- [x] Copy `../markov/bin/markov` to `./bin/markov` before compose verification.
- [x] Rebuild and start the compose stack after copying the binary.
- [x] Upload or import a single-file workflow.
- [x] Upload or import a directory workflow.
- [x] Validate both workflow definitions through the API/UI.
- [x] Generate diagrams for both workflow definitions.
- [x] Run both workflow definitions and inspect run detail pages.
- [x] Verify project re-sync updates a directory workflow definition.
- [x] Submit an invalid directory definition and confirm it is rejected before
      storage or execution.
- [x] Record compose commands, workflow names, run IDs, Markov binary source, and
      observed results in this task before moving it to `done/`.

## Files Likely Involved

- `bin/markov`
- `Makefile`
- `podman-compose.yml`
- `docs/plans/001-workflow-input-formats.md`
- UI and API files touched by the implementation tasks

## Status

Done

## Notes

The required binary refresh command is:

```bash
cp ../markov/bin/markov ./bin/markov
```

Verification completed with the refreshed `../markov/bin/markov` binary copied
to `./bin/markov` before compose startup.

Commands and checks:

```bash
go test ./...
npm run build
cp ../markov/bin/markov ./bin/markov
./bin/markov validate ../markov/examples/dir-based-hello-world
./bin/markov validate ../markov/examples/k8s-job-test.yaml
uv tool run podman-compose build
uv tool run podman-compose up -d
curl http://localhost:8082/api/v1/health
```

Observed results:

- API health returned `{"status":"healthy"}`.
- Single-file workflow `single-file-smoke` validated, diagrammed, and completed
  run `markov-run-43cea868`.
- Manual directory workflow `directory-smoke` validated, diagrammed, and
  completed run `markov-run-20926252`.
- Invalid directory validation was rejected before storage with missing
  `vars.yaml`.
- Project directory workflow `pipeline` imported from project `1`, diagrammed,
  and completed run `markov-run-6b9c3df9`.
- Project re-sync updated `pipeline` from `project-directory-v1` to
  `project-directory-v2`; run `markov-run-6b6cdbd7` completed with stdout
  `project-directory-v2`.
