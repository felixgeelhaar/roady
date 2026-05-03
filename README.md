<p align="center">
  <img src="logo.svg" width="150" alt="Roady Logo">
</p>

[![Go Version](https://img.shields.io/github/go-mod/go-version/felixgeelhaar/roady?logo=go)](https://github.com/felixgeelhaar/roady)
[![Coverage](https://img.shields.io/badge/coverage-82%25-brightgreen?logo=coveralls)](https://github.com/felixgeelhaar/roady/actions)
[![Release](https://img.shields.io/github/v/release/felixgeelhaar/roady?include_prereleases&logo=github)](https://github.com/felixgeelhaar/roady/releases/latest)
[![nox Security](https://img.shields.io/badge/nox-A-brightgreen?logo=lock)](https://github.com/felixgeelhaar/roady/security)
[![nox Scan](https://img.shields.io/badge/scan-0%20findings-brightgreen)](https://github.com/felixgeelhaar/roady/security)

# Roady — planning memory for AI coding agents

> Specs, plans, and execution state that **survive context resets** and
> travel between sessions, agents, and humans. File-based, git-versioned,
> MCP-native.

You pair with Claude Code, Codex, Cursor, or Gemini on a multi-day
feature. Three days in, the agent forgets what was decided, rewrites the
wrong thing, or quietly drifts off-spec. Roady is the durable layer
that holds the answer to *what are we building, what's next, and where
did reality diverge from the plan?* — readable by you and writable by
your agent.

## See it in 60 seconds

```bash
brew install felixgeelhaar/tap/roady     # or: go install github.com/felixgeelhaar/roady/cmd/roady@latest
roady demo                               # scaffolds a sample project + shows drift
```

The demo creates a `roady-demo/` directory with a deliberately drifted
spec/plan, runs `roady drift detect`, and prints the next steps. Zero
prerequisites, zero AI keys, zero signup.

## The actual workflow

```bash
# 1. Hook your agent to Roady (one command per supported tool)
roady setup claude-code           # or claude-desktop, opencode, openai, gemini

# 2. Initialise + import your existing docs
roady init my-project
roady spec analyze docs/          # parses markdown, captures source citations

# 3. Generate a plan (heuristic by default; --ai for richer decomposition)
roady plan generate
roady plan approve

# 4. Drive execution from inside your AI editor
/roady-task                       # agent picks the next ready task
# ...agent implements, commits with [roady:task-id] marker...
roady git sync                    # state moves forward automatically

# 5. Ask the question that matters
roady drift detect                # has reality diverged from intent?
```

Status, drift, and progress all show in `roady status` — including a
`from doc:line` citation for every task so the AI's choices stay
auditable.

## What Roady is, and is not

| Roady is... | Roady is not... |
| --- | --- |
| The plan-of-record for an AI-paired feature | A Jira / Linear replacement |
| Memory that survives `/clear` and session resets | A chat history layer |
| File-based, git-friendly, local-first | A hosted SaaS (today) |
| MCP-native — every operation is a tool | A code-search or context-stuffing tool |

See [`docs/positioning.md`](docs/positioning.md) for the full positioning,
ICP, and category claim.

## How it compares

[`docs/vs.md`](docs/vs.md) — opinionated comparison vs Cursor rules,
Claude.md, spec-kit, Backlog.md, Linear, GitHub Projects.

## Everything else

The headline workflow is intentionally short. Roady supports billing
rates, debt scoring, dependency graphs, multi-project org dashboards,
plugin syncers, fsnotify watch mode, web dashboards, D3 visualisations,
realtime SSE streaming, webhook + Slack notifications, and more — see
[`docs/advanced.md`](docs/advanced.md) for the full catalogue grouped by
audience (solo dev / small team / org).

## Roadmap

[`ROADMAP.md`](ROADMAP.md) sketches what's next, including the planned
**Roady Cloud** open-core boundary (hosted MCP, multi-repo org
dashboard, audit retention, SOC2).

## Contributing & license

Contributions welcome — open an issue or PR. MIT License, see `LICENSE`.

---

*Built with `cobra`, `bubbletea`, `mcp-go`, `fortify`. Domain-driven Go
with `pkg/domain` / `pkg/application` / `internal/infrastructure`.
Architecture notes in [`docs/architecture.md`](docs/architecture.md) (or
the existing DDD docs in `docs/ddd-*.md`).*
