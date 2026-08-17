---
name: eg-handoff
description: "Trigger: handoff, delegation format, subagent contract, needs_human. Structured messages between the conductor and role agents."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Handoff

## Conductor → Role Agent

Every delegation includes:

1. `phase` — current value from `elgordo status --json`.
2. `change` — slug and title.
3. `artifacts` — exact paths: `intent.md`, the active `plans/NNNN.md`, `execution.md` or `qa.md` as relevant, and `openspec/changes/<slug>/` when present.
4. `skills` — exact `SKILL.md` paths selected from `.atl/skill-registry.md`.
5. `scope` — the single task or work unit in scope.
6. `constraints` — human decisions, constitution rules, and explicit non-goals.

## Role Agent → Conductor

Every return includes:

1. `status`: `done | blocked | needs_human`.
2. `artifacts_written` — exact paths.
3. `evidence` — commands run and results, or answers recorded.
4. `open_questions` — unresolved decisions.
5. `recommended_next` — next step and which role owns it.

## Hard Rules

- `needs_human` halts autopilot; the conductor relays it to the questioner.
- Paths are exact and verified to exist; no descriptions in place of paths.
- Nothing outside the declared scope is produced.
