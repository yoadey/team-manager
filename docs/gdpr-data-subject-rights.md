# GDPR Data-Subject Rights — Design & Implementation Plan

> Looking for the end-user explanation of data export/account deletion
> instead of the implementation details? See
> [`end-user/daten-und-datenschutz.md`](./end-user/daten-und-datenschutz.md).

Status: **implemented** (erasure by anonymization + data export) ·
Last updated: 2026-07-07 (round 32)

This document covers GDPR Articles 15 (right of access / export) and 17 (right
to erasure) for the Teamverwaltung backend. Both are now live end to end.

**Decision made:** erasure is implemented by **anonymization, not hard delete**
(see "Open decision" below — now resolved), confirmed by retyping the account
email rather than re-entering a password, so the same flow works uniformly
across every login method — including a future OIDC-only account with no
password, should OIDC ever be implemented (it currently is not; today all
accounts authenticate via email/password, either self-registered or
invite-provisioned).

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
| Erasure (Art. 17) | `DELETE /api/v1/auth/me` | session owner (confirm by retyping email) |

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

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
```

No partial index was added -- `email` already has a plain `UNIQUE` index from
`00001_init.sql`, and login/validation lookups exclude anonymized accounts via
an explicit `deleted_at IS NULL` predicate rather than an index shaped around
it.

### Service

`Repository.EraseUser(ctx, userID)` runs in a single transaction:
1. **Lock:** take `pg_advisory_xact_lock(hashtextextended(team_id, 0))` for
   every team the user belongs to, in deterministic (`team_id`) order — the
   same per-team lock key `members.SetRoles`/`RemoveMember` and
   `roles.UpdateRole`/`DeleteRole` take before mutating role assignments.
   Without this, the sole-admin guard below could race a concurrent role
   change stripping another member's settings:write: both transactions see a
   stale "another admin still exists" snapshot under READ COMMITTED and
   commit, leaving the team with zero settings:write holders.
2. **Guard:** if erasing this user would leave any team with no other living
   (`deleted_at IS NULL`) settings:write member, reject with
   `ErrSoleSettingsAdmin` (HTTP 409) instead of proceeding — erasure only
   anonymizes `users`, it does not touch `memberships`/`membership_roles`, so
   without this check the team would be left with a role assignment that
   satisfies every "last settings admin" guard elsewhere but belongs to a
   permanently unauthenticatable account, locking the team out of its own
   role/member/settings management. The caller must reassign settings:write
   to someone else (or have another admin already) before they can self-erase.
3. Blank PII columns on `users`, set `deleted_at = now()`.
4. `NULL`/blank free-text PII in `event_comments`, `attendance`, `absences`
   (attendance's free-text `reason` is personal data too — self-reported
   absence reasons like "Grippe" — and is included verbatim in the Art. 15
   export, so it is anonymized here rather than merely left FK-intact).
5. `DELETE FROM sessions WHERE user_id = $1`.
6. Leave membership/finance foreign keys intact.

`auth.Login` and `ValidateToken` must reject accounts where `deleted_at IS NOT
NULL` (add the predicate to `FindUserByEmail` / `FindUserByID`).

### Hardening / confirmation

- **No password re-entry.** `password_hash` is nullable so the schema can
  support a future OIDC-only account with no password (no OIDC integration
  exists yet — see `CLAUDE.md`), so erasure is authorized by the active
  session and confirmed by the user **retyping their account email**, which
  the server verifies (`DeleteAccountRequest.confirmEmail`). This proves
  intent and guards against an accidental or forged blind DELETE, uniformly
  across password and (future) passwordless accounts.
- Emits an `auth.account_erase` audit event (actor + outcome) — ties into the
  audit-log finding.

## Frontend

- `serviceLayerReal.ts` / `serviceLayer.ts`: `auth.deleteAccount(confirmEmail)`
  on both real and mock layers; `AppContext.deleteAccount` anonymizes then drops
  to the login screen.
- "Daten & Datenschutz" section in the `ProfileSheet` with a destructive
  "Konto löschen" action that reveals an inline confirm (retype email →
  enabled). Export action will join this section later.

## Implementation checklist

Erasure (Art. 17) — **done**:
- [x] Product decision: **anonymize**, not hard-delete; **email confirmation, no
      password** (forward-compatible with a future OIDC-only account, not
      currently implemented).
- [x] `DELETE /auth/me` + `DeleteAccountRequest` in `openapi/openapi.yaml`; regenerated.
- [x] Migration `00003_user_erasure.sql` (`deleted_at`).
- [x] `auth.Repository.EraseUser` (transactional anonymization) + `auth.Service.EraseAccount`
      (email-confirmation re-auth) with unit tests.
- [x] Exclude `deleted_at` accounts from login/validation lookups.
- [x] Session cookie cleared on erasure; `auth.account_erase` audit event
      (includes the blocking team IDs when rejected as a sole settings admin).
- [x] Guard against self-erasing while the sole living settings:write member
      of a team (`ErrSoleSettingsAdmin`, HTTP 409) — see "Service" above.
- [x] Serialize the sole-admin guard against concurrent role/membership
      changes via the shared per-team advisory lock — see "Service" step 1.
- [x] Frontend `auth.deleteAccount` on both mock and real service layers (+ tests).
- [x] Frontend UI: "Daten & Datenschutz" section in `ProfileSheet` with the
      retype-email confirm, wired via `AppContext` (+ tests).

Export (Art. 15) — **done**:
- [x] `GET /auth/me/data-export` (free-form JSON document + `Content-Disposition`).
- [x] `auth.Repository.ExportUserData` gathers profile, memberships+roles,
      attendance, comments, absences, authored news, created polls, votes,
      penalty assignments and contributions (read-only).
- [x] Handler `GetMyDataExport` + service method (+ handler tests).
- [x] Frontend `auth.exportData` (mock + real) and `AppContext.exportMyData`
      (downloads JSON); "Meine Daten exportieren" button in `ProfileSheet`.
- [x] `SECURITY.md` updated with the data-protection / retention statement.
