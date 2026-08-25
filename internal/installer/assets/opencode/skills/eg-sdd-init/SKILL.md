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
- When Engram or OpenSpec is missing, explain why each is required and obtain explicit approval for each installation. Tell the conductor to include only the approved `--accept-engram-install` and `--accept-openspec-install` flags on `elgordo init`.
- Node.js 20.19.0+ and npm are prerequisites for OpenSpec. Detect and report a missing or unsupported Node.js installation with its remediation; never install Node.js automatically.
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

Return detected state, dependency approvals, the chosen constitution values, and the exact bootstrap command for the conductor to run. The private runtime installs approved dependencies, then initializes OpenSpec and local workflow state atomically.
