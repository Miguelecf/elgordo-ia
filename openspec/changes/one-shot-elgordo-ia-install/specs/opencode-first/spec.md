# OpenCode-First Harness

## Requirements

- The installer SHALL configure OpenCode so `elgordo-ia` is the primary agent.
- The installer SHALL not require the user to run `/eg` or `elgordo init` in the happy path.
- The workflow runtime SHALL remain private and SHALL preserve user-owned OpenCode settings when they conflict.
- The workflow harness SHALL lazily bootstrap project state the first time the user invokes the workflow in OpenCode.
- Hidden role agents SHALL remain isolated from the main UX.

## Scenarios

### Scenario: OpenCode starts in ElGordo mode
GIVEN a fresh installation
WHEN the user starts OpenCode
THEN `elgordo-ia` is available as the primary agent
AND the user can start a workflow conversation without typing a slash command

### Scenario: Existing default agent is preserved or restored
GIVEN OpenCode already has a different `default_agent`
WHEN the installer runs and the user declines replacement
THEN the existing default agent remains unchanged
AND the installer records the decision without breaking the installation

### Scenario: Legacy slash command is not part of the happy path
GIVEN the installed assets
WHEN the user inspects the available public workflow entrypoints
THEN `/eg` is not required for the normal start flow

### Scenario: Workflow bootstraps lazily
GIVEN a repository with no ElGordo state
WHEN the user asks `elgordo-ia` to begin work
THEN the harness initializes the minimum required local workflow state on demand
AND then continues with the user's original request
