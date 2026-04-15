# Admin Audit Log Page Display Enhancement Design

Date: 2026-04-15

## Goal

Improve the existing admin audit log page so operators can understand entries without decoding raw IDs and internal action codes.

This increment covers three concrete outcomes:

1. The audit log list returns operator user identity fields and target user identity fields when the target is a user.
2. The audit log page renders action module and action type in Chinese instead of raw codes.
3. The system has a clear, explicit mapping of which modules and actions currently generate admin audit logs.

This is an enhancement to the already implemented audit log page. It does not change the page route, menu entry, permission model, or the underlying audit write points.

## Current State

The current page at `/console/audit-logs` is already wired into:

- route
- sidebar menu
- sidebar permission snapshots
- action-permission gating with `audit_management.read`
- list querying against `GET /api/admin/audit-logs`

The remaining usability gaps are:

- the backend list API returns only `operator_user_id` and `target_id`
- the page shows raw `action_module` and `action_type` codes
- there is no visible reference on the page that tells operators which codes are expected

## Scope

### In Scope

- Extend the audit log list API response shape with username/display-name fields.
- Keep the underlying `admin_audit_logs` table schema unchanged.
- Render operator and target identity in a human-readable format.
- Render `action_module` and `action_type` using Chinese labels.
- Document the currently supported audit module/action combinations in code.
- Add or update backend and frontend tests for the new behavior.

### Out of Scope

- New audit write points
- New audit filters
- Audit detail modal
- `before_json` / `after_json` display
- Export
- Internationalizing the new Chinese mappings into multiple frontend locale files in this increment

## Approaches Considered

### Option 1: Backend-enriched list response plus frontend display mapping

The backend joins `users` for operator and target identities, then returns enriched list items. The frontend only formats and displays them.

Pros:

- single request per page load
- no client-side identity guessing
- clean API contract
- easiest to test

Cons:

- requires a small response-type change in backend service/controller tests

### Option 2: Frontend performs extra identity lookups

The page would fetch audit logs first, then do one or more extra user lookups.

Pros:

- no backend response change

Cons:

- N+1 or batch-lookup complexity
- no existing dedicated batch user lookup for this page
- harder to keep consistent with permission scope

### Option 3: Show only operator username, keep target as raw ID

Pros:

- smallest backend change

Cons:

- does not satisfy the user request well enough when `target_type = user`

### Recommendation

Use Option 1.

It matches the current architecture, keeps the UI simple, avoids extra requests, and gives a stable contract for future detail-view enhancements.

## Chosen Design

### Backend API Contract

Change the list service/controller output from raw `[]model.AdminAuditLog` items to a dedicated response item type that embeds the original audit fields plus display fields:

- `operator_username`
- `operator_display_name`
- `target_username`
- `target_display_name`

Rules:

- `operator_*` are populated when the operator user record exists.
- `target_*` are populated only when `target_type = user` and the target user record exists.
- for non-user targets, `target_*` remain empty.
- the database schema is unchanged; all enrichment is query-time only.

### Backend Query Design

Implement the list query with `LEFT JOIN users AS operator_users` and `LEFT JOIN users AS target_users`.

Join rules:

- operator join: `operator_users.id = admin_audit_logs.operator_user_id`
- target join: only when `admin_audit_logs.target_type = 'user'` and `target_users.id = admin_audit_logs.target_id`

This approach is compatible with SQLite, MySQL, and PostgreSQL because it uses plain joins through GORM query composition rather than database-specific SQL features.

### Frontend Table Design

Keep the page layout, filters, pagination, and permission behavior unchanged.

Replace the current raw identity columns with display-oriented rendering:

- `操作人`
  - render as `username`
  - if `display_name` exists and differs from `username`, append `（display_name）`
  - append `#ID`
- `目标`
  - when `target_type = user`, render the same identity format plus `#ID`
  - otherwise render `目标类型 + #target_id`

Keep these columns:

- `动作模块`
- `动作类型`
- `IP`
- `时间`

The goal is a compact list that is readable without opening detail views.

### Chinese Mapping Design

Render action codes using a frontend mapping file local to the audit page.

Current module mapping:

- `admin_management` -> `管理员管理`
- `agent` -> `代理管理`
- `user_management` -> `用户管理`
- `permission` -> `权限管理`
- `quota` -> `额度管理`

Current action mapping:

- `create` -> `创建`
- `update` -> `更新`
- `enable` -> `启用`
- `disable` -> `禁用`
- `delete` -> `删除`
- `bind_profile` -> `绑定权限模板`
- `clear_profile` -> `清空权限模板`
- `adjust` -> `额度调整`
- `adjust_batch` -> `批量额度调整`

Fallback rule:

- unmapped codes render as the original raw value, not `-`

This prevents the page from hiding future audit actions before the mapping is updated.

### Audit Coverage Reference

The current codebase writes admin audit logs for these module/action pairs:

- `admin_management`
  - `create`
  - `update`
  - `enable`
  - `disable`
- `agent`
  - `create`
  - `update`
  - `enable`
  - `disable`
- `user_management`
  - `create`
  - `update`
  - `enable`
  - `disable`
  - `delete`
- `permission`
  - `bind_profile`
  - `clear_profile`
- `quota`
  - `adjust`
  - `adjust_batch`

This list should be captured in a small frontend helper local to `AdminAuditLogsPageV1` so that the display mapping is explicit, testable, and easy to update with future audit write points.

## Files To Change

Backend:

- `service/audit_service.go`
- `controller/admin_audit.go`
- `controller/admin_audit_test.go`

Frontend:

- `web/src/pages/AdminAuditLogsPageV1/index.jsx`
- `web/src/pages/AdminAuditLogsPageV1/source.test.js`

Possible helper extraction if needed:

- `web/src/pages/AdminAuditLogsPageV1/display.js`
- `web/src/pages/AdminAuditLogsPageV1/display.test.js`

## Testing Strategy

### Backend

Add a failing test first that proves the list API returns:

- `operator_username`
- `operator_display_name`
- `target_username`
- `target_display_name`

Test both:

- `target_type = user`
- non-user target, where target display fields should stay empty

### Frontend

Add failing tests first for the display helpers or source contract:

- action module code renders Chinese
- action type code renders Chinese
- unknown codes fall back to raw value
- operator/target display formatting includes username and ID

### Verification

After implementation:

- run focused Go tests for admin audit logs
- run focused node tests for the audit page
- run the existing frontend build

## Risks and Mitigations

### Risk 1: Missing target user record

If the target user was deleted or the record is unavailable, the target display fields can be empty.

Mitigation:

- frontend falls back to `target_type + #target_id`

### Risk 2: Mapping drift as new audit actions are added

Mitigation:

- use raw-code fallback
- keep the mapping table close to the audit page
- document the current write points explicitly in code comments or a helper constant

### Risk 3: Response shape change breaks existing tests

Mitigation:

- keep all original audit fields unchanged
- only add new optional JSON fields to each item

## Acceptance Criteria

This enhancement is complete when:

- the audit list API returns operator and target display fields
- the page shows usernames instead of only raw IDs
- the page shows Chinese labels for action module and action type
- unknown action codes still remain visible via raw fallback
- focused backend and frontend tests pass
- the current audit module/action coverage can be listed directly from the audit-page helper code without manual grep
