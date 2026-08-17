---
name: eg-architecture-discovery
description: "Trigger: architecture, conventions, paradigm, project structure, ADR. Determine the authoritative architecture before planning or changing code."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Architecture Discovery

## Authority Order

Resolve architecture questions in this order; earlier wins:

1. Explicit human decision in the active change.
2. AGENTS.md and project documentation.
3. ADRs and recorded design decisions.
4. Conventions observed in the code.
5. Engram memories — verify stale entries against code before trusting.
6. Generic guidance — last resort, and say so.

## Hard Rules

- Detect the paradigm (layered, hexagonal, event-driven, modular monolith, etc.) from structure and naming before proposing anything.
- Never invent conventions that conflict with observed code.
- Material conflicts between authorities go to the engineer as `needs_human` with options.
- Cite the exact file or document backing each architectural claim.

## Output Contract

Return: detected paradigm, authoritative conventions with evidence paths, constraints the plan must respect, and open architecture decisions needing the engineer.
