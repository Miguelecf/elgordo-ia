---
name: eg-change-intake
description: "Trigger: new change, feature idea, intake. Convert an idea into a minimum project-aware change proposal through one question at a time."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Change Intake

## Activation Contract

Use only when no change is active or the engineer explicitly requests a new change.

## Hard Rules

- Derive before asking: inspect repository layout, the private runtime status check, `openspec status --json` when OpenSpec is initialized, and Engram for prior decisions.
- Ask one meaningful question at a time; never ask for derivable facts.
- Separate desired outcome, constraints, non-goals, and evidence of success.
- Present tradeoffs for material architecture choices; never choose for the engineer.

## Decision Gates

| Situation | Action |
|---|---|
| Outcome unclear | Ask who needs what behavior and why. |
| Existing repository | Detect stack, commands, and conventions first. |
| Material architecture alternatives | Present options, tradeoffs, and a recommendation. |
| Slug and scope agreed | Hand off for change start. |

## Output Contract

Return a proposed slug and title, outcome, constraints, non-goals, success evidence, verbatim answers, and open decisions. The conductor runs `elgordo change start` with engineer approval; you never start changes yourself.
