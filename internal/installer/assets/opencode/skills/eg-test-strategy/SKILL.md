---
name: eg-test-strategy
description: "Trigger: test strategy, test plan, coverage, which tests. Choose verification layers by risk and set the TDD policy for the change."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Test Strategy

## Activation Contract

Use during planning to define per-unit verification, and during QA to judge its adequacy.

## Layer Selection

Choose by risk and behavior, not habit:

| Layer | Use for |
|---|---|
| Unit | Pure logic, branches, boundaries. |
| Integration | Real boundaries: DB, HTTP, filesystem — where risk justifies cost. |
| End-to-end | Critical vertical slices only. |
| Manual exploratory | What automation cannot judge; becomes the QA charter. |

## TDD Policy

- Strict TDD is the default for every behavior change: failing test first, then implementation, then refactor.
- Exceptions must be explicit in the plan with a reason (spike, mechanical refactor, generated code) and replacement verification.
- Changes to existing behavior start from a characterization test.

## Hard Rules

- Every spec scenario maps to at least one planned check.
- Flaky or environment-dependent checks are flagged, never silently relied on.

## Output Contract

Per work unit: layers chosen, scenarios covered, TDD or explicit exception, and the exact verification command.
