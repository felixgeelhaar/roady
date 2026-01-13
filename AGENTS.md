# Repository Guidelines

## Project Structure & Module Organization
- `cmd/`: CLI entrypoints (`cmd/roady`) and plugin binaries (`cmd/roady-plugin-*`).
- `pkg/`: Public library surface and core domains (spec, planning, drift, policy, plugins).
- `internal/`: Infrastructure and CLI wiring (e.g., `internal/infrastructure/cli`).
- `docs/`: Product docs and design artifacts (vision, PRD, roadmap, TDD).
- `dist/` and root binaries (`roady`, `roady-plugin-*`) are build outputs and should stay uncommitted.

## Build, Test, and Development Commands
- Build the CLI: `go build -v -o roady ./cmd/roady`
- Build a plugin: `go build -v -o roady-plugin-mock ./cmd/roady-plugin-mock`
- Run tests: `go test -v -coverprofile=coverage.out ./...`
- Run all tests quickly: `go test ./...`

## Coding Style & Naming Conventions
- Go formatting is standard `gofmt` (tabs for indentation); run `gofmt -w` on touched files.
- Package names are lowercase; exported identifiers are `CamelCase`.
- Keep code and comments ASCII-only (repo history removed emojis/non-ASCII).

## Testing Guidelines
- Tests use the Go `testing` package and live alongside code as `*_test.go`.
- Prefer table-driven tests for domain logic in `pkg/` and CLI coverage in `internal/`.
- Include coverage output when updating tests: `coverage.out` is the repo convention.

## Commit & Pull Request Guidelines
- Follow Conventional Commits seen in history: `feat:`, `refactor:`, `docs:`, `build:`, `ci:`, `chore:`.
- Add task sync markers when applicable: `git commit -m "Subject [roady:task-id]"`.
- PRs should include a clear description, linked issue (if any), and the tests you ran.
- For CLI or docs changes, note any user-visible behavior updates.

## Security & Configuration Tips
- Local Roady state lives in `.roady/` (policy and execution state); avoid committing generated state files.
- CI uses Go 1.25 and runs `go test -coverprofile=coverage.out ./...` (see `.github/workflows/ci.yml`).
