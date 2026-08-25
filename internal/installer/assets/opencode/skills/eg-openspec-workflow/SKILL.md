---
name: eg-openspec-workflow
description: "Trigger: OpenSpec, proposal, specs, design, tasks, openspec CLI. Coordinate ElGordo orchestration with OpenSpec artifact format and status."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo OpenSpec Workflow

## Authority Split

- ElGordo owns orchestration: workflow phase, gates, and approved scope come from the private runtime status check only.
- OpenSpec owns artifact format and per-artifact status. Its `state.yaml` is never workflow authority.
- Never use `/opsx` editor commands. The OpenSpec CLI is required for versioned artifacts.

## Artifact Flow

Use the ElGordo change slug as the OpenSpec change name. Create artifacts in dependency order under `openspec/changes/<slug>/`:

1. `proposal.md` — why and what changes.
2. `specs/**` — requirement deltas with GIVEN/WHEN/THEN scenarios.
3. `design.md` — technical approach, when the change needs one.
4. `tasks.md` — ordered implementation tasks mirroring plan work units.

## CLI Usage

Use the OpenSpec CLI rather than hand-writing schema logic:

- `openspec status --change <slug> --json` — artifact completion and next steps.
- `openspec instructions <artifact> --change <slug> --json` — authoritative template, rules, and output path per artifact.
- `openspec validate <slug> --strict --json` — validate before marking planning ready.

When the CLI is absent, return `needs_human` so the conductor can obtain approval for lazy dependency bootstrap. Do not create or validate artifacts until `elgordo init` has installed or verified the pinned CLI.

## Output Contract

Artifacts validate, mirror the ElGordo plan, and stay committed to Git while `.elgordo/` remains local.
