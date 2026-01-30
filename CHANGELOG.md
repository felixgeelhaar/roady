# Changelog

Release 0.3.0

Release 0.4.1

## [0.5.0] - Domain Events Wired into Production

### Added
- **Event-Sourced Audit in Production:** `BuildAppServices` now creates `EventSourcedAuditService` with `FileEventStore` and `InMemoryEventPublisher`, replacing the plain `AuditService` for all wired services.
- **Domain Event Dispatcher:** `EventDispatcher` with `LoggingHandler`, `DriftWarningHandler`, and `TaskTransitionHandler` registered and active in production.
- **Live Velocity Projection:** `ExtendedVelocityProjection` subscribes to the event publisher, receiving live events in addition to startup hydration from stored events.
- **D3.js Interactive Visualizations:** 10 MCP apps with D3 charts — donut, force-directed graph, arc gauge, horizontal bars, line chart, collapsible tree, and swimlane views.
- **Vue 3 + D3.js App Source:** Full `app/` build pipeline with Vite, Vue 3, D3.js and `apps.go` embed directive for serving interactive visualizations from MCP tools.

### Changed
- `InitService` and `DebtService` now accept `domain.AuditLogger` interface instead of concrete `*AuditService`, enabling polymorphic audit implementations.
- `BaseEvent` includes `Action` field for backward-compatible JSON serialization with `domain.Event` readers.
- `FileEventStore` defers directory creation to first write, avoiding interference with `IsInitialized()` project checks.
- Deduplicated `BuildAppServices` and `BuildAppServicesWithProvider` into shared `buildServicesWithProvider` helper.

### Fixed
- **User-Friendly MCP Errors:** All MCP handler error messages replaced with actionable, human-readable messages instead of raw Go errors.
- **FlexBool/FlexInt Types:** MCP args now accept both native and string JSON values.
- Embedded MCP app dist files committed for CI `go:embed` compatibility.

## [0.4.3] - User-Friendly MCP Errors & App Source

### Added
- **Vue 3 + D3.js App Source:** Committed full `app/` build pipeline (Vite, Vue 3, D3.js) and `apps.go` embed directive for serving interactive visualizations from MCP tools.

### Fixed
- **FlexBool/FlexInt Types:** MCP args now accept both native and string JSON values, fixing `json: cannot unmarshal string` errors from clients that send `"true"` instead of `true`.
- **User-Friendly MCP Errors:** All MCP handler error messages replaced with actionable, human-readable messages instead of raw Go errors (e.g., "Failed to load plan. Generate a plan first." instead of internal stack traces).

## [0.4.2] - Interactive D3 Visualizations

### Added
- **Shared D3 Chart Library:** `app/src/lib/d3-charts.ts` with 6 reusable chart functions — donut, force-directed graph, arc gauge, horizontal bars, line chart, and collapsible tree.
- **Status App:** D3 donut chart for task status distribution with click-to-filter.
- **Plan App:** Force-directed DAG with nodes colored by status, directed dependency edges, drag and zoom.
- **State App:** Mini donut chart + swimlane columns (pending/in_progress/blocked/done).
- **Drift App:** Horizontal severity bar chart with click-to-filter issue list.
- **Debt App:** Multi-chart dashboard — health gauge (summary), category donut + component bars (report), line chart (trend).
- **Deps App:** Force-directed dependency network graph with edges colored by type and node sizing.
- **Usage App:** Token gauge arc (green→red) with threshold coloring.
- **Spec App:** Collapsible tree diagram (project → features → requirements).
- **Policy App:** Compliance gauge showing pass/violation state.
- **Git Sync App:** Vertical D3 timeline with colored dots per sync result.

### Changed
- Updated README with D3 visualization feature and architecture note.
- Updated marketing site MCP section with visualization showcase card.
- Updated marketing site features section with "Every Tool, Visualized" callout.

## [0.2.0] - Horizon 2: Increase Leverage

### Added
- **Interactive Dashboard:** `roady dashboard` TUI for visualizing plans and drift.
- **Dynamic Policy Engine:** Configurable `.roady/policy.yaml` (e.g., `max_wip`).
- **Plugin Architecture:** gRPC-based `Syncer` interface for external integrations.
- **Intelligent Drift:** Now detects empty files (`empty-code-*`) as high-severity drift.
- **Smart Plan Injection:** `roady_update_plan` MCP tool for AI-driven architecture.

## [0.1.0] - Horizon 1: Core Foundation

### Added
- **Core Domain Models:** Spec, Plan, Task, Drift.
- **Spec Ingestion:** `roady spec import` from Markdown.
- **Spec Locking:** `.roady/spec.lock.json` for immutable planning boundaries.
- **Plan Reconciliation:** Merging new specs without destroying existing task state.
- **Drift Detection:** Spec vs Plan and Plan vs Code existence checks.
- **Audit Trail:** Structured logging to `.roady/events.jsonl`.
- **MCP Server:** `roady mcp` exposing core tools to AI agents.
- **Resilience:** `fortify` integration for filesystem retries.
- **State Management:** `statekit` FSM for task transitions.
