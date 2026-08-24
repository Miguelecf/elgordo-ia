<p align="center">
  <img src="docs/elgordoanime.png" alt="ElGordo IA mascot" width="220">
</p>

# ElGordo IA

**Plan fat. Execute thin. Test without mercy.**

ElGordo IA is a harness/workflow layer mounted on OpenCode. OpenCode owns the user experience; ElGordo IA owns orchestration, workflow state, gates, and plan integrity. The engineer owns product judgment, architecture, and every approval gate.

> **En pocas palabras:** El Gordo IA convierte el agente principal de OpenCode en un orquestador de desarrollo. Tú sigues trabajando dentro de OpenCode y conversas con `elgordo-ia`; por debajo, el sistema organiza la planificación, la implementación, las revisiones y las aprobaciones humanas.

## Conoce ElGordo IA

Consulta la guía visual del proyecto para entender el flujo completo, los roles y la relación entre OpenCode y El Gordo IA:

**[Abrir la guía visual de ElGordo IA](https://miguelecf.github.io/elgordo-ia/)**

ElGordo IA no reemplaza OpenCode, no crea otra terminal y no te obliga a aprender una colección de comandos nuevos. OpenCode conserva la interfaz, la sesión, los modelos, las herramientas y la conversación. El Gordo IA añade la capa de workflow: decide cuándo planificar, pedir aprobación, ejecutar, revisar, probar o volver a una fase anterior.

## Descargar ElGordo IA

### Usar la versión estable

Para la mayoría de usuarios, instala la última versión publicada:

```bash
curl -fsSL https://raw.githubusercontent.com/Miguelecf/elgordo-ia/v0.1.0/scripts/install.sh | sh
```

### Descargar la rama `dev`

La rama `dev` contiene el trabajo más reciente y puede cambiar antes de una nueva versión estable. Para descargarla:

```bash
git clone --branch dev --single-branch https://github.com/Miguelecf/elgordo-ia.git
cd elgordo-ia
```

Si ya tienes el repositorio descargado:

```bash
git fetch origin
git switch dev
git pull --ff-only origin dev
```

Para comprobar en qué rama estás:

```bash
git branch --show-current
```

Debe mostrar:

```text
dev
```

## Install

Supported in v0.1.0: macOS, Linux, and WSL.

```bash
curl -fsSL https://raw.githubusercontent.com/Miguelecf/elgordo-ia/v0.1.0/scripts/install.sh | sh
```

The installer:

1. Downloads the release artifact for your OS and architecture.
2. Verifies its published SHA-256 checksum for transfer integrity.
3. Installs the private ElGordo runtime plus managed OpenCode assets.
4. Configures OpenCode so `elgordo-ia` is the primary agent.
5. Preserves unrelated OpenCode configuration and backs up any replaced value.
6. Adds Context7 without replacing unrelated OpenCode configuration.

It never uses `sudo`. The runtime is installed privately under the ElGordo config area, not on `PATH`.

Start or restart OpenCode after installation.

OpenSpec remains the versioned artifact engine for workflow changes. It is bootstrapped lazily when the workflow first needs it.

## Quick Path

```bash
opencode
```

When OpenCode opens, `elgordo-ia` is the primary agent. Just describe the change you want in plain language.

## Agent And Skill Contracts

The installed source of truth is embedded in this repository:

```text
internal/installer/assets/opencode/agents/elgordo-ia.md
internal/installer/assets/opencode/agents/eg-*.md
internal/installer/assets/opencode/skills/<skill>/SKILL.md
```

Agent-to-agent calls are declared in each agent frontmatter under `permission.task`. Agent-to-skill calls are declared under `permission.skill`; the conductor passes exact `SKILL.md` paths selected from `.atl/skill-registry.md`. Installer tests verify that every declared agent and skill reference resolves and that every embedded skill has a consumer. OpenCode loads these definitions globally after installation.

The workflow bootstraps lazily. If a repository has no ElGordo state yet, the primary agent prepares it on demand.

## Codebase Tour

ElGordo is workflow code plus Markdown agent contracts. The Go runtime enforces the state machine and seals; the OpenCode assets define the orchestration roles. Read it in layers:

```text
cmd/elgordo/main.go            Private runtime entrypoint
internal/cli/cli.go            Deterministic workflow protocol and gates
internal/workflow/             Domain logic: state machine, plan seal, skill registry
internal/installer/            Install, sync, uninstall, backups, global lock
internal/installer/assets/opencode/
  agents/elgordo-ia.md         Primary conductor agent
  agents/eg-*.md               Hidden workflow roles
  skills/*/SKILL.md            Embedded skills consumed through .atl/skill-registry.md
openspec/changes/<slug>/       Versioned proposal, specs, design, tasks (committed)
.elgordo/changes/<slug>/       intent, plans, execution, qa, state.json (local only)
scripts/install.sh             The curl installer entrypoint
```

Two parallel worlds confuse newcomers; keep them apart:

| Directory | What it holds | Committed |
|---|---|---|
| `openspec/changes/<slug>/` | Versioned artifacts: proposal, specs, design, tasks | Yes |
| `.elgordo/changes/<slug>/` | Operational state: plans, reports, `state.json`, `events.jsonl` | No |

Mechanisms worth understanding before touching anything:

- **Explicit state machine**: `PLANNING -> PLAN_REVIEW -> EXECUTING -> CODE_REVIEW -> QA -> FINAL_REVIEW -> DONE`, defined in `internal/workflow/state.go`. Prompts never advance phases; the runtime does.
- **Plan seal**: `internal/workflow/seal.go` builds a deterministic SHA-256 manifest over the plan and every OpenSpec Markdown artifact. Any post-seal edit blocks transitions until replanning.
- **Role isolation**: each agent declares its writable paths in frontmatter. Contract tests in `internal/installer/installer_test.go` verify that the conductor, planner, executor, and QA stay within scope.

### Contributor Path

1. Read `AGENTS.md`, this README, and [`docs/v0.1.0-contract.md`](docs/v0.1.0-contract.md).
2. Run `go test ./...`; read `internal/workflow/store_test.go` first because the tests document intended behavior better than prose.
3. Trace one simple command end to end: `main.go` -> `cli.Run` -> `app.status` -> `workflow.Store`.
4. Treat `internal/installer/assets/opencode/` as product code. Changing an agent or skill requires passing the installer contract tests.

## Workflow

```text
Questioner -> Planner -> Human plan gate -> Executor -> Human code gate -> QA -> Human final gate
```

Failures return explicitly:

- Implementation defect -> Executor.
- Broken scope or architecture -> Planner with a new plan revision.

`elgordo-ia` owns routing and may run bounded short iterations within a phase. It never crosses a review phase. `eg-questioner` only asks questions; `eg-planner`, `eg-executor`, and `eg-qa` return blockers to the conductor. Plan, code, and final approval are human gates.

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

ElGordo workflow state lives in `.elgordo/` and is excluded through `.git/info/exclude`:

```text
.elgordo/changes/<change>/
├── intent.md
├── plans/0001.md
├── execution.md
├── qa.md
├── state.json
└── events.jsonl
```

It also creates `.engram/config.json` when needed and `.atl/skill-registry.md` as a delegator-only index of available project and user skills. `.elgordo/`, `.engram/`, and `.atl/` are local and excluded from Git by default.

The registry prefers project skills over duplicate user skills, omits internal `sdd-*`, `_shared`, and `skill-registry` entries, and stores exact `SKILL.md` paths rather than generated summaries.

## Human Gates

OpenCode prompts the human at the workflow gates. The runtime verifies the transitions, seals, and state; the user stays in OpenCode.

## Quality Policy

- Use strict TDD for behavior changes unless an explicit exception is recorded.
- Resolve architecture authority in this order: human, repository docs and `AGENTS.md`, ADRs, code, Engram, generic skills.
- Name branches `feat|fix|refactor|docs|test|chore/<slug>`.
- Make one Conventional Commit per complete work unit, including its code, tests, and documentation.
- Treat a 400 LOC pull request as a warning to propose a split, not a universal blocking rule.
- A human approves every push and pull request.

OpenSpec scenarios use Given/When/Then Markdown. `eg-gherkin-verification` maps every scenario to an executed test or explicit manual evidence. Create executable `.feature` files only when the repository already has a BDD runner or the human explicitly chooses one.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/elgordo
```

### Install From Source

Building locally from `dev` produces the runtime used by the installer. Esta es la opción recomendada para probar los cambios más recientes de la rama de desarrollo:

```bash
git clone --branch dev --single-branch https://github.com/Miguelecf/elgordo-ia.git
cd elgordo-ia
go build -o elgordo ./cmd/elgordo
./elgordo install   # installs the private runtime and OpenCode assets
```

Después, abre o reinicia OpenCode:

```bash
opencode
```

`elgordo-ia` aparecerá como el agente principal/orquestador. Selecciónalo y describe normalmente lo que quieres construir. No necesitas ejecutar `/eg` ni iniciar una sesión paralela fuera de OpenCode.

Para conocer la autoridad, los límites y el contrato técnico, consulta [`docs/v0.1.0-contract.md`](docs/v0.1.0-contract.md).

## License

MIT
