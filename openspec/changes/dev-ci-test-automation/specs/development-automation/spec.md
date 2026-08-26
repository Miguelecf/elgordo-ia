## ADDED Requirements

### Requirement: Development branch validation
The repository SHALL run automated validation on every push to `dev` and `main`, and on pull requests targeting either branch. It SHALL cancel superseded runs for the same reference.

#### Scenario: A developer pushes to dev
- **WHEN** a commit is pushed to `dev`
- **THEN** CI runs formatting, module, unit-test, vet, build, and installer syntax checks
- **AND** a newer push cancels any older in-progress CI run for `dev`

### Requirement: Reproducible local quality checks
The repository SHALL provide documented Make targets for formatting, module hygiene, tests, vet, build, installer syntax, race detection, and coverage.

#### Scenario: A contributor runs the local quality gate
- **WHEN** the contributor runs `make check`
- **THEN** it fails if any baseline CI quality check fails
- **AND** the contributor can run race detection and coverage through separate documented targets

### Requirement: Release quality gate
The release workflow SHALL verify that a release tag is reachable from `main`, run baseline and race checks, and validate packaging before publishing release artifacts.

#### Scenario: A tag points outside main history
- **WHEN** a `v*` tag is pushed for a commit not reachable from `main`
- **THEN** the release workflow fails before GoReleaser publishes artifacts

#### Scenario: A tag points to main history
- **WHEN** a `v*` tag is pushed for a validated commit reachable from `main`
- **THEN** the workflow runs quality checks and publishes GoReleaser archives with checksums
