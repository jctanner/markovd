# Task: Verify Workflow Definition Formats End to End

## Goal

Prove that both single-file and directory workflow definitions work through the
full markovd stack.

## Context

Directory workflow support depends on a newer Markov binary than the one
currently checked into `./bin/markov`. Compose verification must refresh that
binary from the sibling Markov checkout before the stack is built or started.

## Acceptance Criteria

- [ ] Copy `../markov/bin/markov` to `./bin/markov` before compose verification.
- [ ] Rebuild and start the compose stack after copying the binary.
- [ ] Upload or import a single-file workflow.
- [ ] Upload or import a directory workflow.
- [ ] Validate both workflow definitions through the API/UI.
- [ ] Generate diagrams for both workflow definitions.
- [ ] Run both workflow definitions and inspect run detail pages.
- [ ] Verify project re-sync updates a directory workflow definition.
- [ ] Submit an invalid directory definition and confirm it is rejected before
      storage or execution.
- [ ] Record compose commands, workflow names, run IDs, Markov binary source, and
      observed results in this task before moving it to `done/`.

## Files Likely Involved

- `bin/markov`
- `Makefile`
- `podman-compose.yml`
- `docs/plans/001-workflow-input-formats.md`
- UI and API files touched by the implementation tasks

## Status

Pending

## Notes

The required binary refresh command is:

```bash
cp ../markov/bin/markov ./bin/markov
```
