# Dashboard

> **Status:** stable since v0.11.0 (Kanban) / v0.12.0 (live actions + auth)

Roady ships a small web dashboard for any project. It surfaces the same
data the CLI does — plan, tasks, drift, stats — but in a form you can
keep open in a side tab while you and your agent work.

## Start it

```bash
roady dashboard serve --port 3000
```

Routes:

| Path | What you get |
|---|---|
| `/` | Stats grid + recent tasks |
| `/plan` | Approved plan view |
| `/tasks` | Flat task list with status pills |
| `/kanban` | Five-column Kanban board (per-project) |
| `/org/kanban` | Cross-project Kanban (root + every `.roady/projects/<name>/`) |
| `/events` | Server-Sent Events stream (`task-changed`) |
| `/api/plan`, `/api/state`, `/api/kanban`, `/api/org/kanban` | JSON for external tools |

## Kanban

Five status columns:

- **Backlog** — pending tasks whose dependencies are not yet satisfied
- **Ready** — pending tasks whose deps are all done/verified
- **In Progress** — currently being worked on
- **Blocked** — explicitly blocked with a reason
- **Done** — completed (verified rolls into done for column purposes)

### Click

Each card has contextual buttons:

| Column | Buttons |
|---|---|
| Ready | ▶ Start |
| In Progress | ✓ Complete · ⊘ Block |
| Blocked | ↺ Unblock |
| Done | ↺ Reopen |

### Drag-and-drop

Drag any card to a target column. Allowed transitions show a green
outline on drop hover; disallowed transitions show red (silently
ignored).

| From | To | Action |
|---|---|---|
| Ready | In Progress | start |
| In Progress | Done | complete |
| In Progress | Blocked | block |
| Blocked | In Progress | unblock |
| Done | Backlog / Ready | reopen |

### Live updates

The board subscribes to `/events` via `EventSource`. After every action
the server broadcasts `task-changed` and connected clients reload
within ~200 ms. A 25-second heartbeat keeps the stream alive through
proxies (Cloudflare, nginx). A 60 s meta-refresh is kept as fallback
for browsers without `EventSource`.

## Cross-project Kanban (`/org/kanban`)

When the workspace has nested sub-projects (see
[`rfcs/0001-nested-projects.md`](rfcs/0001-nested-projects.md)),
`/org/kanban` merges every project under the workspace root into one
board. Cards are tagged with their origin project label
(e.g. `repo/feature-auth`), and DnD drops route the mutation to the
right sub-project's `TaskService` via the same
`/actions/task/{start,complete,...}` endpoints with `project_path` +
`project` form fields.

The header strip lists every discovered project with task / done
counts.

## Action endpoints

All form-encoded, all redirect 303 to the referring page. Mounted only
when the server has task actions wired (the CLI does this
automatically via `services.Task`).

```
POST /actions/task/start     (form: id, optional owner, project_path, project)
POST /actions/task/complete  (form: id, optional evidence, project_path, project)
POST /actions/task/block     (form: id, optional reason,   project_path, project)
POST /actions/task/unblock   (form: id, optional project_path, project)
POST /actions/task/reopen    (form: id, optional project_path, project)
```

When `project_path` and/or `project` are present, the request routes
through `dashboard.OrgTaskActions.ResolveTaskActions` to the right
sub-project's `TaskService`. Otherwise the server-default
`taskActions` is used.

## Auth

The dashboard is public by default — fine for `localhost`, **not** fine
if you put it behind a tunnel or share the URL. Protect it with a
shared bearer token:

```bash
roady dashboard serve --port 3000 --auth-token "$(openssl rand -hex 16)"
```

Or via env:

```bash
ROADY_DASHBOARD_TOKEN="$(openssl rand -hex 16)" roady dashboard serve
```

Three ways to present the token:

1. `Authorization: Bearer <token>` header (programmatic clients)
2. `Cookie: roady_token=<token>` (browsers after one handshake)
3. `?token=<token>` query — one-time handshake: the server sets the
   cookie, redirects to strip the secret from the URL, then drops the
   query. Share the `?token=` link once; the recipient stays logged in
   for 30 days.

Comparison is constant-time. The cookie is `Secure` over TLS / behind
`X-Forwarded-Proto: https` (Cloudflare, nginx). Empty token = public.

## Putting it behind a Cloudflare tunnel

```bash
# Terminal 1
roady dashboard serve --port 3000 --auth-token "$(openssl rand -hex 16)"

# Terminal 2 (separate machine or same)
cloudflared tunnel --url http://localhost:3000
```

Cloudflare prints a `*.trycloudflare.com` URL. Append `?token=<value>`
the first time you open it; the cookie sticks.

For a persistent named tunnel + DNS, use
[Cloudflare Zero Trust](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/).

## Limitations

- HTML5 drag-and-drop is desktop-only. Mobile users have the buttons;
  touch DnD is on the roadmap.
- The board reloads on every change; large boards (>1k tasks) will
  feel that. SSE delivers a diff hint but not the diff itself.
- `/org/kanban` DnD requires the CLI to have wired
  `OrgTaskActions` (default behaviour of `roady dashboard serve`).
  Custom embedders need to call `Server.EnableOrgTaskActions`.

## JSON shape

`GET /api/kanban`:

```json
{
  "columns": [
    {"name": "Backlog", "status": "backlog", "tasks": [...], "count": 2},
    {"name": "Ready",   "status": "ready",   "tasks": [...], "count": 4},
    ...
  ],
  "project_name": "...",
  "total_tasks": 33,
  "updated_at": "2026-05-16T13:00:00Z"
}
```

`GET /api/org/kanban` adds:

```json
{
  "projects": [
    {"label": "repo", "path": "/abs/path", "sub_project": "", "total": 33, "done": 23},
    {"label": "repo/feature-auth", "path": "/abs/path/.roady/projects/feature-auth", "sub_project": "feature-auth", "total": 6, "done": 3}
  ],
  "total_tasks": 39,
  "total_done": 26
}
```

Cards include `Task`, `Status`, `Owner`, `ProjectLabel`, `ProjectPath`,
`ProjectName` (last three populated on org boards).
