# GDPR Data-Subject Rights — Design & Implementation Plan

Status: **proposed** · Owner: _unassigned_ · Last updated: 2026-06-26

This document is the implementation plan for GDPR Articles 15 (right of access /
export) and 17 (right to erasure) for the Teamverwaltung backend. It is written
as a ready-to-execute spec; it is **not yet implemented** because erasure
semantics require a product/legal decision (see "Open decision" below) and the
work spans the OpenAPI contract, a DB migration, and a new service path.

## Why this is needed

The app stores personal data of club members in `users` (`name`, `email`,
`phone`, `birthday`, `address`, `photo_data`) and in linked records
(`memberships`, `attendance`, `event_comments`, `absences`, `news`,
`finances/*`). For an EU-facing deployment, a data subject must be able to
obtain a copy of their data and request its deletion. There is currently no
endpoint or process for either.

## Scope

| Right | Endpoint (proposed) | Auth |
|-------|---------------------|------|
| Access / export (Art. 15) | `GET /api/v1/auth/me/data-export` | session owner |
| Erasure (Art. 17) | `DELETE /api/v1/auth/me` | session owner (with re-auth) |

Both are **self-service** (operate on the authenticated user only). Admin-driven
erasure of another member is out of scope for this iteration.

## 1. Data export (Art. 15) — low risk, do first

Returns a single JSON document with everything tied to the user across all
teams: profile, memberships + roles, attendance, comments, absences, authored
news, finance rows referencing them, and votes.

- **OpenAPI**: add `GET /auth/me/data-export` returning `application/json`
  (a `UserDataExport` schema) and a `Content-Disposition: attachment` header.
- **Layering**: `auth.Handler` → new `Service.ExportUserData(ctx, userID)` →
  read-only repository queries fanned out across the existing per-feature
  repositories (reuse them rather than writing raw SQL in `auth`).
- **No schema change required.**

## 2. Erasure (Art. 17) — requires the decision below

### Open decision — anonymize vs hard-delete

Some referenced data **cannot simply be cascade-deleted**:

- **Financial records** (`transactions`, `penalty_assignments`, `contributions`)
  often must be retained for accounting/tax reasons even after a member leaves.
- **Attendance / event history** is shared team data; hard-deleting it distorts
  other members' statistics.

`users` currently has `ON DELETE CASCADE` from `memberships`, `attendance`,
`event_comments`, etc. A raw `DELETE FROM users` would therefore destroy shared
and legally-retained records. **Recommended approach: anonymize, don't delete.**

> **Recommended:** Replace direct identifiers on the `users` row with neutral
> values (`name = 'Gelöschtes Mitglied'`, `email = 'deleted+<uuid>@invalid'`,
> `phone/birthday/address = NULL`, `photo_data = NULL`, `password_hash = NULL`),
> delete all sessions, and add a `deleted_at` marker. Referenced rows keep their
> foreign key but no longer resolve to identifiable personal data. Free-text
> fields that may contain personal data (`event_comments.text`,
> `absences.reason`) are blanked.

This satisfies erasure (the person is no longer identifiable) while preserving
the integrity of shared and retained records — the standard GDPR pattern for
this conflict. The alternative (true hard delete) needs explicit product/legal
sign-off and changes to the financial-retention requirements.

### Migration (`00003_user_erasure.sql`)

```sql
-- +goose Up
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
-- partial index so the login lookup can skip anonymized accounts cheaply
CREATE INDEX ON users (email) WHERE deleted_at IS NULL;
-- +goose Down
DROP INDEX IF EXISTS users_email_idx;  -- name as generated; verify before applying
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
```

### Service

`Service.EraseUser(ctx, userID)` runs in a single transaction:
1. Blank PII columns on `users`, set `deleted_at = now()`.
2. `NULL`/blank free-text PII in `event_comments`, `absences`.
3. `DELETE FROM sessions WHERE user_id = $1`.
4. Leave membership/attendance/finance foreign keys intact.

`auth.Login` and `ValidateToken` must reject accounts where `deleted_at IS NOT
NULL` (add the predicate to `FindUserByEmail` / `FindUserByID`).

### Hardening

- Require recent re-authentication (password re-entry) before erasure to prevent
  a hijacked session from deleting the account.
- Emit an audit-log event (`user.erased`, actor + timestamp) — ties into the
  separate "audit log" finding.

## Frontend

- `serviceLayerReal.ts`: add `auth.exportData()` (triggers a file download) and
  `auth.deleteAccount()`; mirror in the mock `serviceLayer.ts`.
- Add a "Daten & Datenschutz" section in account settings with export + delete
  actions (delete behind a confirm dialog).

## Implementation checklist

- [ ] Product/legal sign-off on **anonymize vs hard-delete**.
- [ ] Add the two paths + schemas to `openapi/openapi.yaml`; run `make generate`.
- [ ] Migration `00003_user_erasure.sql`.
- [ ] `auth.Service.ExportUserData` + `EraseUser` (+ repo queries) with unit tests.
- [ ] Exclude `deleted_at` accounts from login lookups.
- [ ] Frontend export/delete UI + service-layer methods (both mock and real).
- [ ] Update `SECURITY.md` / privacy docs with the retention statement.
