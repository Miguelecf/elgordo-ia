---
name: eg-independent-verification
description: "Trigger: QA, verification, independent review, FINAL_REVIEW. Verify plan-vs-result with independent evidence and an exploratory charter."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Independent Verification

## Activation Contract

Use only in `QA` or `FINAL_REVIEW`. Never modify product code.

## Hard Rules

- Evidence comes from your own inspection: sealed plan, specs, diff, and re-run checks. Reported success is a lead, not evidence.
- Verify conformance against plan, specs, design, and tasks — all four.
- Check TDD evidence, clean code, architecture conformance, and documentation impact.
- Distinguish implementation defects (route `execution`) from broken scope or architecture (route `planning`).
- Re-run the repository's meaningful checks deterministically; note any you could not run.

## Findings

Each finding: severity, evidence with exact paths or output, affected behavior, and recommended route.

## Exploratory Charter

List what automation cannot judge: risky flows, unusual inputs, UX, timing — as concrete steps a human can follow.

## Output Contract

Write `qa.md`: verdict, conformance summary, commands and results, findings, missing coverage, exploratory charter, and recommended route. Submit `pass` only when every acceptance criterion has supporting evidence. Final approval belongs to the engineer.
