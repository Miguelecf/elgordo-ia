## Why

OpenCode-first one-shot installation of ElGordo IA

## What Changes

- The installer prepares OpenCode so `elgordo-ia` is the primary agent.
- The installer installs the private ElGordo runtime and managed OpenCode assets without requiring the user to run a separate workflow CLI.
- The happy path becomes `curl ... | sh` followed by `opencode`.
- The first workflow detects missing Engram or OpenSpec, requests explicit approval, and installs approved dependencies before creating project state.

## Capabilities

- OpenCode boots directly into the `elgordo-ia` orchestration mode.
- The workflow lazily bootstraps repository state on first use.
- Internal role agents remain isolated and hidden from the main UX.

## Impact

- `scripts/install.sh`
- `internal/installer/`
- `internal/installer/assets/opencode/`
- `README.md`, `docs/`, and the OpenSpec change tree
