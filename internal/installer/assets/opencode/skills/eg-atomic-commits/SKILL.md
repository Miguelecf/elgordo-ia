---
name: eg-atomic-commits
description: "Trigger: commit, atomic commit, conventional commits, staged diff. Commit one complete work unit at a time with tests and docs."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Atomic Commits

## Hard Rules

- One commit per completed work unit: behavior, its tests, and its docs land together.
- Message follows Conventional Commits: `type(scope): summary`, with a body for the why when non-obvious. Type matches `feat|fix|refactor|docs|test|chore`.
- Inspect the full staged diff before committing: no scope creep, no unrelated edits, no debug leftovers.
- Scan for secrets (tokens, keys, credentials) before staging; never commit them.
- Never amend, force, or rewrite history; never commit on a protected branch.
- Commit only when the unit's checks pass; a red tree stays uncommitted.

## Pre-Commit Checklist

1. `git status` — only intended files.
2. `git diff --staged` — complete unit, nothing extra.
3. Verification command green.
4. Message states behavior, not mechanics — the what is in the diff.

## Output Contract

Record commit hash and message in `execution.md`. Commits are human-approved bash actions; push always belongs to the engineer.
