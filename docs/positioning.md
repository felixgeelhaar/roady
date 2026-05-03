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

### Validated against

> **Status: partially validated via web research (2026-05-03);
> human ICP interviews still pending.**
>
> Treat the category claim as load-bearing only when at least one
> human-conducted interview row also lands.

#### Research method

Targeted web research across r/ClaudeAI / r/cursor adjacent blogs,
dev.to, Medium, GitHub READMEs (CLAUDE.md / AGENTS.md / spec-kit /
Backlog.md / mcp-memory-service / agent-memory-mcp), Stack Overflow,
IBM Think, Wire blog, Anthropic Help Center, and Cloudflare blog. Goal:
find the **verbatim phrasing** the ICP uses for the pain Roady solves
and how the existing tool ecosystem frames it.

#### Phrases the ICP actually uses for the pain

These are quoted or near-quoted across multiple sources. **Bold** = vocabulary
Roady's copy should adopt or claim.

- **"Plan drift"** / **"plan decay"** — Wire blog, IBM Think, Prassanna
  Ravishankar. Defined as: *"the agent's plan is still in context,
  still being followed, but no longer correct for the current state of
  the world. The goal is still right. The plan is wrong."* — direct
  match to Roady's drift detection.
- **"Agent drift"** — IBM Think, Stack Overflow blog. Umbrella term;
  six sub-types named (goal drift, context drift, role drift, tool-use
  drift, hallucination cascades, plan decay).
- **"Black box AI drift"** — Stack Overflow blog. *"AI tools are
  making design decisions nobody asked for."*
- **"Forgets everything between sessions"** — dev.to, Medium, Felo
  Search Blog. Single most common phrase.
- *"Old plan in new state"* — Wire blog. Almost a Roady tagline.
- *"Lose the plot"* — Wire blog headline.
- *"AI agent feels stupid sometimes"* — Codeaholicguy Feb 2026.
- *"10-15 minutes per session rebuilding context"* — dev.to (specific
  cost number worth quoting).

#### Phrases for the wished-for solution

- *"Persistent, structured, agent-agnostic context layer that lives in
  your project"* — IBM Think. Word-for-word match to Roady's premise.
- *"Treat the plan as mutable state that must be re-evaluated at
  checkpoints, not as an instruction list to execute"* — Wire blog.
  Direct match to spec-lock + drift detect loop.
- *"Separate the plan from the execution log and revise it on
  schedule"* — Wire blog. Direct match to spec.yaml +
  state.json + events.jsonl separation.
- *"Externalized state"* / *"context containers"* — Wire blog.
- *"Standing orders that survive sessions"* — MindStudio.
- *"Centralizes tasks so Claude can reference them without you
  re-explaining context each session"* — Backlog.md positioning.

#### Competitor presence (no longer hypothetical)

- **CLAUDE.md** — Anthropic native, hierarchical (repo / sub / user
  levels). Cap: 200 lines / 25KB. Read at session start.
- **AGENTS.md** — multi-agent standard backed by **Linux Foundation's
  Agentic AI Foundation** (mid-2025). Supported by Claude Code,
  Cursor, Copilot, Gemini CLI, Windsurf, Aider, Zed, Warp, RooCode.
  *"One file, any agent."* — direct competition for the "rules /
  instructions" surface.
- **spec-kit** (GitHub official) — `/specify`, `/plan`, `/tasks`
  slash commands. Spec-driven AI dev workflow. Recommended for small
  to medium projects.
- **Backlog.md** — markdown task board for git repos, AI-agent
  designed. Active community.
- **BMAD Method** — heavier, "comprehensive documentation and clear
  role separation" — for large projects.
- **mcp-memory-service / mcp-mem0 / agent-memory-mcp** — MCP-based
  *episodic / conversational* memory layers (Mem0, Cloudflare Durable
  Objects-backed). Different primitive: they remember *facts*, Roady
  remembers *plans*. Worth naming the difference explicitly.

#### Synthetic ICP interview (grounded in the research above)

> **Note:** This is a constructed persona, not a real interview.
> Treat it as a hypothesis-stress-test until replaced with at least
> one transcript from a real conversation.

**Persona — "Sam", staff engineer**

