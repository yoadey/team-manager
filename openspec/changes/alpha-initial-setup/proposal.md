## Why

The project has only ever been released under an `alpha` tag and has never been
deployed anywhere real — every "upgrade path" concern the codebase currently
carries (29 incremental goose migrations, a deferred BYTEA→S3 image backfill,
docs describing rollback/recovery of migrations already "in production") is
premature for a system with no live data to preserve. Treating this as an
**initial setup** rather than an ongoing production system removes that
accumulated weight before the first real deployment:

- `backend/internal/db/migrations/` has 29 files (`00001`..`00029`) that
  incrementally evolve the schema, including a two-phase image-storage
  migration (`00026` adds `*_object_key` columns; dropping the legacy
  `*_data`/`*_mime` BYTEA columns is deferred to a follow-up backfill change,
  per `openspec/changes/move-images-to-object-storage/tasks.md` §8). Since no
  alpha deployment holds real image bytes in `*_data`, that backfill is moot —
  the columns can be dropped outright and several `has_photo` queries
  (`internal/{polls,finances,events,notifications,stats,absences}/repository.go`,
  `internal/db/queries/news.sql`) still OR against `photo_data IS NOT NULL` as
  a legacy fallback that can be removed too.
- `docs/operations.md` documents migration-rollback/recovery scenarios
  (e.g. "every existing deployment already has 00004 applied") and an image
  "Data migration note" about the deferred backfill — both describe a
  production history that never actually happened.
- Separately, and unrelated to the migration/storage cleanup:
  README.md's feature list claimed a shipped **"OIDC-Login"**, while the same
  README's "Noch offen" section said the opposite (mock login only), and the
  actual code confirms the mock/demo state — `internal/auth/handler.go`'s
  `ListProviders` is hardcoded to return only `password`, no OIDC library is
  in `go.mod`, and the login screen rendered a static, always-on
  `"OIDC · Authorization Code Flow + PKCE"` footer regardless of which backend
  it talked to. Worse, `docs/end-user/erste-schritte.md` — the real end-user
  guide — told actual club members to "log in via an Identity Provider (SSO)"
  and that there is no self-service registration, both wrong: the real login
  is email/password, and self-service registration (`POST /auth/register`)
  already shipped. `docs/end-user/daten-und-datenschutz.md`,
  `docs/gdpr-data-subject-rights.md`, and `SECURITY.md` similarly asserted
  accounts "authenticate via OIDC" as if it were live, when it is schema
  scaffolding (`oidc_accounts` table, nullable `password_hash`) with no
  code path ever populating it.

## What Changes

- Consolidate the 29 goose migrations into a single initial-setup migration
  (`00001_init.sql`) reflecting the schema an alpha deployment needs from day
  one — including `photo_object_key`/`logo_object_key` baked in directly, and
  **without** the legacy `photo_data`/`photo_mime`/`logo_data`/`logo_mime`
  BYTEA columns (no backfill needed, since no alpha install holds real image
  data). Delete migrations `00002`-`00029`.
- Update every repository/query `has_photo` computation to key off
  `*_object_key IS NOT NULL` only, dropping the `photo_data IS NOT NULL`
  fallback.
- Update `docs/operations.md` (drop the migration-rollback recovery example
  tied to historical migration numbers, and the image "Data migration
  note"/backfill caveat) and `move-images-to-object-storage/tasks.md` §8
  (mark superseded by this change) to match the squashed, backfill-free state.
- Correct every place documentation or UI asserted OIDC/SSO login is live:
  README.md's feature list and "Noch offen" section, the login screen's
  static OIDC footer, `docs/end-user/erste-schritte.md` (real email/password
  + self-registration flow, invite-link flow), `docs/end-user/daten-und-datenschutz.md`,
  `docs/gdpr-data-subject-rights.md`, `SECURITY.md`, and a stray comment in
  `NavSheets.tsx` — replacing "authenticate via OIDC" framing with "schema
  supports a future OIDC-only account; no OIDC integration exists yet".

## Capabilities

### New Capabilities
- `deployment-setup`: a fresh install applies one initial-setup migration
  instead of replaying 29 incremental ones.

### Modified Capabilities
- `image-storage`: images are stored only via object-store keys; no legacy
  BYTEA fallback or backfill path remains.
- `end-user-docs`: the getting-started guide accurately describes the
  shipped login methods (email/password, self-service registration) and no
  longer claims a live OIDC/SSO integration that does not exist. (README.md,
  the login screen's OIDC footer, `docs/gdpr-data-subject-rights.md`,
  `SECURITY.md`, and a `NavSheets.tsx` comment carry the same
  documentation-accuracy fix but aren't part of the `end-user-docs` capability
  itself — see Impact.)

## Impact

- `backend/internal/db/migrations/00001_init.sql` (rewritten), `00002`-`00029`
  (deleted).
- `backend/internal/{polls,finances,events,notifications,stats,absences}/repository.go`,
  `backend/internal/db/queries/news.sql` (+ regenerated `internal/db/gen/news.sql.go`),
  `backend/internal/db/gen/models.go`, `backend/internal/teams/model.go`,
  `backend/internal/auth/repository.go` (drop `*_data`/`*_mime` fields and the
  `has_photo` OR-fallback).
- `docs/operations.md`, `openspec/changes/move-images-to-object-storage/tasks.md`.
- `README.md`, `frontend/src/features/auth/components/Login.tsx`,
  `docs/end-user/erste-schritte.md`, `docs/end-user/daten-und-datenschutz.md`,
  `docs/gdpr-data-subject-rights.md`, `SECURITY.md`,
  `frontend/src/features/team/components/NavSheets.tsx` (docs/comment fixes,
  already applied ahead of this proposal being written — see tasks.md).
- CI: `backend-migration-rollback`, `backend-migration-safety`,
  `backend-openapi-drift` (sqlc regen), golangci-lint, `go test`; frontend
  lint/typecheck/test/build for the `Login.tsx`/`NavSheets.tsx` edits.
