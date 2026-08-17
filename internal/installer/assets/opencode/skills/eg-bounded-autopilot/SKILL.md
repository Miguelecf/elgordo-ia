---
name: eg-bounded-autopilot
description: "Trigger: autopilot, autonomous continuation, keep going, bounded iteration. Conductor-only limits for unattended progress."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Bounded Autopilot

## Activation Contract

Conductor only. Governs how far the workflow advances without engineer input.

## Hard Rules

- Autopilot exists only inside work phases: `PLANNING`, `EXECUTING`, `QA`.
- Always stop at every review phase (`PLAN_REVIEW`, `CODE_REVIEW`, `FINAL_REVIEW`) and before every human gate command.
- Stop on any CLI failure, any subagent `needs_human`, or any delegation that produces no new artifact or evidence (no-progress stop).
- Keep iterations short: one delegation, then re-read `elgordo status --json` before the next.
- Never run a gate command to keep momentum; never let a producer review its own work.

## Budget

An autopilot run ends at the earliest of: review phase reached, gate required, subagent blocked, no progress, or the engineer interrupts.

## Output Contract

On stop, report: delegations made, artifacts changed, current CLI phase, and the exact decision now required from the engineer.
