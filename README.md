<p align="center">
  <img src="docs/elgordoanime.png" alt="ElGordo IA mascot" width="220">
</p>

# ElGordo IA

**Plan fat. Execute thin. Test without mercy.**

ElGordo IA is a human-led engineering workflow for OpenCode. ElGordo is the sole orchestrator and gate authority; OpenSpec is the versioned artifact engine. The engineer owns product judgment, architecture, and every approval gate.

## Install

Supported in v0.1.0: macOS, Linux, and WSL.

```bash
curl -fsSL https://raw.githubusercontent.com/Miguelecf/elgordo-ia/v0.1.0/scripts/install.sh | sh
```

The installer:

1. Downloads the release artifact for your OS and architecture.
2. Verifies its published SHA-256 checksum for transfer integrity.
3. Requires OpenCode to be installed.
4. Offers Engram when it is missing.
5. Requires Node.js 20.19.0 or newer, but never installs Node.js.
6. Offers the pinned OpenSpec CLI only after consent; use `--accept-openspec-install` for non-interactive consent.
7. Installs managed agents, skills, and `/eg` globally.
8. Adds Context7 without replacing unrelated OpenCode configuration.

It never uses `sudo`. The binary is installed in `${ELGORDO_BIN_DIR:-$HOME/.local/bin}`.

The checksum is downloaded from the same GitHub release and does not provide independent signature verification. Signed provenance is planned after v0.1.0.

Restart OpenCode after installation.

OpenSpec is installed through npm only as `@fission-ai/openspec@1.5.0`. `elgordo doctor` checks both Node.js and OpenSpec.

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

## Agent And Skill Contracts

The installed source of truth is embedded in this repository:

```text
internal/installer/assets/opencode/agents/<agent>.md
internal/installer/assets/opencode/skills/<skill>/SKILL.md
internal/installer/assets/opencode/commands/eg.md
```

Agent-to-agent calls are declared in each agent frontmatter under `permission.task`. Agent-to-skill calls are declared under `permission.skill`; the agent body also requires the conductor to pass exact `SKILL.md` paths selected from `.atl/skill-registry.md`. Installer tests verify that every declared agent and skill reference resolves and that every embedded skill has a consumer. OpenCode loads these definitions globally after installation; restart OpenCode after syncing assets.

`elgordo init` runs `openspec init --tools none`. This creates no `/opsx-*` commands or competing OpenSpec agents: `/eg` remains the sole human entrypoint.

## Workflow

```text
Questioner -> Planner -> Human plan gate -> Executor -> Human code gate -> QA -> Human final gate
```

Failures return explicitly:

- Implementation defect -> Executor.
- Broken scope or architecture -> Planner with a new plan revision.

The conductor (`eg`) owns routing and may run bounded short iterations within a phase. It never crosses a review phase. `eg-questioner` only asks questions; `eg-planner`, `eg-executor`, and `eg-qa` return blockers to the conductor. Plan, code, and final approval are human gates.

The plan seal covers a deterministic manifest of the plan and every Markdown OpenSpec artifact. Any post-seal modification requires a replan.

## Model Choice

ElGordo intentionally does not hardcode models. Select one with OpenCode `/models` before each phase.

| Phase | Initial suggestions |
|---|---|
| Planner | GPT-5.6 Sol, Kimi K3 |
| Executor | DeepSeek V4 Pro/Flash, MiMo 2.5, Kimi K2.7 Code |
| QA | GPT-5.6 Terra, Kimi K3, MiMo 2.5 Pro |

The workflow and evidence contract should make model capability less decisive than engineering context and human judgment.

## Artifacts And Local State

OpenSpec artifacts are committed:

```text
openspec/changes/<slug>/
├── proposal.md
├── specs/
├── design.md
└── tasks.md
```

`elgordo init` excludes local operational state through `.git/info/exclude`:

```text
.elgordo/changes/<change>/
├── intent.md
├── plans/0001.md
├── execution.md
├── qa.md
├── state.json
└── events.jsonl
```

It also creates `.engram/config.json` so Engram resolves the repository deterministically and `.atl/skill-registry.md` as a delegator-only index of available project and user skills. `.elgordo/`, `.engram/`, and `.atl/` are local and excluded from Git by default.

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

## Quality Policy

- Use strict TDD for behavior changes unless an explicit exception is recorded.
- Resolve architecture authority in this order: human, repository docs and `AGENTS.md`, ADRs, code, Engram, generic skills.
- Name branches `feat|fix|refactor|docs|test|chore/<slug>`.
- Make one Conventional Commit per complete work unit, including its code, tests, and documentation.
- Treat a 400 LOC pull request as a warning to propose a split, not a universal blocking rule.
- A human approves every push and pull request.

OpenSpec scenarios use Given/When/Then Markdown. `eg-gherkin-verification` maps every scenario to an executed test or explicit manual evidence. Create executable `.feature` files only when the repository already has a BDD runner or the human explicitly selects one.

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
