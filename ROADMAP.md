# Roady — Roadmap

> Public, opinionated, subject to change. Issues with the `roadmap`
> label are the live tracking surface.

The CLI + MCP server is and stays free, MIT, and self-hostable. The
roadmap below describes what's coming next on the open core, and the
intended **open-core boundary** for a future hosted product.

---

## Now (v0.10.x — shipped)

- Eval harness over heuristic + AI planners + drift corpus
- Task provenance: `Origin` (heuristic / ai / human) + source citations
  back to the doc that motivated each task
- Provider streaming end-to-end (Anthropic / OpenAI / Gemini / Ollama)
  with MCP progress notifications and CLI `withAIProgress` integration
- MCP tool consolidation (`roady_tasks` parameterised + deprecation
  aliases for the legacy task-listing tools)
- `roady_cost_estimate` pre-flight token + USD projection per AI op
- Unified `roady notify` namespace; `messaging` and `webhook notif`
  retained as deprecation aliases
- AI command progress + clean SIGINT cancellation across the five AI
  CLI surfaces
- `roady demo` for <1s aha; `roady init --interactive` default in TTY;
  empty-state ladder on `roady status`

## Next (v0.11.x)

- **Positioning + narrative cleanup** (this release): one-page
  positioning doc, README hero rewrite, advanced features moved to
  `docs/advanced.md`, competitive comparison in `docs/vs.md`, this
  roadmap.
- **Website refresh** to mirror the README positioning.
- Real provider matrix run via `-tags evals_ai` documented in
  `evals/README.md` so contributors can self-serve a confidence check.

## Soon (v0.12 — v0.13)

- **Drift explainer follow-ups**: synthesised "explain + propose patch"
  output that lands a PR-ready diff for accepted drift.
- **Cross-repo planning**: a single `.roady/` workspace can declare
  member repos, share spec context, and aggregate plan state.
- **Per-task subagent dispatch**: `roady task start` can hand a ready
  task to a subagent (Claude Code `Task` tool, Codex `agent run`, etc.)
  with the spec source attached and a deterministic completion hook.
- **Spec-to-PR loop**: CI integration that auto-detects drift on PR
  merge and either accepts or opens a follow-up issue.

## Later

- **Plugin marketplace** for syncers and notifiers, opinionated quality
  bar (signed binaries, contract tests must pass).
- **Native source citations** through providers that surface them
  (Gemini grounding metadata, Anthropic citation API once stable).
- **Drift detection over code semantics**, not just structure — diff
  the implemented behaviour against the spec's natural-language
  requirement using a constrained AI checker.

---

## Roady Cloud (future, no committed date)

Open-core boundary, intended scope:

- **Hosted MCP** — managed multi-tenant MCP server so teams without
  a per-developer install can plug their agents into a shared
  workspace.
- **Multi-repo org dashboard** with persistent storage and historical
  metrics across all member projects.
- **Audit log retention** beyond what fits comfortably in `events.jsonl`,
  with structured search and export.
- **SSO / RBAC / SCIM** for enterprise IdP integrations.
- **SOC 2** compliance posture for the hosted plane.

What stays open and free, forever:

- The full CLI, MCP server, and every planning / drift / spec /
  notify / billing capability.
- The `.roady/` file format and all storage adapters.
- Plugin contract tests + reference syncer plugins.

If Cloud lands, opting in is a `roady cloud login` away. Opting out is
the existing local workflow with no behavioural change.

---

## Out of scope

- Becoming a Linear / Jira replacement. We integrate with them; we do
  not compete with them.
- A web-based code editor or AI agent of our own.
- Hosted general-purpose memory for non-coding workflows.

If you want any of these, Roady is the wrong tool — we are deliberately
narrow.
