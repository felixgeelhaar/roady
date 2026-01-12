# Roady Project Context

## Project Overview

**Roady** is a planning-first system of record for software work. It acts as a durable memory layer between intent (what we want to build), plans (how we will build it), and execution (the actual work). It is designed to help individuals, teams, and AI agents answer "What are we building?", "Why?", and "What's next?" without re-deriving context.

*   **Type:** CLI Tool & MCP Server (Local-first, Open Source Core)
*   **Goal:** To be the deterministic source of truth for software planning, preventing drift between specs and reality.
*   **Key Principles:** Planning before execution, explicit intent, drift detection, and local-first data ownership.

## Documentation Structure

The project is currently in the design and planning phase. The `docs/` directory contains the foundational documents:

*   **`docs/vision.md`**: The high-level philosophy and "North Star" of the project. Defines what Roady is (planning engine, source of truth) and what it is not (task runner, chat transcript).
*   **`docs/prd.md` (Product Requirements Document)**: Detailed capabilities for Roady Core.
    *   **Core Capabilities:** Spec Management, Plan Generation, Drift Detection, Status & Querying.
    *   **Interfaces:** CLI (`roady`) and MCP (Model Context Protocol).
    *   **Scope:** Focuses on individual and repo-scoped planning (Core), excluding org-wide governance (Pro).
*   **`docs/roadmap.md`**: The problem-driven roadmap, split into Horizons (Core Foundation, Core Maturity, Pro Exploration, Org Intelligence).
*   **`docs/tdd.md` (Technical Design Document)**: Architectural overview.
    *   **Architecture:** DDD-oriented, event-driven, bounded contexts (Spec, Planning, Drift, Policy, Plugin).
    *   **Tech Stack Components:** Mentions usage of `statekit` (state management) and `fortify` (resilience).
    *   **Storage:** Local `.roady/` directory, git-friendly formats (YAML/JSON).

## Architecture & Design

Roady is designed as a modular system with clear boundaries:

*   **Spec Context:** Handles product specifications and requirements.
*   **Planning Context:** Manages plans as Directed Acyclic Graphs (DAGs) of tasks.
*   **Drift Context:** Detects discrepancies between specs, plans, and actual code/state.
*   **Interfaces:**
    *   **CLI:** `roady init`, `roady spec`, `roady plan`, `roady drift`, `roady status`.
    *   **MCP:** Exposes these capabilities to AI agents.

## Project Status

**Current Phase:** Design & Documentation.
**Codebase:** No source code implementation is currently present in the root directory. The project is defined by its specifications in `docs/`.

## Intended Usage

Once implemented, Roady will be used via the command line to manage the software development lifecycle:

```bash
# Example Workflow
roady init              # Initialize a new roady project
roady spec generate     # Create a spec from intent/docs
roady plan generate     # Create a plan from the spec
roady drift detect      # Check if reality matches the plan
```
