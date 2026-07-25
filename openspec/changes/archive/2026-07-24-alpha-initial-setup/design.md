## Context

Nothing has ever been deployed beyond the `alpha` tag, so there is no live
database whose migration history must be preserved and no real image bytes
sitting in `*_data` columns that would need a backfill. The repo can be
treated as if this were the *first* setup rather than the accumulation of 29
historical steps.

`backend/internal/db/migrations/00001_init.sql` already uses the
`-- +goose NO TRANSACTION` + `CREATE TABLE IF NOT EXISTS`-style pattern the
rest of the migrations converge on, and `00006_team_scoped_indexes.sql` shows
the established safe non-PK index pattern
(`CREATE INDEX CONCURRENTLY IF NOT EXISTS ...` under `NO TRANSACTION`). The
squashed migration follows the same two conventions rather than inventing a
new one.

No Docker daemon is available in this sandbox (consistent with prior changes'
notes), so the squashed migration cannot be schema-diffed against a live
Postgres locally; correctness instead comes from a careful manual fold of
each of the 29 files' net effect (every `ADD COLUMN`, `ADD CONSTRAINT`,
`CREATE INDEX`, and backfill `UPDATE`) into the rewritten `00001_init.sql`,
cross-checked column-by-column and index-by-index against the original files
before they're deleted. CI's `backend-migration-rollback` job (up → down-to-0
→ up against a real `postgres:17` service container) is the actual
executable check that the squashed file is syntactically and semantically
sound.

## Goals / Non-Goals

**Goals:**
- One migration file represents the schema a brand-new alpha install needs;
  no dead `*_data`/`*_mime` columns, no two-phase backfill plan.
- Every `has_photo` computation reflects the current (only) source of truth:
  `*_object_key IS NOT NULL`.
- Docs and UI never assert a login method (OIDC/SSO) that doesn't exist in
  code; the end-user guide describes the actual invite → register/login flow.

**Non-Goals:**
- Building real OIDC/SSO (explicitly out of scope; tracked as "Noch offen").
- Web push notifications (user is handling separately).
- Touching `oidc_accounts`/nullable `password_hash` schema scaffolding itself
  — it's harmless, forward-compatible, and not what's inaccurate; only the
  docs/UI claiming it's *live* are being fixed.

## Decisions

- **Single squashed migration, not a "true" from-scratch rename.** Keep the
  filename `00001_init.sql` (goose's convention, and what every fresh
  `migrate up` will run first); delete `00002`-`00029` entirely rather than
  leaving empty/no-op stand-ins, since there is no deployed environment whose
  `goose_db_version` table already recorded those version numbers.
- **No dual-write compatibility window.** Because this is a pre-first-deploy
  reset, there's no rolling-upgrade concern (docs/operations.md's "Rolling
  upgrades & schema-changing migrations" section, which assumes replicas
  running old and new binary versions simultaneously against a live DB,
  doesn't apply to a squash that ships before the first deploy).
- **Fold non-PK indexes as `CREATE INDEX CONCURRENTLY IF NOT EXISTS` under
  `-- +goose NO TRANSACTION`**, matching `00006`'s existing pattern, even
  though the table is empty at creation time — this keeps the file compatible
  with the `backend-migration-safety` CI lint, which flags any *changed*
  migration file's bare `CREATE INDEX` unconditionally (it has no notion of
  "table is empty," only of the SQL statement shape), and since the squashed
  file counts as "changed" for that job (it's rewritten, not new), every
  index in it is linted as if it were live.
- **Inline constraints stay inline.** `PRIMARY KEY`/`UNIQUE`/`CHECK`/
  `FOREIGN KEY`/`NOT NULL` declared as part of `CREATE TABLE` (not via a
  separate `ALTER TABLE ... ADD CONSTRAINT`) never match the safety lint's
  `ADD CONSTRAINT`/`ALTER COLUMN ... SET NOT NULL` patterns, so the squashed
  file's inline constraints don't need the `NOT VALID` two-step dance the
  original incremental migrations (`00007`, `00016`, `00018`) used — those
  existed specifically because they altered a table that might already hold
  rows in a live deployment, which no longer applies here.
- **Drop `*_data`/`*_mime` outright, no backfill migration.** `photo_data`,
  `photo_mime`, `logo_data`, `logo_mime` are removed from the schema entirely;
  every `has_photo`/`has_logo` computation drops its
  `OR photo_data IS NOT NULL` fallback and keys solely off
  `*_object_key IS NOT NULL`.
- **OIDC/SSO documentation:** state plainly, everywhere it was previously
  implied otherwise, that no OIDC/SSO integration exists; the nullable
  `password_hash`/`oidc_accounts` schema is forward-compatible scaffolding
  for a possible future feature, not evidence of one shipping today.

## Risks / Trade-offs

- A hand-folded 29-file merge risks silently dropping a column, constraint,
  or index — mitigated by an explicit statement-by-statement tally (every
  `ADD COLUMN`/`ADD CONSTRAINT`/`CREATE INDEX`/`CREATE TABLE` across
  `00001`-`00029`) cross-checked against the rewritten file's contents before
  the old files are deleted, plus CI's migration-rollback job as the final
  executable gate.
- Any other environment that already applied some subset of `00001`-`00029`
  against a real database would desync (goose would see version 1 as already
  applied and skip re-running it, permanently missing anything folded in from
  `00002`+). Acceptable here specifically because nothing has ever been
  deployed beyond `alpha`.
