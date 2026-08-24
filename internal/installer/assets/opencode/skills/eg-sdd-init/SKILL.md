---
name: eg-sdd-init
description: "Trigger: SDD init, OpenSpec setup, project onboarding, ElGordo bootstrap. Detect existing spec tooling and guide initialization choices before anything is created."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo SDD Init

## Activation Contract

Use only when the project lacks ElGordo or OpenSpec initialization.

## Hard Rules

- Detect before proposing: existing `openspec/` directory, OpenSpec config and schemas, AGENTS.md, and the private runtime status check.
- Initialize only with explicit engineer permission. Never overwrite existing specs, config, or constitution.
- OpenSpec artifacts under `openspec/` are committed to Git; `.elgordo/` workflow state stays local and excluded.
- Gherkin GIVEN/WHEN/THEN scenarios are mandatory in every spec.

## Constitution Choices

Present one decision at a time, each with a recommendation:

| Choice | Options |
|---|---|
| Spec depth | Minimal deltas vs. full specifications |
| Review strictness | Warn vs. hard-fail thresholds (e.g. PR size) |
| TDD policy | Strict default vs. explicit exceptions |
| Schema | OpenSpec default schema vs. an existing project schema |

## Output Contract

Return detected state, the chosen constitution values, and the exact bootstrap steps for the conductor to run with approval (private runtime bootstrap, then OpenSpec init when absent).
