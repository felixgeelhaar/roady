# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Roady is a planning-first system of record for software work. It acts as a durable memory layer between **intent** (specs), **plans** (task DAGs), and **execution** (state tracking). Designed for individuals, teams, and AI agents via MCP (Model Context Protocol).

## Build & Test Commands

```bash
# Build main binary
go build -o roady ./cmd/roady

# Build plugin binaries
go build -o roady-plugin-mock ./cmd/roady-plugin-mock
go build -o roady-plugin-github ./cmd/roady-plugin-github
go build -o roady-plugin-jira ./cmd/roady-plugin-jira
go build -o roady-plugin-linear ./cmd/roady-plugin-linear

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Run a single test
go test -run TestFunctionName ./path/to/package

# Run tests for a specific package
go test ./pkg/application/...
go test ./internal/infrastructure/cli/...

# Verbose test output
go test -v ./...
```

## Architecture

### Domain-Driven Design Structure

```
pkg/domain/           # Pure domain logic (no external dependencies)
├── spec/            # ProductSpec, Feature, Requirement entities
├── planning/        # Plan, Task, ExecutionState, DAG validation
├── drift/           # Issue, Report, drift detection types
├── policy/          # Policy rules (WIP limits, dependencies)
├── ai/              # Provider interface definition
└── plugin/          # Syncer interface for external integrations

pkg/application/      # Use-case services orchestrating domain logic
├── init_service.go
├── spec_service.go
├── plan_service.go
├── drift_service.go
├── policy_service.go
├── task_service.go
├── audit_service.go
├── ai_planning_service.go
├── git_service.go
└── sync_service.go

internal/infrastructure/  # Adapters and framework integrations
├── cli/             # Cobra CLI commands (root, init, spec, plan, drift, etc.)
├── mcp/             # MCP server implementation
├── config/          # Configuration loading (ai.yaml)
└── wiring/          # Service composition and dependency injection

pkg/storage/         # Filesystem repository (YAML/JSON in .roady/)
pkg/ai/              # AI provider implementations (Ollama, OpenAI, Anthropic, Gemini)
pkg/plugin/          # HashiCorp go-plugin loader for external syncers
```

### Key Dependencies

- **cobra**: CLI framework
- **bubbletea/lipgloss**: TUI dashboard
- **mcp-go**: MCP server protocol
- **statekit**: FSM for task state transitions
- **fortify**: Resilience (retry, timeout) for AI calls
- **go-plugin**: HashiCorp plugin system for external syncers

### Data Storage (.roady/)

All artifacts are git-friendly files:
- `spec.yaml` - Product specification (features, requirements)
- `spec.lock.json` - Pinned spec snapshot for drift detection
- `plan.json` - Task DAG with approval status
- `state.json` - Execution state (task statuses, paths)
- `policy.yaml` - Governance (max_wip, allow_ai, token_limit)
- `ai.yaml` - Provider/model defaults
- `events.jsonl` - Immutable audit trail (hash-chained)
- `usage.json` - AI token consumption telemetry

### Service Wiring

Services are composed via `internal/infrastructure/wiring`:
- `BuildAppServices(root)` returns all services with shared dependencies
- CLI and MCP share the same service instances
- `AuditService` is injected into all services for event logging

### AI Provider Architecture

```go
// pkg/domain/ai/provider.go - Interface
type Provider interface {
    Generate(ctx context.Context, prompt string) (string, error)
    Name() string
    Model() string
}

// pkg/ai/factory.go - Factory
NewProvider(providerName, modelName string) (ai.Provider, error)
// Supports: ollama, openai, anthropic, gemini, mock
```

Environment variables override file config:
- `ROADY_AI_PROVIDER`, `ROADY_AI_MODEL`
- `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`

### Task State Machine

Tasks follow strict FSM transitions via statekit:
```
pending → in_progress → done → verified
            ↓     ↑
         blocked
```

Guards enforce:
- WIP limits (policy.max_wip)
- Dependency completion before start
- Plan approval before execution

### MCP Tools

The MCP server (`internal/infrastructure/mcp/server.go`) exposes these tools:
- `roady_init`, `roady_get_spec`, `roady_get_plan`, `roady_get_state`
- `roady_generate_plan`, `roady_update_plan`, `roady_approve_plan`
- `roady_detect_drift`, `roady_accept_drift`, `roady_explain_drift`
- `roady_transition_task`, `roady_check_policy`, `roady_status`
- `roady_forecast`, `roady_git_sync`, `roady_sync`

Run MCP server:
```bash
roady mcp                          # stdio (default)
roady mcp --transport http --addr :8080
roady mcp --transport ws --addr :8080
```

### Plugin System

Plugins use HashiCorp go-plugin over RPC:
- Interface: `pkg/domain/plugin/Syncer`
- Loader: `pkg/plugin/loader.go`
- Examples: `cmd/roady-plugin-github`, `cmd/roady-plugin-jira`, `cmd/roady-plugin-linear`

## Common Workflows

```bash
# Initialize project
roady init my-project

# Analyze docs and generate spec
roady spec analyze docs/ --reconcile

# Generate plan (heuristic or AI)
roady plan generate
roady plan generate --ai

# Check drift and accept if intentional
roady drift detect
roady drift accept

# Task lifecycle
roady task start <task-id>
roady task complete <task-id>

# Git-based sync
git commit -m "Implement feature [roady:task-id]"
roady git sync
```

## Testing Patterns

- Unit tests alongside source files (`*_test.go`)
- Table-driven tests preferred
- Mock provider at `pkg/ai/mock.go` for AI tests
- Test helpers in `internal/infrastructure/cli/test_helpers_test.go`
