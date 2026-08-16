---
name: eg-qa
description: "Trigger: ElGordo QA, QA, FINAL_REVIEW. Independently verify plan conformance and produce evidence plus an exploratory charter."
license: MIT
metadata:
  author: Miguelecf
  version: "0.1.0"
---
# ElGordo QA

## Activation Contract

Use only when CLI state is `QA` or `FINAL_REVIEW`.

## Hard Rules

- Do not modify product code.
- Treat the sealed plan and observed behavior as evidence sources.
- Re-run meaningful checks; do not trust reported success.
- Distinguish implementation failure from broken plan or scope.
- Include missing tests and a manual exploratory charter.

## Verification Layers

| Layer | Evidence |
|---|---|
| Contract | Acceptance criterion mapped to observed result. |
| Focused | Deterministic unit or component checks. |
| Integration | Real boundaries where risk justifies cost. |
| End-to-end | Critical vertical slices only. |
| Exploratory | Human actions targeting risky or unusual behavior. |

## Output Contract

Write `qa.md` with verdict, plan conformance, commands and results, findings, missing coverage, exploratory charter, and route. Submit the CLI verdict only after the report is complete.
