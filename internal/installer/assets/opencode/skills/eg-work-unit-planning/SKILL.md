---
name: eg-work-unit-planning
description: "Trigger: work units, plan breakdown, task decomposition. Split a change into complete, independently verifiable, commit-ready units."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Work Unit Planning

## Activation Contract

Use when drafting or revising `.elgordo/changes/<slug>/plans/NNNN.md` work units.

## Hard Rules

- A work unit is one complete behavior, not a line-count target or a file list.
- Each unit is independently verifiable and leaves the repository working.
- Declare dependencies between units and order them so each builds on green predecessors.
- Each unit carries: behavior, touched areas, tests, verification command, and rollback note.
- Each unit is commit-ready: code, tests, and docs land together.

## Splitting Tests

| Signal | Action |
|---|---|
| Unit needs another's code to run | Merge or reorder. |
| Unit can't be verified alone | Redefine the boundary. |
| Unit spans unrelated behaviors | Split. |
| Rollback unclear | Shrink until revert is obvious. |

## Output Contract

Ordered units with dependencies, acceptance criteria mapped to spec scenarios, verification, risks, and rollback — ready for human seal.
