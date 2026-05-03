# Positioning

> One page. One ICP. One category. Source of truth for all README, website,
> and pitch copy. Updated when we change our mind, not when we ship features.

## In one sentence

**Roady is the planning memory layer for AI coding agents.**

Specs, plans, and execution state persist as durable, git-versioned files
that survive context resets and travel between sessions, agents, and
humans.

## ICP — who we are for

Engineers who pair with AI coding agents (Claude Code, Codex, Cursor,
Gemini, OpenCode) on real, multi-day work and lose context between
sessions or when the agent rewrites the wrong thing because it forgot
what was decided yesterday.

Concretely:

- **Primary**: founders and senior individual contributors shipping
  features end-to-end through an AI agent on one or two repos.
- **Adjacent (not primary)**: 3-10 person engineering teams adopting
  AI agents who need shared planning state across collaborators.
- **Out of scope for v0.x**: large engineering orgs, classical PM
  tooling buyers, hand-managed Jira shops without agent workflows.

## Category

**Planning memory for AI coding agents.**

A memory layer is not a planning tool, not an issue tracker, not a wiki.
It is the durable state record that intent, plans, and execution operate
against. We are claiming this category before agent runtimes (Claude
Code, Cursor, Codex) bake their own opinionated, lock-in versions of it.

## The alternatives — what users compare us to

| If they were not using Roady, they would be using... | And the gap is... |
| --- | --- |
| Cursor rules / Claude.md / AGENTS.md | Static instructions, no plan state, no drift detection |
| spec-kit / Backlog.md | Spec-only or task-only; no execution loop |
| Linear / Jira / GitHub Projects | Built for human PMs; agents cannot read or update them durably |
| Aider repo-map / Sweep | Read-time context; nothing persisted as intent |
| Nothing — chat history + memory | Lossy across resets; no shared truth |

## Unique attributes — what only Roady does

1. **Spec-lock + drift detection.** Snapshots intent (`spec.lock.json`)
   and continuously diffs current spec, plan, and code reality. Surfaces
   exactly where intent and execution have parted company.
2. **Hash-chained event log.** Every state change is an immutable,
   replayable domain event. Audit trail, projections, and rollback for
   free.
3. **MCP-first surface.** All planning operations exposed as MCP tools so
   any agent (Claude Code, Codex, Gemini, OpenCode) can read and write
   the same state without bespoke integration.
4. **Provenance to the source.** Every task carries an `Origin`
   (heuristic / ai / human) and an optional `from doc:line` citation
   back to the spec document that motivated it.
5. **Local-first, file-based, git-friendly.** `.roady/` lives in the
   repo. No SaaS lock-in, no separate auth, no vendor channel.

## Value those attributes deliver

For the ICP, the working day improves like this:

- Resume work after a 3-day break and the agent picks up exactly where
  the plan said to next.
- Catch the moment your AI added a feature you never asked for, before
  it ships.
- Hand a session over to a colleague (or another agent) by pointing
  them at the repo. No briefing required.
- Stay inside your editor / agent. No tab-switching to a planning
  product.

## What we are explicitly NOT

- **Not a Jira / Linear replacement.** No sprints, no swimlane UI, no
  triage rituals. We track intent and reality, not human ceremony.
- **Not a chat memory layer.** We persist plans, not conversations.
- **Not a hosted SaaS** (today). The CLI and MCP server are the
  product. Roady Cloud is on the roadmap, see `ROADMAP.md`.
- **Not a code-search or context-stuffing tool.** We complement those;
  we do not replace them.

## Tone of voice

Direct, technical, present tense. Lead with the workflow, not the
feature list. Avoid PM jargon ("velocity", "story points",
"stakeholders"). Avoid AI-pitch jargon ("synergize", "agentic
revolution"). Default audience knows what an MCP tool is and has run
`brew install` in the last week.

## Anti-positioning — what to drop on sight

If a draft includes any of these, rewrite:

- "Plan, build, and ship faster." (says nothing)
- "AI-powered project management." (wrong category)
- "For teams of all sizes." (no, just our ICP)
- Long bulleted feature lists in the hero. (move to docs/advanced.md)
