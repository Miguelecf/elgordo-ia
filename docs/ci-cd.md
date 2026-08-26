# CI/CD And Automated Testing

ElGordo IA uses GitHub Actions for continuous integration and release publishing. The workflows intentionally separate fast feedback for `dev` from immutable public releases.

## Branch Model

| Branch or reference | Purpose | Automation |
|---|---|---|
| `dev` | Day-to-day integration | CI feedback on every push; direct pushes are allowed for the sole developer. |
| `main` | Stable source | CI on every pull request; merges require the successful checks below. |
| `v*` tag on `main` | Immutable public release | Quality gate, package, and GitHub Release |

Never tag a commit outside `main`. The release workflow verifies this before publishing.

## Local Checks

Run these commands before opening a pull request:

```bash
make check
make race
make coverage
```

`make check` fails on unformatted Go code, an untidy module, unit-test failures, vet failures, build failures, or invalid installer shell syntax. `make race` runs the Go race detector. `make coverage` produces `coverage.out`, prints the function coverage summary, and keeps the profile locally for inspection; Git ignores it.

## Continuous Integration

`.github/workflows/ci.yml` runs for pushes and pull requests targeting `dev` and `main`.

| Job | Runner | Purpose |
|---|---|---|
| `test` | Ubuntu and macOS | Runs `make check` on both supported host families. |
| `quality` | Ubuntu | Runs race detection and coverage once to control CI cost. |
| `package` | Ubuntu | Builds a GoReleaser snapshot to validate cross-platform packaging without publishing artifacts. |

Newer commits cancel older in-progress CI runs for the same branch or pull request. This keeps feedback focused on the newest commit.

## Release Pipeline

`.github/workflows/release.yml` runs only when a `v*` tag is pushed. It:

1. Confirms the tag commit is reachable from `main`.
2. Runs `make check` and `make race`.
3. Invokes pinned GoReleaser to produce platform archives and `checksums.txt`.
4. Publishes the GitHub Release.

Create a release only after the intended commit is merged to `main`:

```bash
git switch main
git pull --ff-only origin main
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

The release installer must use the immutable tag URL documented in the README, never a moving branch URL.

## GitHub Repository Settings

Workflow files cannot enforce repository-level branch rules by themselves. This repository uses a solodev-friendly configuration:

1. Leave `dev` unprotected so the sole developer can push directly; CI still reports every push.
2. On `main`, require a pull request and these successful checks: `test (ubuntu-latest)`, `test (macos-latest)`, `quality`, and `package`.
3. On `main`, require zero approving reviews and do not require the branch to be up to date before merging.
4. On `main`, restrict direct pushes, force pushes, and branch deletion.
5. Keep the `v*` tag pattern protected; create release tags only after merge to `main`.

## Boundaries

CI validates source, packaging, and installer syntax. It does not perform a global OpenCode installation or automatically install Node.js/npm. Those operations remain explicit, consent-gated behavior of the installed runtime.
