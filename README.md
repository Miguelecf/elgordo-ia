# ElGordo IA

**Plan fat. Execute thin. Test without mercy.**

ElGordo IA is a human-led engineering workflow for OpenCode. One agent plans, another implements, and another tries to break the result. The engineer owns product judgment, architecture, and every expensive gate.

## Install

Supported in v0.1.0: macOS, Linux, and WSL.

```bash
curl -fsSL https://raw.githubusercontent.com/Miguelecf/elgordo-ia/v0.1.0/scripts/install.sh | sh
```

The installer:

1. Downloads the release artifact for your OS and architecture.
2. Verifies its published SHA-256 checksum for transfer integrity.
3. Checks OpenCode and Engram.
4. Asks before installing Engram when it is missing.
5. Installs managed agents, skills, and `/eg` globally.
6. Adds Context7 without replacing unrelated OpenCode configuration.

It never uses `sudo`. The binary is installed in `${ELGORDO_BIN_DIR:-$HOME/.local/bin}`.

The checksum is downloaded from the same GitHub release and does not provide independent signature verification. Signed provenance is planned after v0.1.0.

Restart OpenCode after installation.

## Quick Path

```bash
cd your-project
elgordo init
opencode
```

Then run:

```text
/eg add filtering to the order screen
```

`/eg` asks one necessary question at a time, inspects the repository and Engram before asking derivable questions, and routes exactly one phase agent according to CLI state.

## Workflow

```text
Planner -> Human plan gate -> Executor -> Human code gate -> QA -> Human final gate
```

Failures return explicitly:

- Implementation defect -> Executor.
- Broken scope or architecture -> Planner with a new plan revision.

The approved plan is sealed by its exact SHA-256 bytes. Executor, QA, and final transitions fail if it changes.

## Model Choice

ElGordo intentionally does not hardcode models. Select one with OpenCode `/models` before each phase.

| Phase | Initial suggestions |
|---|---|
| Planner | GPT-5.6 Sol, Kimi K3 |
| Executor | DeepSeek V4 Pro/Flash, MiMo 2.5, Kimi K2.7 Code |
| QA | GPT-5.6 Terra, Kimi K3, MiMo 2.5 Pro |

The workflow and evidence contract should make model capability less decisive than engineering context and human judgment.

## Local State

`elgordo init` creates local state and excludes it through `.git/info/exclude`:

```text
.elgordo/changes/<change>/
├── intent.md
├── plans/0001.md
├── execution.md
├── qa.md
├── state.json
└── events.jsonl
```

It also creates `.engram/config.json` so Engram resolves the repository deterministically and `.atl/skill-registry.md` as a delegator-only index of available project and user skills. All three directories are excluded through `.git/info/exclude` and are not committed by default.

The registry prefers project skills over duplicate user skills, omits internal `sdd-*`, `_shared`, and `skill-registry` entries, and stores exact `SKILL.md` paths rather than generated summaries. Refresh it after skill changes:

```bash
elgordo skill-registry refresh --force
```

## Human Gates

The conversational `/eg` entrypoint explains each gate. The underlying commands remain explicit:

```bash
elgordo plan hash
elgordo plan seal --expect sha256:<hash>
elgordo plan replan --reason "approved scope must change"
elgordo code approve
elgordo code reject --route execution --reason "..."
elgordo final approve
```

OpenCode permission prompts are the v0.1.0 human-presence boundary. They are not cryptographic identity proof.

## Operations

```bash
elgordo status --json
elgordo skill-registry refresh --force
elgordo doctor
elgordo sync
elgordo uninstall
elgordo version
```

`uninstall` removes only unchanged ElGordo-managed assets. It preserves Engram, Context7, and any managed file modified by the user.

## Recovery

- Re-run `elgordo sync` after an interrupted asset update.
- Run `elgordo doctor` to detect missing dependencies or managed assets.
- ElGordo backs up replaced managed assets and the original OpenCode config under `~/.config/elgordo/backups/`.
- A workflow lock is reclaimed when its recorded process no longer exists; malformed lock directories are reclaimable after ten minutes.
- If an approved plan is edited accidentally, `elgordo plan replan --reason "..."` restores its sealed snapshot into a new revision.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/elgordo
```

See [`docs/v0.1.0-contract.md`](docs/v0.1.0-contract.md) for the authority and scope contract.

## License

MIT
