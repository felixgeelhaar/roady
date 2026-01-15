# AI Configuration Guide

This project records **governance control** and **provider preferences** in two distinct places so CLI, MCP, and automation always use the same wiring helper without polluting policy.

## Files

- `.roady/policy.yaml` – keep only governance flags such as:
  ```yaml
  max_wip: 3
  allow_ai: true
  token_limit: 50000
  ```
  Policy gates AI usage, budgets, and WIP limits. It does **not** control which provider is used.

- `.roady/ai.yaml` – configure provider/model defaults consumed by `internal/infrastructure/wiring.LoadAIProvider`:
  ```yaml
  provider: ollama
  model: llama3
  ```
  The wiring helper combines this with `AuditService` and services so CLI and MCP share the same provider selection logic.

## Commands & Workflow

- Run `roady ai configure --allow-ai=true --token-limit=12345 --provider=openai --model=gpt-4o` to edit both files at once. The command stores governance data in `.roady/policy.yaml` and provider/model choices in `.roady/ai.yaml`.
- Use `roady init --interactive` to prompt for AI defaults after creating a workspace; it writes the same pair of files.
- Override defaults temporarily with environment variables:
  ```
  export ROADY_AI_PROVIDER=gemini
  export ROADY_AI_MODEL=gemma3
  ```
  These overrides are evaluated before `.roady/ai.yaml` so agents can point at different providers without editing the repo.

## AI Onboarding Checklist

1. Run `roady init --workspace=<path>` (or `roady init --interactive` inside the repo) so the workspace creates `.roady`, the audit trail, and the default policy file.
2. Configure policy and provider defaults with `roady ai configure --allow-ai --token-limit=5000 --provider=ollama --model=llama3.2`, or use `--interactive` to prompt for each value. This command writes governance flags to `.roady/policy.yaml` and model/provider choices to `.roady/ai.yaml`.
3. Confirm `.roady/policy.yaml` contains the desired `allow_ai`, `token_limit`, and `max_wip` settings, and that `.roady/ai.yaml` names the intended provider/model pair. The wiring helper at `internal/infrastructure/wiring` reads both files so CLI and MCP share the same provider selection logic.
4. For temporary overrides (experiments, CI, or alternate models) set `ROADY_AI_PROVIDER`/`ROADY_AI_MODEL` before running `roady plan generate --ai`. Remember to document the override in your PR or governance notes if it affects plan generation.

## Governance Logging

- Every important transition uses the shared wiring helper so the audit trail stays consistent. For example, `DriftService.AcceptDrift` logs a `drift.accepted` event via `AuditService.Log`.
- The `roady drift accept` command and MCP’s `roady_accept_drift` tool both run through the same services, ensuring csv entries mention the same `spec_id`, `spec_hash`, and actor (`cli` or `ai`).

## Validation & Troubleshooting

- If you change providers, rebuild the workspace (`wiring.BuildAppServices`) to refresh the bound services. The CLI/MCP wiring already prints warnings when it falls back to an embedded provider.
- Keep API keys out of the repo; store them in host environment variables (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, etc.).

## Interpreting `.roady/events.jsonl`

`AuditService.Log` appends a newline-delimited JSON event to `.roady/events.jsonl` whenever a deterministic transition occurs. Each entry looks like:

```json
{
  "action": "plan.generate",
  "actor": "cli",
  "metadata": {
    "plan_id": "plan-gov-170",
    "spec_id": "gov"
  },
  "prev_hash": "...",
  "hash": "...",
  "timestamp": "2025-01-01T..."
}
```

- **Plan transitions** (`plan.generate`, `plan.update_smart`, `plan.approve`, `plan.reject`, `plan.prune`) include `plan_id`/`spec_id` so you can trace plan edits and approvals. The wiring helper shares the same services for CLI and MCP, so both transports write the same actions.
- **AI retries** emit `plan.ai_decomposition_retry` with retry reason/attempt, making it clear when the model initially returned invalid JSON before succeeding.
- **Drift acceptance** is captured as `drift.accepted`; the metadata includes `spec_id` and `spec_hash`, proving the governance decision to sync the plan.

Use `jq -c '.action' .roady/events.jsonl` to list actions and `jq 'select(.action=="plan.approve")'` to inspect specific transitions when you need to audit agents or CLI commands.

## Governance Checklist

- After you edit `docs/`, `docs/PLUGIN_ROADMAP.md`, or `.roady/spec.yaml`, regenerate the plan (`roady plan generate --ai` or `roady plan generate`) so `plan.generate` records the new snapshot.
- Approve the refreshed plan (`roady plan approve`) whenever the spec/intention has stabilized; this writes `plan.approve` and keeps the approval metadata aligned with the current spec hash.
- When drift is expected because you intentionally changed the spec, run `roady drift accept` and verify that `drift.accepted` appears in `.roady/events.jsonl`, capturing the `spec_id`, `spec_hash`, and the actor that accepted the drift.
- Use `jq -c 'select(.action | startswith("plan."))' .roady/events.jsonl` to trace the lifecycle of each plan (generate, approve, reject, prune). If you need a printable checklist, see `docs/governance-checklist.md` for the expanded audit workflow.
