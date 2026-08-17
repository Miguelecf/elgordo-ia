---
name: eg-documentation-impact
description: "Trigger: documentation impact, docs, README, API docs, migration guide. Decide what documentation a change must carry and prove it landed."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Documentation Impact

## Activation Contract

Use during planning to declare impact, and during execution and QA to verify it landed.

## Hard Rules

- Public behavior, API, configuration, CLI surface, or migration changes require doc updates in the same work unit.
- Detect impact from the diff and specs, not from memory.
- "No docs impact" requires an explicit rationale recorded in the plan or `execution.md`.
- Update project docs (README, AGENTS.md, docs/) that describe what changed; stale docs are a defect.

## Impact Signals

| Change | Doc |
|---|---|
| New or changed flag, command, endpoint | Usage docs. |
| User-visible behavior change | README or feature docs. |
| Config or schema change | Config reference and migration note. |
| Convention or workflow change | AGENTS.md. |

## Output Contract

Per work unit: docs touched or an explicit no-impact rationale. QA flags undocumented impact as a finding.
