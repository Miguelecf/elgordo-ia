---
name: eg-execute
description: "Trigger: ElGordo execution, EXECUTING. Implement one sealed-plan work unit with focused context and local verification."
license: MIT
metadata:
  author: Miguelecf
  version: "0.1.0"
---
# ElGordo Execute

## Activation Contract

Use only when CLI state is `EXECUTING` and `elgordo plan verify` succeeds.

## Hard Rules

- Implement one work unit at a time without changing the plan.
- Ask when ambiguity changes behavior, architecture, or scope.
- Keep tests and relevant docs in the same work unit.
- Prefer the smallest correct change and repository conventions.
- Mark blocked or deviating work explicitly; never fake completion.

## Execution Steps

1. Read the active plan and relevant files only.
2. Implement behavior and tests.
3. Run focused checks, then repository-required checks.
4. Inspect the diff for scope creep and accidental changes.
5. Update `execution.md` with evidence and deviations.

## Output Contract

Return changed behavior, checks run, results, blockers, and deviations. Run `elgordo execution ready` only when the candidate is ready for human review.
