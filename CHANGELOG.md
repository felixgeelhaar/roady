# Changelog

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
