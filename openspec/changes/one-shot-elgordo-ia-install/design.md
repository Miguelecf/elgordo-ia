## Context

ElGordo IA is a workflow harness mounted on OpenCode, not a competing CLI. The runtime may remain internal, but the user experience must be OpenCode-first: install once, open OpenCode, and talk to `elgordo-ia`.

## Goals / Non-Goals

### Goals

- Make `elgordo-ia` the primary OpenCode agent after installation.
- Remove `/eg` and manual `elgordo init` from the happy path.
- Keep the deterministic workflow state machine, seals, and role isolation.
- Preserve user-owned OpenCode configuration and restore previous values when needed.

### Non-Goals

- Replacing OpenCode.
- Introducing a new terminal UI.
- Building host adapters for other harnesses in v0.1.0.

## Decisions

- Use OpenCode `default_agent: elgordo-ia` for activation.
- Keep the workflow runtime private and hidden from the user.
- Configure OpenCode lazily for workflow dependencies only when needed.
- Remove `/eg` from the public happy path.

## Risks / Trade-offs

- The runtime still exists internally, so prompts and permissions must track the private path precisely.
- OpenCode config merges must not overwrite unrelated user settings.
- The first bootstrap path must stay idempotent.

## Migration Plan

1. Rename the primary conductor asset to `elgordo-ia`.
2. Hide the internal role agents.
3. Remove the `/eg` command asset.
4. Set `default_agent` to `elgordo-ia` with backup/restore for conflicts.
5. Update installer copy and docs.
6. Add tests for lazy install, migration, and uninstall restore.

## Open Questions

- Should future host adapters use the same core workflow runtime or a thinner OpenCode-native layer?
