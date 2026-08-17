# ElGordo IA

- Runtime: Go 1.24+, OpenCode, Engram, Node.js 20.19.0+, and the pinned OpenSpec CLI.
- Run `go test ./...`, `go vet ./...`, and `go build ./cmd/elgordo` before delivery.
- Keep each commit as one complete work unit with its tests and docs.
- Do not weaken CLI state or hash checks with prompt-only conventions.
- OpenCode assets live under `internal/installer/assets/opencode/` and omit models intentionally.

## Workflow Policy

- ElGordo is the sole orchestrator and gate authority; OpenSpec is the versioned artifact engine.
- `openspec/` is committed. `.elgordo/`, `.engram/`, and `.atl/` are local state excluded through `.git/info/exclude`.
- Seal the deterministic manifest of the plan and all Markdown OpenSpec artifacts; any later change requires replan.
- Architecture authority order: human, repository docs and `AGENTS.md`, ADRs, code, Engram, generic skills.
- Use strict TDD for behavior changes unless an explicit exception is recorded. Human approval is required for push and PR creation.
