# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.3] - 2026-02-08

### Fixed

- Propagate actual error details in `roady_transition_task` and `roady_assign_task` MCP handlers instead of returning generic messages ([#3](https://github.com/felixgeelhaar/roady/issues/3))
- Initialize task status to `pending` when `SetTaskOwner`, `AddEvidence`, or `SetExternalRef` creates a new entry in the task states map, preventing broken transitions on tasks assigned before their first status write

## [0.6.2] - 2026-02-03

### Fixed

- Fix undefined label in spec MCP app visualization (title vs name mismatch)
- Fix undefined `has_drift` field in drift MCP app (derive from issues array length)

## [0.6.1] - 2026-02-01

### Fixed

- Fix burndown chart dipping below zero on completed projects
- Enable Vue runtime compiler and remove TypeScript syntax from templates

## [0.6.0] - 2026-02-01

### Added

- Org-level dashboard aggregating status across multiple Roady projects
- fsnotify-based file watching replacing polling
- Plugin contract testing for Syncer interface
- Reliable webhook delivery with retry
- Org policy inheritance with child project overrides
- Cross-project drift detection
- Plugin registry with versioning and health monitoring
- Pluggable messaging adapters (Slack support)
- Realtime event streaming via SSE
- Selective watch patterns with include/exclude globs
- Auto-sync workflows on file changes
- Shell completions for bash/zsh/fish/powershell
- Structured CLI error types with actionable hints
- Interactive config wizard for `ai.yaml` and `policy.yaml`
- Guided onboarding with starter templates (minimal, web-api, cli-tool, library)
- Versioned MCP tool schema with backward-compatible evolution
- Public Go SDK client package (`pkg/sdk/`)
- OpenAPI 3.0 spec generation from MCP tools
- Typed SDK request/response helpers
- Task assignment with owner field
- Role-based access control (`team.yaml`)
- Optimistic locking for concurrent state modifications
- Git-based workspace sync (push/pull)
- AI spec quality review with completeness scoring
- AI priority suggestions from dependency analysis
- Context-aware task decomposition with codebase scanning
- Natural language query interface for project status
- Interactive D3.js MCP app visualizations (org, sync, forecast, git-sync, and 10 others)

### Fixed

- Move roady-sync example out of workflows to prevent CI execution

## [0.5.0] - 2026-01-30

### Added

- Event-sourced audit in production with `FileEventStore` and `InMemoryEventPublisher`
- Domain event dispatcher with `LoggingHandler`, `DriftWarningHandler`, and `TaskTransitionHandler`
- Live velocity projection subscribing to event publisher for real-time updates
- D3.js interactive visualizations across 10 MCP apps (donut, force-directed graph, arc gauge, horizontal bars, line chart, collapsible tree, swimlane)
- Vue 3 + D3.js app source with Vite build pipeline and `apps.go` embed directive

### Changed

- `InitService` and `DebtService` now accept `domain.AuditLogger` interface instead of concrete `*AuditService`
- `BaseEvent` includes `Action` field for backward-compatible JSON serialization
- `FileEventStore` defers directory creation to first write, avoiding interference with `IsInitialized()` checks
- Deduplicated `BuildAppServices` into shared `buildServicesWithProvider` helper

### Fixed

- User-friendly MCP errors replacing raw Go error strings
- FlexBool/FlexInt types accepting both native and string JSON values
- Embedded MCP app dist files committed for CI `go:embed` compatibility

## [0.4.1] - 2026-01-19

### Fixed

- Patch release with minor fixes (see [v0.4.0...v0.4.1](https://github.com/felixgeelhaar/roady/compare/v0.4.0...v0.4.1))

## [0.4.0] - 2026-01-16

### Added

- GitHub Actions CI integration
- Web dashboard for plan visualization
- Event sourcing for audit trail
- gRPC transport for plugin communication and MCP
- Push method on Syncer interface for bidirectional sync
- Notion, Asana, and Trello plugins
- Push support for Linear, GitHub, Jira, and mock plugins
- Interactive TUI for plugin configuration with auto-install
- Per-plugin configuration file support
- HTTP webhook server for real-time sync
- Marketing website migrated to Astro with Vue

### Changed

- MCP wiring refactored with split AI config
- Phase 3 and Phase 4 code quality improvements

### Fixed

- CI builds binary to `dist/` for e2e tests
- Website `.nojekyll` for GitHub Pages compatibility

## [0.3.0] - 2026-01-13

### Added

- Drift accept command for acknowledging intentional spec divergence

## [0.2.1] - 2026-01-13

### Changed

- Expose core packages in `pkg/` to enable library usage
- Inject version flags in goreleaser config

## [0.2.0] - 2026-01-13

### Added

- Jira plugin for bidirectional sync
- Linear plugin with external task linking
- External refs on task state for plugin integration
- Interactive dashboard (`roady dashboard`) TUI for visualizing plans and drift
- Dynamic policy engine with configurable `.roady/policy.yaml`
- Plugin architecture with gRPC-based Syncer interface
- Smart plan injection via `roady_update_plan` MCP tool
- GoReleaser for multi-platform builds and Homebrew distribution
- Release automation for GitHub and Homebrew

## [0.1.0] - 2026-01-13

### Added

- Core domain models: Spec, Plan, Task, Drift
- Spec ingestion from Markdown (`roady spec import`)
- Spec locking (`.roady/spec.lock.json`) for immutable planning boundaries
- Plan reconciliation merging new specs without destroying existing task state
- Drift detection for Spec vs Plan and Plan vs Code
- Audit trail with structured logging to `.roady/events.jsonl`
- MCP server (`roady mcp`) exposing core tools to AI agents
- Resilience via `fortify` integration for filesystem retries
- State management via `statekit` FSM for task transitions

[0.6.3]: https://github.com/felixgeelhaar/roady/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/felixgeelhaar/roady/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/felixgeelhaar/roady/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/felixgeelhaar/roady/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/felixgeelhaar/roady/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/felixgeelhaar/roady/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/felixgeelhaar/roady/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/felixgeelhaar/roady/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/felixgeelhaar/roady/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/felixgeelhaar/roady/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/felixgeelhaar/roady/releases/tag/v0.1.0
