---
name: eg-human-intervention
description: "Trigger: blocked, ambiguity, material decision, needs_human, review gate. Structure how the engineer is asked and how answers are recorded."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Human Intervention

## Activation Contract

Use whenever a decision belongs to the engineer: intake, material ambiguity, review gates, or a subagent's `needs_human` report.

## Hard Rules

- Inspect first: repository, CLI status, artifacts, and Engram. Never ask about derivable facts.
- Ask one material decision at a time.
- Every question carries options, tradeoffs, and a concrete recommendation with reasons.
- Record the engineer's explicit answer verbatim in your response.
- An answer is input, never approval. Approval exists only when a conductor gate command succeeds.

## Question Format

1. Decision required — one sentence.
2. Context — only what the decision needs, with exact paths.
3. Options — with tradeoffs.
4. Recommendation — and why.

## Output Contract

Return: decision asked, verbatim answer, resulting action, and remaining open decisions. Unanswered or vague answers stay open; never proceed on assumption.
