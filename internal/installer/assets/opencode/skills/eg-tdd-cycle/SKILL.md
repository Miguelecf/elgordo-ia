---
name: eg-tdd-cycle
description: "Trigger: TDD, red green refactor, test first. Apply the TDD cycle with captured evidence and an explicit exception contract."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo TDD Cycle

## Activation Contract

Use for every behavior change in `EXECUTING` unless the plan records an explicit exception.

## Cycle

1. **Red** — write the smallest failing test that names the behavior; run it and capture the failure.
2. **Green** — write the minimum code to pass; run it and capture the pass.
3. **Refactor** — improve structure with tests green; re-run.

One behavior per cycle. Never batch multiple behaviors into one red.

## Evidence

Record in `execution.md` per unit: the failing output, the passing output, and refactor notes. Claimed TDD without captured output is not evidence.

## Exception Contract

An exception is valid only when the plan states it with a reason and replacement verification (e.g. characterization test after a mechanical refactor). Otherwise, missing red-green evidence is a QA finding.

## Hard Rules

- Never write implementation before its failing test.
- Never weaken or delete a test to reach green.
- Keep cycles small enough to revert in minutes.
