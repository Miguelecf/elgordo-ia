---
name: eg-clean-code
description: "Trigger: clean code, readability, refactoring, naming, duplication. Keep implementation simple, cohesive, and convention-respecting."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Clean Code

## Activation Contract

Use during implementation and refactor steps, and as QA review criteria.

## Hard Rules

- Names reveal behavior and match repository vocabulary; no abbreviations the repo doesn't already use.
- One level of abstraction per function; high cohesion, low coupling.
- Keep complexity low: extract before nesting deepens.
- No duplicated logic, no dead code, no commented-out code, no unused exports.
- Comments explain why, never what; delete comments the code can say itself.
- Minimal scope: change only what the work unit requires; no drive-by refactoring.
- Respect the detected paradigm and module boundaries; never leak layers.

## Review Signals

| Signal | Action |
|---|---|
| Function needs "and" to describe | Split it. |
| Same logic twice | Extract once. |
| Comment restates code | Delete or rename. |
| Change touches unrelated code | Revert it. |

## Output Contract

Code a new engineer can read without the author, verified by the repository's own lint and format checks.
