# Repository Guidelines

## Project Structure & Module Organization
- `cmd/hint/main.go`: CLI entrypoint (`-init` flow, argument parsing, command execution handoff).
- `internal/ai/`: OpenAI-compatible chat client and response sanitization.
- `internal/config/`: config load/save and interactive initialization.
- `internal/ui/`: Bubble Tea TUI states (loading, retry, edit, danger confirmation).
- `internal/executor/`: shell execution and dangerous-command detection.
- `pkg/sysinfo/`: runtime environment detection (`GOOS`, distro, shell, cwd).
- Root files: `go.mod`, `go.sum`, `README.md`.

## Build, Test, and Development Commands
- `go mod tidy`: sync and clean module dependencies.
- `go build ./cmd/hint`: build the CLI binary.
- `go run ./cmd/hint -init`: initialize local config interactively.
- `go run ./cmd/hint "list files changed in last 3 days"`: run directly without building.
- `go test ./...`: run all tests (currently no test files; add tests with new features).
- `go vet ./...`: run static checks before opening a PR.

## Coding Style & Naming Conventions
- Follow standard Go formatting; run `gofmt ./...` before committing.
- Keep packages focused and small; prefer `internal/` for app-specific logic, `pkg/` only for reusable units.
- Use idiomatic Go naming: exported identifiers use `CamelCase` (`ConfigPath`, `SuggestCommand`), unexported helpers use `camelCase` (`legacyConfigPath`, `sanitize`).
- Keep comments concise and meaningful; avoid restating obvious code.

## Testing Guidelines
- Place tests next to source files, named `*_test.go`.
- Name tests as `TestXxx` and prefer table-driven tests for parsing, validation, and safety checks.
- Prioritize tests for config validation/path fallback logic, AI output sanitization, dangerous command detection, and UI state transitions where practical.
- Run `go test ./...` locally before push.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (observed pattern: `feat: ...`), e.g. `fix: handle empty AI response`.
- Keep commits scoped to one logical change.
- PRs should include what changed and why, how to test (`go test ./...` plus manual CLI/TUI steps), linked issue (if applicable), and terminal screenshots/GIFs for UI behavior changes.

## Security & Configuration Notes
- Never commit real API keys or local config files.
- Config is stored under user config directories (see `README.md`); preserve restricted file permissions.
- Treat command execution changes as high-risk and keep safety checks conservative.
