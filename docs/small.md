# Roady – Product Vision

Roady is a **planning-first system of record for software work**.

It exists to be the **durable memory layer** between intent and execution, helping individuals and teams maintain continuity as projects evolve.

## Ownership Boundaries

| Roady Owns | Roady Does NOT Own |
|------------|-------------------|
| **Spec** (The "What" and "Why") | **Execution** (The act of doing) |
| **Plan** (The "How" and "When") | **Code** (The implementation) |
| **Drift Awareness** (The Reality Check) | **Tickets** (The task tracking) |
| | **Deployment** (The shipping) |

---

## Core Philosophy

| Principle | Meaning |
|-----------|---------|
| **Determinism** | Same input → same plan |
| **Explicitness** | Nothing inferred without record |
| **Drift-first** | Drift is a signal, not an error |
| **Local-first** | Works offline, AI optional |
| **Policy-bound** | Never acts beyond constraints |
| **Extensible** | Integrates, never replaces |

---

## Horizons

### Core (Open Source)
- **Repo-scoped:** Planning happens where the code lives.
- **Single logical plan:** One source of truth for the workspace.
- **Async collaboration:** Shared via Git, just like code.
- **CLI + MCP:** Interfaces for both humans and AI agents.
- **Plugin system:** Extendable through gRPC/RPC integrations.

### Pro (Commercial)
- **Multi-user visibility:** Shared dashboards and progress views.
- **Team coordination:** Orchestrating across multiple contributors.
- **Governance & compliance:** Automated audit ledgers and policy guardrails.
- **Cross-repo intelligence:** Planning that spans multiple repositories and services.

**Core is complete by itself.** Pro builds **on top**, never underneath.

---

## North Star Experience

At any time, a user (or AI agent) can ask:

> “Where are we, why, and what should happen next?”

Roady answers clearly and confidently, without re-deriving history.