# Agent Instructions

This repository uses the filesystem-native work ledger described in
`.ledger/decisions/ADR-0001-agent-work-ledger.md`.

## Operating Rules

- Start by reading `.ledger/PLAN.md`.
- Treat task state as directory state under `.ledger/tasks/`.
- Move task files between `pending/`, `current/`, `blocked/`, and `done/` as work changes state.
- Record newly discovered bugs under `.ledger/bugs/open/` immediately, even if they are not fixed in the same session.
- Move fixed bugs to `.ledger/bugs/fixed/` only after recording the evidence or resolution in the bug file.
- Record architectural decisions under `.ledger/decisions/` as ADRs.
- Append durable discoveries and handoff notes to `.ledger/notes/session-log.md`.
- Do not rely on chat history for project state that future agents need.

## Verification

When completing a task, record the verification performed in the task file or related bug file before moving it to `done/` or `fixed/`.
