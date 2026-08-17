---
name: eg-specification
description: "Trigger: writing specs, requirements, scenarios, OpenSpec specs. Write testable SHALL/MUST requirements with observable Gherkin scenarios."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Specification

## Activation Contract

Use when drafting or revising `openspec/changes/<slug>/specs/**`.

## Hard Rules

- Requirements use SHALL or MUST; every requirement has at least one scenario.
- Scenarios use GIVEN/WHEN/THEN and describe observable behavior, never implementation details.
- Cover positive, negative, boundary, and failure cases for every behavior.
- Write deltas against existing specs; do not restate unchanged behavior.
- Follow `openspec instructions specs --change <slug> --json` when the CLI exists.

## Scenario Quality

| Smell | Fix |
|---|---|
| Vague outcome ("works correctly") | State the observable result. |
| Implementation coupling ("calls X") | Describe external behavior. |
| Missing failure path | Add WHEN the precondition is violated. |
| Boundary ignored | Add scenarios at the edges. |

## Output Contract

Each requirement: one SHALL/MUST statement plus scenarios with concrete GIVEN context, a single WHEN action, and a THEN observable outcome — named so tests can trace to them.