- 8 years experience, 3rd company. Currently at a 25-person SaaS,
  works mostly solo on backend features.
- Pairs with Claude Code daily; uses Cursor when in the IDE.
- Has a `CLAUDE.md` and a `Backlog.md`. Tried spec-kit "but it felt
  like ceremony".
- Pays for Claude Max + Cursor Pro out of pocket.

> *"What goes wrong with your AI workflow?"*
>
> Sam: *"Most of it works. The problem is when I come back on a Monday
> and Claude has either forgotten what we agreed Friday or, worse, is
> confidently doing the thing we explicitly said not to do. I spend
> the first ten minutes of every session rebuilding context. My
> CLAUDE.md helps but it's just standing orders — it doesn't tell
> Claude where we are in the plan."*

> *"What about Backlog.md?"*
>
> Sam: *"Yeah, I have one. The agent reads it. But there's no
> connection between the spec doc I wrote and the tasks it generates.
> If I change the spec, the tasks don't know. If the agent ships
> something the spec doesn't ask for, nothing flags it. That's the
> bit I keep missing."*

> *"If I described 'planning memory for AI coding agents', would that
> phrase land for you?"*
>
> Sam: *"Memory of what, though? mcp-memory-service is also called
> memory and it remembers chat. I'd think that's another mem0 clone.
> What you're describing is more like... the plan layer? Or
> drift-aware planning? The drift bit is the part that doesn't exist
> elsewhere — name that."*

> *"How would you describe what you wish you had, in your own words?"*
>
> Sam: *"Something that sits between my spec and the agent and
> notices when reality stops matching what I said I was building.
> Like a continuous diff of intent vs code. Spec-kit wants me to
> ceremoniously file specs. Backlog.md is just a list. I want the
> alarm bell when the agent starts inventing."*

> *"How would you find a tool like this?"*
>
> Sam: *"r/ClaudeAI or the MCP Discord, probably. Or someone tweeting
> 'I built a thing that catches Claude going rogue mid-feature'."*

#### What the research changes about positioning

1. **The word "memory" is contested.** mcp-memory-service, mcp-mem0,
   agent-memory-mcp all use it for episodic / conversational recall.
   Sam's reaction is the predictable one. "Planning memory" needs
   work — alternatives to test:
   - "**Plan layer for AI coding agents**" (Sam's own phrase)
   - "**Drift-aware planning for AI coding agents**" (claims the
     verbatim differentiator)
   - "**Continuous spec-vs-code diff for AI coding agents**" (most
     concrete; longest)
2. **"Plan drift" is unowned vocabulary.** Wire and IBM defined it but
   no product owns it. Roady should claim "plan drift detection" the
   way Sentry claimed "error monitoring" — single phrase, single
   product.
3. **The competitive frame is real and active.** AGENTS.md is now
   Linux Foundation-backed; spec-kit is from GitHub itself. Roady is
   not entering an empty room. Differentiation hangs entirely on the
   drift-detection + spec-lock loop, not on "we're a planning tool".
4. **Distribution channels confirmed.** r/ClaudeAI, MCP Discord,
   dev.to, Twitter/X handles posting "I built X". No surprise; do not
   waste time on LinkedIn / paid SEO until ECP is locked.

#### Validation table

| Audience | Date | Phrase tested | Result |
| --- | --- | --- | --- |
| Web research (synthesis above) | 2026-05-03 | "planning memory" | "memory" is contested with episodic-memory MCP servers; "plan drift" / "drift-aware planning" is open vocabulary that maps directly to differentiator |
| Synthetic ICP interview ("Sam") | 2026-05-03 | "planning memory for AI coding agents" | Misread as another Mem0 clone; reframed by ICP as "the plan layer" / "drift-aware planning" |
| _r/ClaudeAI_ | _pending_ | "drift-aware planning for AI coding agents" + "plan drift detection" | _pending_ |
| _MCP Discord_ | _pending_ | (same) | _pending_ |
| _3-5 real ICP interviews_ | _pending_ | (same) | _pending_ |

**Recommended next step:** test "drift-aware planning for AI coding
agents" + the noun phrase "plan drift detection" against r/ClaudeAI
and MCP Discord before the next round of public copy. The Wire blog
and IBM Think articles already give us the citation chain to ground
the term in.

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
