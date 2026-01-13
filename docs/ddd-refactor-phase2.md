# DDD Refactor Phase 2: Hardening & Cleanup

Harden the refactor by covering configuration paths, updating remaining documentation, and verifying external compatibility.

## Audit Docs & Examples
Update any remaining references to `policy.yaml` for provider/model settings.
- Find and fix docs or examples that still describe provider/model in `policy.yaml`.
- Ensure `.roady/ai.yaml` is documented as the provider/model source.
- Keep guidance for env var overrides intact.

## AI Config & Wiring Tests
Add coverage for AI config and wiring fallbacks.
- LoadAIConfig handles missing or invalid files.
- LoadAIProvider uses defaults when `ai.yaml` is missing.
- CLI AI configure writes `ai.yaml` alongside `policy.yaml`.

## MCP Shim & Compatibility
Validate and document the `pkg/mcp` shim.
- Confirm downstream imports still work with the shim.
- Decide whether to keep or deprecate the shim.
- Add a short note in docs if the shim is temporary.
