---
name: eg-context-budget
description: "Trigger: context budget, too much context, what to read, token budget. Read only what the current work needs and pass exact paths."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Context Budget

## Hard Rules

- Read only the current work unit and the authorities it names: active plan, relevant specs, touched files, backing ADRs.
- Pass and request exact file paths — never broad directory dumps, never whole-repo scans.
- Pull Engram memories by targeted search, not bulk context.
- For large documents, read the sections the unit cites.
- Stop and report when a unit genuinely needs more context than fits; that is a plan-splitting signal.

## For The Conductor

Delegate with: exact artifact paths, exact `SKILL.md` paths from `.atl/skill-registry.md`, and the single work unit in scope. Nothing else.

## For Role Agents

Refuse unscoped exploration. Map what you must read before reading it, and note what you deliberately skipped.

## Output Contract

Report files read and files deliberately not read. Unread-but-relevant material becomes an explicit risk, never silent ignorance.
