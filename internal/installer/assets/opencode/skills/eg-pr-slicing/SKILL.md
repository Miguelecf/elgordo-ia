---
name: eg-pr-slicing
description: "Trigger: PR size, pull request, chained PRs, stacked PRs, review slice. Keep PRs reviewable and slice oversized changes."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo PR Slicing

## Hard Rules

- Recommend at most ~400 changed LOC per PR to protect review focus.
- Over the threshold: warn and propose chained PRs — never hard-fail unless the project constitution says so.
- Slice along work-unit boundaries; every slice is green and reviewable alone.
- Push and PR creation always require explicit engineer approval; the executor never pushes.

## Slicing Strategy

| Shape | Slice by |
|---|---|
| Sequential behavior | Dependency-ordered work units. |
| Broad mechanical change | Mechanical first, behavioral after. |
| Mixed refactor and feature | Refactor PR, then feature PR. |

## Chained PRs

Each chain link: own branch off the previous link, own checks, own review. Report chain order and rebase needs to the engineer; never rebase without approval.

## Output Contract

Report estimated LOC, recommended slices or chain, and rationale. The engineer decides slice boundaries and when anything is pushed.
