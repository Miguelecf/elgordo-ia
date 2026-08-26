## Context

ElGordo IA is a Go CLI with installer shell code and GoReleaser-based releases. Its existing CI validates unit tests, vet, build, and shell syntax only on `main` pushes and pull requests.

## Decisions

- Keep macOS and Linux `make check` coverage because both are supported installation hosts.
- Run race detection and coverage once on Ubuntu to limit runner cost.
- Run a GoReleaser snapshot in CI to exercise archive configuration without publishing mutable dev artifacts.
- Keep `dev` installation source-based; no mutable dev release channel is introduced.
- Require release tag ancestry from `main` and repeat quality checks before GoReleaser publishes.
- Use workflow-level concurrency to cancel obsolete CI runs, but never cancel a release run.

## Test Strategy

This is CI configuration and documentation rather than product behavior, so strict TDD is not applicable. Replacement verification runs every Make target, validates OpenSpec, validates workflow syntax through GitHub Actions CI, and validates GoReleaser through its snapshot job.

## Risks

- Snapshot packaging adds CI time in exchange for detecting release misconfiguration before a tag.
- Branch protection and tag protection are repository settings; the documentation makes their required configuration explicit.
