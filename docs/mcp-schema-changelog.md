# MCP Schema Changelog

## Schema Evolution Rules

- **Patch** (1.0.x): Description fixes only
- **Minor** (1.x.0): New optional fields (`omitempty`), new tools, fields deprecated
- **Major** (x.0.0): Required fields added/removed, tool signatures changed

## v1.0.0 — Baseline

Initial schema version. All 37 existing MCP tools and their argument structs are frozen as the v1 contract:

`roady_init`, `roady_get_spec`, `roady_get_plan`, `roady_get_state`,
`roady_generate_plan`, `roady_update_plan`, `roady_detect_drift`,
`roady_accept_drift`, `roady_status`, `roady_check_policy`,
`roady_transition_task`, `roady_explain_spec`, `roady_approve_plan`,
`roady_get_usage`, `roady_explain_drift`, `roady_add_feature`,
`roady_forecast`, `roady_org_status`, `roady_git_sync`, `roady_sync`,
`roady_deps_list`, `roady_deps_scan`, `roady_deps_graph`,
`roady_debt_report`, `roady_debt_summary`, `roady_sticky_drift`,
`roady_debt_trend`, `roady_org_policy`, `roady_org_detect_drift`,
`roady_plugin_list`, `roady_plugin_validate`, `roady_plugin_status`,
`roady_messaging_list`, `roady_get_snapshot`, `roady_get_ready_tasks`,
`roady_get_blocked_tasks`, `roady_get_in_progress_tasks`

No deprecated fields.
