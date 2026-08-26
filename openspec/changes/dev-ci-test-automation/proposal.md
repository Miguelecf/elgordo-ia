## Why

Direct `dev` pushes currently bypass CI, releases do not independently repeat quality checks, and contributors must remember several test commands manually.

## What Changes

- Run CI on `dev` and `main` pushes plus pull requests.
- Add shared local Make targets for formatting, module hygiene, tests, vet, build, race detection, coverage, and shell syntax.
- Validate a GoReleaser snapshot in CI and quality-gate release tags from `main`.
- Document the branch, CI, release, and GitHub protection workflow.

## Capabilities

- Fast feedback for active `dev` development.
- Reproducible local verification.
- Release validation before public artifact publication.

## Impact

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `Makefile`, `.gitignore`, `docs/ci-cd.md`, and `README.md`
