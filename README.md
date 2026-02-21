<p align="center">
  <img src="logo.svg" width="150" alt="Roady Logo">
</p>

[![Go Version](https://img.shields.io/github/go-mod/go-version/felixgeelhaar/roady?logo=go)](https://github.com/felixgeelhaar/roady)
[![Coverage](https://img.shields.io/badge/coverage-82%25-brightgreen?logo=coveralls)](https://github.com/felixgeelhaar/roady/actions)
[![Release](https://img.shields.io/github/v/release/felixgeelhaar/roady?include_prereleases&logo=github)](https://github.com/felixgeelhaar/roady/releases/latest)
[![nox Security](https://img.shields.io/badge/nox-A-brightgreen?logo=lock)](https://github.com/felixgeelhaar/roady/security)
[![nox Scan](https://img.shields.io/badge/scan-0%20findings-brightgreen)](https://github.com/felixgeelhaar/roady/security)

# Roady

**Roady** is a planning-first system of record for software work. It acts as a durable, high-integrity memory layer between **intent** (what you want to build), **plans** (how you'll build it), and **execution** (the actual work).

Designed for individuals, teams, and AI agents, Roady ensures that your development roadmap never drifts from your original intent.

## Key Features

*   **Spec-Driven Inference:** Automatically derive functional specifications from multiple markdown documents (`roady spec analyze`).
*   **Adaptive AI Planning:** Decompose high-level features into granular task graphs using OpenAI or local Ollama models (`roady plan generate --ai`).
*   **Deterministic Drift Detection:** Instantly catch misalignments between docs, plans, and code reality (`roady drift detect`).
*   **Organizational Intelligence:** Discover projects across your machine (`roady discover`) and get unified progress views with aggregated metrics (`roady org status --json`). Shared policy inheritance lets org-level defaults cascade to projects (`roady org policy`), and cross-project drift detection aggregates issues across all repos (`roady org drift`).
*   **AI Governance:** Enforce policy-based token limits to control agentic spending.
*   **Event-Sourced Audit:** Every action is an immutable domain event with hash-chain integrity. Live handlers react to task transitions, drift warnings, and plan changes in real time (`roady audit verify`).
*   **Realtime Event Streaming:** Server-Sent Events endpoint streams live events to clients with type filtering and reconnection support.
*   **fsnotify File Watching:** Efficient OS-level file monitoring with configurable debounce, selective include/exclude glob patterns (`--include "*.md" --exclude "*.tmp"`), and a `--reconcile` flag for full auto-sync workflows (`roady watch docs/ --reconcile`).
*   **Pluggable Messaging:** Webhook and Slack adapters with a factory registry for event notifications (`roady messaging add/list/test`).
*   **Plugin Registry & Health:** Register, validate, and monitor syncer plugins with health checks (`roady plugin list/register/validate/status`).
*   **Outgoing Webhook Notifications:** HMAC-SHA256 signed webhook delivery with retry and dead-letter queue (`roady webhook notif add/list/test`).
*   **Plugin Contract Testing:** Automated contract test suite validates plugins conform to Syncer interface semantics.
*   **Continuous Automation:** Watch documents for changes and sync task statuses via Git commit markers (`[roady:task-id]`).
*   **Interactive TUI:** Real-time visibility into your project's health and velocity (`roady dashboard`).
*   **Interactive D3 Visualizations:** Rich, browser-based charts embedded in MCP apps — donut charts for status breakdowns, force-directed DAGs for plan and dependency graphs, gauges for usage and compliance, bar charts for drift severity, line charts for debt trends, and tree diagrams for spec hierarchies.
*   **Billing & Cost Tracking:** Define hourly rates with multi-currency and tax support, log time on task transitions, and generate cost reports with estimated-vs-actual variance analysis (`roady cost report`, `roady rate add`, `roady cost budget`).
*   **Quantified Debt Analysis:** Score and classify recurring drift as planning debt with sticky thresholds, category breakdowns, and historical trend analysis (`roady debt report`, `roady debt score`, `roady debt trend`).
*   **Team & Workspace Sync:** Manage team members with role-based access (admin/member/viewer) and synchronize `.roady/` state across collaborators via Git with conflict detection (`roady team add`, `roady workspace push/pull`).
*   **Cross-Repo Dependencies:** Declare runtime, build, and data dependencies between repositories, visualize the graph, detect cycles, and scan health (`roady deps add`, `roady deps graph`, `roady deps scan`).
*   **Smart AI Workflows:** Codebase-aware task decomposition, AI-suggested priority rebalancing, and natural language queries about project state (`roady plan smart-decompose`, `roady plan prioritize`, `roady query`).
*   **MCP First:** Seamlessly expose planning capabilities to AI agents via the Model Context Protocol.

## Quick Start

### 1. Installation
**Homebrew (macOS/Linux):**
```bash
brew install felixgeelhaar/tap/roady
```

**Alternative (Go):**
```bash
go install github.com/felixgeelhaar/roady/cmd/roady@latest
```

### 2. Initialize
```bash
roady init my-awesome-project
```

### 3. Plan your Intent
Put your PRDs or feature docs in `docs/`, then:
```bash
roady spec analyze docs/ --reconcile
roady plan generate --ai
```

### 4. Drive Execution
```bash
roady task start <task-id>
# Or automate via Git:
git commit -m "Implement core engine [roady:task-core-engine]"
roady git sync
```

### 5. Check Health & Forecast
```bash
roady status
roady drift detect
roady forecast
```

### 6. Track Costs
```bash
roady rate add --id dev --name "Developer" --rate 120
roady cost report
roady cost budget
```

## Governance & Policy

Configure project guardrails in `.roady/policy.yaml`:

```yaml
max_wip: 3            # Limit concurrent tasks
allow_ai: true        # Enable AI planning
token_limit: 50000    # Hard budget for AI operations
```

Org-level defaults can be set in `.roady/org.yaml` and inherited by all projects:

```yaml
name: my-org
shared_policy:
  max_wip: 5
  allow_ai: true
  token_limit: 100000
```

Project-level values override org defaults. View the merged result with `roady org policy`.

Configure AI provider defaults in `.roady/ai.yaml`:

```yaml
provider: openai   # Use OpenAI or Ollama
model: gpt-4o      # Your preferred model
```

See `docs/ai-configuration.md` for the latest policy/provider split and governance logging details.

Plan lifecycle transitions (generate/update/approve) are logged in `.roady/events.jsonl` so governance can trace when the spec-plan pair changed; the release workflow and MCP approval steps rely on those entries to show when drift was accepted.

## Release Automation

Use `scripts/release.sh` to run coverage recording, tests, AI plan generation, and the Relicta release workflow in a single, repeatable script:

```bash
./scripts/release.sh
```

Ensure `coverctl` and `relicta` are installed before running the script. It regenerates the plan with `--ai` so you can ship Roady and the MCP tools together.

## AI Integration (MCP)

Roady is a first-class MCP server. Add it to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "roady": {
      "command": "roady",
      "args": ["mcp"]
    }
  }
}
```

Run over HTTP or WebSocket when needed:

```bash
roady mcp --transport http --addr :8080
roady mcp --transport ws --addr :8080
```

See `docs/mcp-guide.md` for the complete MCP documentation, including all available tools and example workflows.

## Architecture

Roady is built on clean **Domain-Driven Design (DDD)** principles:
*   **Domain:** Pure business logic for Specs, Plans, Drift, Policy, and Domain Events. Value objects (`TaskStatus`, `TaskPriority`, `ApprovalStatus`) enforce transitions. An `EventDispatcher` routes events to handlers (logging, drift warnings, task transitions) and projections (velocity, state, audit timeline).
*   **Infrastructure:** Modern Go stack using `cobra`, `bubbletea`, `mcp-go`, and `fortify`. Pluggable messaging adapters (webhook, Slack) via factory registry. SSE handler for realtime event streaming. MCP apps built with Vue 3 + D3.js, compiled to self-contained HTML files via Vite.
*   **Storage:** Git-friendly YAML/JSON artifacts in `.roady/`. Events stored as hash-chained JSONL via `FileEventStore` with `InMemoryEventPublisher` for live subscriptions. Messaging config in `.roady/messaging.yaml`. Billing data in `rates.yaml` and `time_entries.yaml`.

## License

MIT License. See `LICENSE` for details.
