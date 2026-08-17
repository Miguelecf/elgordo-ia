---
name: eg-branch-naming
description: "Trigger: branch, git branch, git switch, worktree, protected branch. Create and switch branches with conventional names and protected-branch safety."
license: MIT
metadata:
  author: Miguelecf
  version: "0.2.0"
---
# ElGordo Branch Naming

## Hard Rules

- Format: `feat|fix|refactor|docs|test|chore/<slug>` — type matches the change's dominant work; slug matches the ElGordo change slug.
- Use `git branch` and `git switch`; never `git checkout`.
- Never commit on protected branches (`main`, `master`, `develop`, release branches). If on one, create the change branch first.
- Worktree aware: check `git status` for the current branch and worktree before creating anything; linked worktrees carry independent `.elgordo` state.
- One active change per working tree; the branch belongs to the change.

## Decision Gates

| Situation | Action |
|---|---|
| On protected branch | Create `type/<slug>` and switch. |
| Branch already exists for this change | Switch to it; do not recreate. |
| Wrong branch for the change | Stop and report; never silently commit. |

## Output Contract

Report the branch created or reused and its base. Creation and switching are human-approved bash actions.
