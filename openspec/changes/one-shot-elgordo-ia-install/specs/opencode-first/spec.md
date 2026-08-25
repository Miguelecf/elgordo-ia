## ADDED Requirements

### Requirement: Primary OpenCode entrypoint
The installer SHALL configure OpenCode so `elgordo-ia` is the primary agent. Hidden role agents SHALL remain isolated from the main user experience, and the normal start flow SHALL not require `/eg` or `elgordo init`.

#### Scenario: OpenCode starts in ElGordo mode
- **GIVEN** a fresh installation
- **WHEN** the user starts OpenCode
- **THEN** `elgordo-ia` is available as the primary agent
- **AND** the user can start a workflow conversation without typing a slash command

Verification: `internal/installer.TestInstallConfiguresOpenCodeDefaultsWithoutOptionalDependencies` and `internal/installer.TestEmbeddedAgentGraphAndSkillsStayRoleIsolated`.

### Requirement: Lazy workflow bootstrap
The workflow harness SHALL lazily bootstrap project state on the first workflow invocation. Before creating project state, it SHALL detect and prepare workflow dependencies. It SHALL request explicit approval before installing Engram or the pinned OpenSpec CLI, and SHALL not install Node.js or npm automatically.

#### Scenario: Installation does not bootstrap a repository
- **GIVEN** the user installs ElGordo IA before opening a repository workflow
- **WHEN** the installer configures the private runtime and OpenCode assets
- **THEN** it does not require Engram, Node.js, or OpenSpec to be installed
- **AND** it does not create ElGordo project state in a repository

Verification: `internal/installer.TestInstallConfiguresOpenCodeDefaultsWithoutOptionalDependencies`.

#### Scenario: Workflow bootstraps lazily
- **GIVEN** a repository with no ElGordo state
- **WHEN** the user asks `elgordo-ia` to begin work
- **THEN** the harness initializes the minimum required local workflow state on demand
- **AND** it returns to the user's original request

Verification: `internal/cli.TestProjectLifecycleCommands` and `internal/cli.TestInitStopsWhenOpenSpecInitializationFails`.

#### Scenario: Approved dependencies are installed during bootstrap
- **GIVEN** a repository with no ElGordo state, supported Node.js and npm, and missing Engram and OpenSpec
- **WHEN** the engineer explicitly approves both dependency installations
- **THEN** the harness installs and verifies Engram and `@fission-ai/openspec@1.5.0`
- **AND** initializes OpenSpec before creating ElGordo project state

Verification: `internal/installer.TestEnsureWorkflowDependenciesInstallsApprovedDependencies` and `internal/cli.TestInitBootstrapsApprovedDependenciesBeforeOpenSpec`.

#### Scenario: A missing or declined dependency does not initialize the project
- **GIVEN** a repository with no ElGordo state
- **WHEN** Node.js is unsupported or OpenSpec installation is declined
- **THEN** the harness reports an actionable error without installing Node.js automatically
- **AND** it does not create ElGordo project state

Verification: `internal/installer.TestEnsureWorkflowDependenciesRequiresSupportedNode`, `internal/installer.TestEnsureWorkflowDependenciesStopsWhenOpenSpecInstallIsDeclined`, and `internal/cli.TestInitLeavesProjectUninitializedWhenDependencyBootstrapFails`.

### Requirement: OpenCode configuration migration
The workflow runtime SHALL preserve user-owned OpenCode settings when they conflict. It SHALL remove the legacy `/eg` command asset from managed installation and restore the prior default agent when ElGordo is uninstalled.

#### Scenario: Existing default agent is restored on uninstall
- **GIVEN** OpenCode already has `default_agent: build`
- **WHEN** the installer sets `default_agent: elgordo-ia` and the user later uninstalls ElGordo IA
- **THEN** the installer preserves unrelated OpenCode settings while installed
- **AND** uninstallation restores `default_agent: build`

Verification: `internal/installer.TestConfigureOpenCodeDefaultsRestoresPreviousSelection`.

#### Scenario: Legacy slash command is not part of the happy path
- **GIVEN** a prior installation may have exposed `/eg`
- **WHEN** the installer installs the managed OpenCode assets
- **THEN** it removes the legacy `commands/eg.md` asset
- **AND** `/eg` is not required for the normal start flow

Verification: `internal/installer.TestInstallAssetsAndUninstall` and `internal/installer.TestEmbeddedAgentGraphAndSkillsStayRoleIsolated`.
