# Roady vs the alternatives

Opinionated, current as of 2026-05. PRs welcome to keep it honest.

## At a glance

| | Roady | Cursor rules / Claude.md / AGENTS.md | spec-kit | Backlog.md | Linear / Jira / GitHub Projects | Aider repo-map |
| --- | --- | --- | --- | --- | --- | --- |
| Survives `/clear` and session resets | ✅ | ✅ (static) | ✅ | ✅ | ✅ | ❌ |
| Plan that an agent can read **and write** | ✅ | ❌ | ❌ | partial | ❌ | ❌ |
| Drift detection (intent ↔ plan ↔ code) | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Source citation per task (`from doc:line`) | ✅ | ❌ | partial | ❌ | ❌ | ❌ |
| MCP-native, every operation a tool | ✅ | ❌ | ❌ | ❌ | (third-party) | ❌ |
| File-based, git-versioned, local-first | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| Free / open source | ✅ | varies | ✅ | ✅ | freemium | ✅ |
| Designed for AI-paired workflow | ✅ | ✅ | partial | ❌ | ❌ | ✅ |
| Hash-chained audit log | ✅ | ❌ | ❌ | ❌ | (server-side) | ❌ |

## When you should use Roady

- You pair with an AI coding agent on multi-day work and lose state
  between sessions.
- You want the plan-of-record in the repo, not in a SaaS your agent
  cannot reach.
- You want drift detection — to know *exactly* when reality stopped
  matching what you said you'd build.

## When you should NOT use Roady

- You are running a 50-person engineering org with full PM ceremony
  around Linear / Jira. Roady is not a replacement; it is a
  complement at most.
- You don't use AI coding agents at all. Roady's value is much
  smaller; pick a lighter spec-kit-style tool.
- You want a hosted SaaS today. Roady is a local CLI + MCP server.
  Roady Cloud (open-core) is on the [roadmap](../ROADMAP.md), not
  available yet.

## Detailed comparisons

### vs Cursor rules / Claude.md / AGENTS.md

Cursor rules and `Claude.md` are *static instructions* the agent reads
each turn. They don't track what was decided yesterday, what's done,
what's blocked, or whether the code matches the spec. They are
constants; Roady is state.

Combine them. Use Roady for plan + drift; use Claude.md for stable
project conventions.

### vs spec-kit (GitHub)

spec-kit is for *writing* specs. Roady is for keeping spec, plan, and
execution in sync over time. Roady's `roady spec analyze` reads
spec-kit-style markdown without modification.

Combine them. Author specs however you like; let Roady track the
execution loop.

### vs Backlog.md

Backlog.md ships a markdown task file. Lightweight, no plan/spec
relationship, no drift detection, no MCP. If your workflow is "list of
tasks in a file", Backlog.md is fine. If you need provenance from
spec to task, drift between intent and code, or AI agent integration,
you need Roady.

### vs Linear / Jira / GitHub Projects

These are built for human PMs and human eng teams. They have
sprints, swimlanes, custom fields, and assignees. They do not have
drift detection, source citations from spec docs, or MCP. Their APIs
are not good interfaces for AI agents (rate limits, complex auth,
high-cost reads).

Roady ships a `roady-plugin-linear` / `roady-plugin-jira` /
`roady-plugin-github` syncer if you want both: human PMs in Linear,
agents working through Roady, state synchronised both directions.

### vs Aider repo-map / Sweep

Aider's repo-map and Sweep's context tooling fetch *read-time*
context for the model. Roady tracks *write-time* state — what was
decided, what's done. They solve different problems and compose well.

### vs "nothing — chat history + memory"

The default. Lossy across `/clear`. No shared truth between sessions.
No way for a colleague (or another agent) to pick up your work
without a verbal handover. Works until it doesn't, usually around
day 3 of a feature.

## Honest limitations

- **Single-repo plans dominate.** `roady org` aggregates across
  repos but each repo still owns its `.roady/`. Cross-repo planning
  is on the roadmap.
- **No real-time multi-user UI.** State syncs via git push/pull plus
  optimistic locking. Two collaborators editing in parallel will
  conflict on `.roady/state.json` like any other file.
- **Heuristic planner is intentionally simple** (1 requirement = 1
  task). Real complexity needs `--ai`. The eval harness in
  `evals/` keeps both planners honest.
- **Provider streaming exists but Confidence/Sources from real
  providers is partial.** Confidence comes from `stop_reason`
  heuristics; native API source citations only land for providers
  that expose them (Gemini grounding metadata being the closest).
