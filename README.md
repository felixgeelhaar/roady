<p align="center">
  <img src="logo.svg" width="150" alt="Roady Logo">
</p>

# Roady

**Roady** is a planning-first system of record for software work. It acts as a durable, high-integrity memory layer between **intent** (what you want to build), **plans** (how you'll build it), and **execution** (the actual work).

Designed for individuals, teams, and AI agents, Roady ensures that your development roadmap never drifts from your original intent.

## Key Features

*   **Spec-Driven Inference:** Automatically derive functional specifications from multiple markdown documents (`roady spec analyze`).
*   **Adaptive AI Planning:** Decompose high-level features into granular task graphs using OpenAI or local Ollama models (`roady plan generate --ai`).
*   **Deterministic Drift Detection:** Instantly catch misalignments between docs, plans, and code reality (`roady drift detect`).
*   **Organizational Intelligence:** Discover projects across your machine (`roady discover`) and get unified progress views (`roady org status`).
*   **AI Governance:** Enforce policy-based token limits to control agentic spending.
*   **Immutable Audit Trails:** Every action is cryptographically signed in a verifiable hash chain (`roady audit verify`).
*   **Continuous Automation:** Watch documents for changes and sync task statuses via Git commit markers (`[roady:task-id]`).
*   **Interactive TUI:** Real-time visibility into your project's health and velocity (`roady dashboard`).
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

## Governance & Policy

Configure project guardrails in `.roady/policy.yaml`:

```yaml
max_wip: 3            # Limit concurrent tasks
allow_ai: true        # Enable AI planning
token_limit: 50000    # Hard budget for AI operations
```

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

## Architecture

Roady is built on clean **Domain-Driven Design (DDD)** principles:
*   **Domain:** Pure business logic for Specs, Plans, Drift, and Policy.
*   **Infrastructure:** Modern Go stack using `cobra`, `bubbletea`, `mcp-go`, and `fortify`.
*   **Storage:** Git-friendly YAML/JSON artifacts in `.roady/`.

## License

MIT License. See `LICENSE` for details.
