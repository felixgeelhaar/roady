# Roady Integrations: Jira & Linear Architecture

## Overview

Moving beyond `roady-plugin-mock`, this document outlines the architecture for "Real World" syncers. The goal is to allow Roady to remain the **Source of Truth for Planning** while allowing **Execution** to happen in tools like Linear or Jira.

## 1. The "Link" Concept

Roady currently tracks task execution in `.roady/state.json`. To support integrations, we must enrich the `TaskResult` model to store references to external systems.

### Updated `state.json` Schema Proposal

```json
{
  "task_states": {
    "task-core-foundation": {
      "status": "in_progress",
      "owner": "felix",
      "external_refs": {
        "linear": {
          "id": "e84910-...",
          "identifier": "LIN-123",
          "url": "https://linear.app/roady/issue/LIN-123",
          "last_synced_at": "2024-01-13T10:00:00Z"
        }
      }
    }
  }
}
```

## 2. Plugin Interface Upgrade

The current `Syncer` interface is too simple (`Sync(plan, state) -> updates`). We need a richer protocol to handle the initial creation of tickets and bi-directional linking.

### Proposed Interface

```go
type Syncer interface {
    // Init ensures the plugin can connect (auth check)
    Init(config map[string]string) error

    // Sync performs the bi-directional synchronization
    // 1. Pushes new Roady tasks to External System (if missing)
    // 2. Pulls status updates from External System to Roady
    Sync(plan *planning.Plan, state *planning.ExecutionState) (*SyncResult, error)
}

type SyncResult struct {
    // StatusUpdates: TaskID -> NewStatus (e.g., "done")
    StatusUpdates map[string]planning.TaskStatus
    
    // LinkUpdates: TaskID -> ExternalRef (e.g., newly created Linear IDs)
    LinkUpdates   map[string]ExternalRef
    
    // Errors: Non-fatal errors encountered during sync
    Errors        []string
}
```

## 2. AI Configuration

Roady records provider/model defaults in `.roady/ai.yaml`. If a file doesn’t exist,
the CLI falls back to `ollama` with `llama3`. Environment variables (`ROADY_AI_PROVIDER`
and `ROADY_AI_MODEL`) override either location.

```yaml
provider: ollama
model: qwen3:8b
```

Use `roady ai configure` to update this file alongside `.roady/policy.yaml`, and keep
`policy.yaml` focused on limits (`max_wip`, `allow_ai`, `token_limit`). All MCP transports
and CLI commands read from the same wiring layer to stay in sync.

## 3. Implementation Strategy: Linear (The Pathfinder)

We will implement **Linear** first because its API is modern, fast, and strict, making it a better model for Roady's DAG than Jira's complex workflows.

### Configuration (`.roady/policy.yaml` or Env)

```yaml
integration:
  provider: "linear"
  config:
    team_id: "e849..."
    project_id: "b491..." # Optional: Sync to specific Linear Project
```

### Mapping Logic

| Roady State | Linear State | Action |
| :--- | :--- | :--- |
| `Pending` | (New) | **Create Issue** in Triage/Backlog |
| `In Progress` | `In Progress` | No Change |
| `In Progress` | `Done` | **Update Roady** to `Done` |
| `Done` | `In Progress` | **Update Linear** to `Done` (Roady is truth) |

## 4. Implementation Strategy: Jira (The Enterprise)

Jira requires more configuration due to custom workflows.

### Configuration

```yaml
integration:
  provider: "jira"
  config:
    domain: "acme.atlassian.net"
    project_key: "PROJ"
    # User must map generic Roady states to specific Jira transitions ID
    status_map:
      in_progress: "31" # Jira Transition ID for "Start Progress"
      done: "41"        # Jira Transition ID for "Done"
```

## 5. Drift Detection Impact

The `DriftService` must be updated to check **Synchronization Drift**:
*   *Drift:* Task is `Done` in Roady but `In Progress` in Linear.
*   *Drift:* Task exists in Roady Plan but was deleted in Jira.

## 6. Execution Plan

1.  **Refactor Domain:** Update `TaskResult` struct in `internal/domain/planning/state.go` to support `ExternalRefs`.
2.  **Refactor Plugin:** Update `Syncer` interface in `internal/domain/plugin/interface.go`.
3.  **Implement `roady-plugin-linear`:**
    *   Use `machinebox/graphql` for Linear API.
    *   Implement creation and status polling.
4.  **Implement `roady-plugin-jira`:**
    *   Use `andygrunwald/go-jira`.
    *   Implement transition logic.
