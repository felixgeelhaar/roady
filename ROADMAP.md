# Roady — Roadmap

> Public, opinionated, subject to change. Issues with the `roadmap`
> label are the live tracking surface.

The CLI + MCP server is and stays free, MIT, and self-hostable. The
roadmap below describes what's coming next on the open core, and the
intended **open-core boundary** for a future hosted product.

---

## Now (v0.12.x — shipped)

- **Live Kanban dashboard** at `/kanban` with five status columns
  (Backlog / Ready / In Progress / Blocked / Done) and a `/api/kanban`
  JSON endpoint. Click or drag to drive task transitions; board
  reloads ~200 ms after every change via Server-Sent Events.
- **Cross-project Kanban** at `/org/kanban` merges every project under
  the repo (root + sub-projects) into one board; cards carry their
  origin project and drops route to the right sub-project's
  `TaskService`.
- **Action endpoints**: `POST /actions/task/{start,complete,block,unblock,reopen}`,
  form-encoded, accept optional `project_path` + `project` for org
  routing.
- **Dashboard auth token** (`--auth-token` flag or `ROADY_DASHBOARD_TOKEN`
  env). Bearer / Cookie / one-time `?token=` handshake. Empty = public.

## Recently (v0.11.x — shipped)

- **Nested sub-projects** under `.roady/projects/<name>/`. One repo
  hosts many projects in parallel; coding agents switch context with
  `--project / -P <name>` (CLI) or `project` (MCP). Existing flat
  `.roady/` repos unchanged. See
  [`docs/rfcs/0001-nested-projects.md`](docs/rfcs/0001-nested-projects.md).
- `roady discover` and `roady org status` surface sub-projects.

## Earlier (v0.10.x — shipped)

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

## Next (v0.13.x)

- **Cross-project task dependencies** — `@project:task-id` syntax in
  `DependsOn` so an org-level plan can express "feature-payments task
  X waits on feature-auth task Y".
- **Inline-edit on Kanban actions** — owner / evidence / reason
  prompts on the dashboard instead of the current defaults
  (`owner=dashboard`, `evidence="completed via dashboard"`).
- **Drift explainer follow-ups**: synthesised "explain + propose patch"
  output that lands a PR-ready diff for accepted drift.
- **Per-task subagent dispatch**: `roady task start` can hand a ready
  task to a subagent (Claude Code `Task` tool, Codex `agent run`, etc.)
  with the spec source attached and a deterministic completion hook.
- **Spec-to-PR loop**: CI integration that auto-detects drift on PR
  merge and either accepts or opens a follow-up issue.
- **Touch-device Kanban DnD** — HTML5 DnD is desktop-only today;
  mobile users still have the buttons.

## Soon (v0.14+)

- **Cross-repo planning**: a single `.roady/` workspace can declare
  member repos, share spec context, and aggregate plan state.

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
