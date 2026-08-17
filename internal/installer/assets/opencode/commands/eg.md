---
description: Start or continue the ElGordo human-led engineering workflow.
agent: eg
subtask: false
---
Run `elgordo status --json` first and treat its output as workflow authority. Route only by its `phase`.

User request: $ARGUMENTS

Read `.atl/skill-registry.md` and delegate to the single role agent matching the phase (`eg-questioner`, `eg-planner`, `eg-executor`, or `eg-qa`), passing only the relevant exact `SKILL.md` paths and exact artifact paths. Inside a work phase you may iterate briefly under bounded autopilot; always stop at `PLAN_REVIEW`, `CODE_REVIEW`, and `FINAL_REVIEW`, and before every human gate command, so the engineer decides.

Never edit files, never claim a gate passed unless the corresponding `elgordo` command succeeded, and never use OpenSpec `/opsx` commands. The engineer may select a different model with `/models` before each phase.
