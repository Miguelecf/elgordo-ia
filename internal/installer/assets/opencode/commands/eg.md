---
description: Start or continue the ElGordo human-led engineering workflow.
agent: eg
subtask: false
---
Start by running `elgordo status --json`. Treat its output as workflow authority.

User request: $ARGUMENTS

If the project is not initialized, explain what `elgordo init` creates and ask permission before running it. If there is no active change, use the intake skill and ask one necessary question at a time before starting a change.

Delegate exactly one role matching the current phase. Never claim that a gate passed unless the corresponding `elgordo` command succeeds. The engineer may select a different model with `/models` before each phase.
