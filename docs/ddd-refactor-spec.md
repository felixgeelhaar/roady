# DDD Refactor Plan

Refactor Roady's tool wiring and configuration to align with DDD boundaries while preserving behavior.

## Move MCP Composition Root
Consolidate MCP server wiring at the infrastructure edge.
- Move MCP server construction out of pkg/mcp into internal/infrastructure/mcp.
- Keep pkg/mcp minimal (interfaces/types only) or remove if unused.
- Ensure command wiring uses the new infrastructure entry point.

## Decouple Domain Policy from Provider Selection
Keep domain policy focused on business rules, not adapter choices.
- Remove AI provider/model fields from domain policy config.
- Introduce infra/app config for provider/model defaults.
- Update provider selection to read infra config only.

## Centralize Adapter Construction
Create a single wiring location for repo, audit, AI provider, and services.
- Add internal/infrastructure/wiring (or cmd/roady) constructor.
- Make CLI and MCP use the shared wiring helper.
- Keep application services unchanged.

## Update Imports and Tests
Align references and maintain test coverage.
- Fix imports that reference pkg/mcp after relocation.
- Add temporary shim if external consumers still import pkg/mcp.
- Update tests to use new wiring location.
