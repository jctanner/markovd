# Agent Instructions

This repository uses the filesystem-native work ledger described in
`docs/decisions/ADR-0001-agent-work-ledger.md`.

## Operating Rules

- Start by reading `PLAN.md`.
- Treat task state as directory state under `docs/tasks/`.
- Move task files between `pending/`, `current/`, `blocked/`, and `done/` as work changes state.
- Record newly discovered bugs under `docs/bugs/open/` immediately, even if they are not fixed in the same session.
- Move fixed bugs to `docs/bugs/fixed/` only after recording the evidence or resolution in the bug file.
- Record architectural decisions under `docs/decisions/` as ADRs.
- Append durable discoveries and handoff notes to `docs/notes/session-log.md`.
- Do not rely on chat history for project state that future agents need.

## Verification

When completing a task, record the verification performed in the task file or related bug file before moving it to `done/` or `fixed/`.
