# Roady

**Roady** is a planning-first system of record for software work. It helps individuals and teams answer "What are we building?", "Why?", and "What's next?" without re-deriving context.

It acts as a durable memory layer between **intent** (Specs), **plans** (Tasks), and **execution** (Code).

## Features

*   **Spec Management:** Import PRDs (`roady spec import`) or define them in `spec.yaml`.
*   **Plan Generation:** Deterministically derive execution plans with "Smart Injection" for AI agents.
*   **Drift Detection:**
    *   **Spec Drift:** Detect when the Plan falls behind the Spec.
    *   **Code Drift:** Detect when Code is missing or empty for completed tasks.
*   **Policy Enforcement:** Configurable governance (e.g., Max WIP limits) via `policy.yaml`.
*   **Interactive Dashboard:** A TUI (`roady dashboard`) for real-time status visibility.
*   **Plugin Architecture:** Sync tasks with external tools (GitHub, Linear) via gRPC plugins.
*   **AI Integration (MCP):** Expose all capabilities to AI agents via the Model Context Protocol.

## Installation

```bash
go install github.com/felixgeelhaar/roady/cmd/roady@latest
```

## Quick Start

1.  **Initialize a Project**
    ```bash
    roady init my-project
    ```
    Creates `.roady/` with default spec and policy.

2.  **Import Your Intent**
    ```bash
    roady spec import docs/prd.md
    ```

3.  **Generate a Plan**
    ```bash
    roady plan generate
    ```

4.  **Visualize**
    ```bash
    roady dashboard
    ```

5.  **Check Drift & Policy**
    ```bash
    roady drift detect
    roady policy check
    ```

## Configuration (`.roady/policy.yaml`)

Control your workflow constraints:

```yaml
max_wip: 3  # Limit concurrent "in_progress" tasks
```

## Plugins

Roady supports external syncers via `roady sync <plugin-path>`.
Plugins implement the `Syncer` gRPC interface to map Roady tasks to external issues.

## AI Integration (MCP)

**Claude Desktop Configuration:**
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

**Tools Available to Agents:**
*   `roady_update_plan`: The "Brain" of the operation. Agents read the spec and push a detailed DAG.
*   `roady_detect_drift`: Agents can self-correct if they forgot to implement a file.
*   `roady_check_policy`: Agents can check if they are allowed to start a new task.

## Architecture

Roady follows Domain-Driven Design (DDD):
*   **Domain:** Spec, Plan, Drift, Policy, Plugin.
*   **Infrastructure:** CLI (Cobra), TUI (Bubbletea), Storage (YAML/JSON), MCP (mcp-go).

Data is local-first in `.roady/`.

## Contributing

1.  Clone the repo.
2.  Run `go mod tidy`.
3.  Run `go test ./...` to verify changes.