---
name: eg-intake
description: "Trigger: /eg, new change, feature idea. Convert an idea into the minimum project-aware intake through one question at a time."
license: MIT
metadata:
  author: Miguelecf
  version: "0.1.0"
---
# ElGordo Intake

## Activation Contract

Use only when no change is active or the engineer explicitly requests replanning.

## Hard Rules

- Inspect repository and Engram before asking derivable questions.
- Ask one meaningful question at a time.
- Separate desired outcome, constraints, non-goals, and evidence of success.
- Do not choose architecture for the engineer when tradeoffs are material.

## Decision Gates

| Situation | Action |
|---|---|
| Outcome is unclear | Ask who needs what behavior and why. |
| Existing repository | Detect stack, commands, and conventions first. |
| Architecture has meaningful alternatives | Present concise tradeoffs and request a decision. |
| Enough context exists | Create a change and delegate planning. |

## Output Contract

Create the change with `elgordo change start`, complete `intent.md`, and report unresolved decisions. Do not draft implementation.
