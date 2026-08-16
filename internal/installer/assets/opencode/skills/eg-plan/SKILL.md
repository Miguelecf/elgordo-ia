---
name: eg-plan
description: "Trigger: ElGordo planning, PLANNING, PLAN_REVIEW. Produce or revise one executable sealed-plan candidate."
license: MIT
metadata:
  author: Miguelecf
  version: "0.1.0"
---
# ElGordo Plan

## Activation Contract

Use only for the active change in `PLANNING` or `PLAN_REVIEW`.

## Hard Rules

- Do not write product code.
- Define work units as complete, testable behaviors.
- Include dependencies, acceptance criteria, verification, risks, and rollback.
- Keep assumptions explicit and ask the engineer about material ambiguity.
- Never seal or approve your own plan.

## Decision Gates

| Finding | Action |
|---|---|
| External API uncertainty | Query Context7 narrowly. |
| Prior architecture exists | Retrieve it from Engram and verify against code. |
| Work unit cannot be independently verified | Split or redefine it. |
| Blocking question remains | Leave plan in `PLANNING`. |

## Output Contract

Update `intent.md` and the active `plans/NNNN.md`. End with scope, ordered work units, DoD, risks, and open questions. Run `elgordo plan ready` only when reviewable.
