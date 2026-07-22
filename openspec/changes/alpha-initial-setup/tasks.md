## 1. OIDC/SSO documentation accuracy (independent of the schema work below)
- [x] 1.1 README.md: remove "OIDC-Login" from the shipped feature list;
      rewrite "Noch offen" to state plainly that no OIDC/SSO exists yet, and
      that the mock-mode provider buttons are a demo convenience only
- [x] 1.2 `frontend/src/features/auth/components/Login.tsx`: remove the
      always-on "OIDC · Authorization Code Flow + PKCE" footer (false
      regardless of which backend is in use)
- [x] 1.3 `docs/end-user/erste-schritte.md`: replace the "log in via Identity
      Provider / SSO, no external registration" claim with the actual flow
      (email + password login; self-service registration with email
      verification; invite link joins the team after login/registration)
- [x] 1.4 `docs/end-user/daten-und-datenschutz.md`: drop the false
      "no password because login is via Identity Provider" justification for
      why account deletion only asks for email
- [x] 1.5 `docs/gdpr-data-subject-rights.md`: reword "accounts authenticate
      primarily via OIDC" (in the summary and the "Hardening/confirmation"
      section) to describe reality — `password_hash` is nullable
      forward-compatible scaffolding, no OIDC integration exists
- [x] 1.6 `SECURITY.md`: same reframe for its "Confirmation without a
      password" bullet
- [x] 1.7 `frontend/src/features/team/components/NavSheets.tsx`: fix the
      account-erasure comment's "since accounts may be OIDC-only" framing

## 2. Squash migrations into a single initial-setup migration
- [x] 2.1 Read and tally every `ADD COLUMN`/`ADD CONSTRAINT`/`CREATE INDEX`/
      `CREATE TABLE`/backfill `UPDATE` across `00001`-`00029`
- [x] 2.2 Rewrite `00001_init.sql` as the single migration producing that
      exact end schema, baking `photo_object_key`/`teams.photo_object_key`/
      `teams.logo_object_key` in directly and omitting
      `photo_data`/`photo_mime`/`logo_data`/`logo_mime` entirely; non-PK
      indexes as `CREATE INDEX CONCURRENTLY IF NOT EXISTS` (matching `00006`'s
      existing pattern) so the `backend-migration-safety` lint stays green
- [x] 2.3 Write the matching `-- +goose Down` (drop everything the Up section
      creates)
- [x] 2.4 Delete `00002_audit_columns.sql` through
      `00029_email_verification_tokens_indexes.sql`
- [x] 2.5 Cross-check: every column/index/constraint present across the
      original 29 files also exists in the rewritten `00001_init.sql` (or was
      deliberately dropped per §3 below) — verified for real, not just by
      inspection: started a local `postgresql-16` cluster, ran
      `goose up` → `down-to 0` → `up` against it (all clean), then diffed
      `\dt`/`\di`/`\d <table>` output against the tally from 2.1 — every
      table, index, FK (including the auto-named
      `penalty_assignments_penalty_id_fkey`), and named CHECK constraint
      matched exactly

## 3. Drop legacy image columns and their fallback reads
- [x] 3.1 `internal/polls/repository.go`, `internal/finances/repository.go`,
      `internal/events/repository.go`, `internal/notifications/repository.go`,
      `internal/stats/repository.go`, `internal/absences/repository.go`,
      `internal/db/queries/news.sql`: drop the `OR photo_data IS NOT NULL`
      (and `length(photo_data) > 0`) fallback from every `has_photo`
      computation — key off `*_object_key IS NOT NULL` only
- [x] 3.2 `internal/auth/repository.go`: drop the `photo_data = NULL,
      photo_mime = NULL` clause from the photo-delete statement (column no
      longer exists); also fixed a stale comment referencing a
      `FindUserPhotoByID` function and a `photo_data BLOB` that no longer
      exist (photo delivery is presigned-URL-only, per
      `move-images-to-object-storage`)
- [x] 3.3 `sqlc generate` (via `make generate`) to drop
      `PhotoData`/`PhotoMime`/`LogoData`/`LogoMime` from
      `internal/db/gen/models.go`; removed the same dead, never-read
      `PhotoData []byte` field from `internal/teams/model.go`'s `MemberRow`
- [x] 3.4 `make generate` (oapi-codegen + genrbac) produces no unrelated diff
      (verified: `internal/gen/api.gen.go` and `rbac_table.gen.go` are
      byte-identical; only `internal/db/gen/models.go` and `news.sql.go`
      changed, exactly matching the column rename)

## 4. Docs: drop the now-moot migration/backfill narrative
- [x] 4.1 `docs/operations.md`: rewrote "Recovering from a migration killed
      mid-flight" — the squashed migration's `CREATE TABLE` block is one
      atomic implicit transaction (goose `StatementBegin`/`StatementEnd`),
      so the old 00004-specific recovery runbook no longer applies; replaced
      with general forward-looking guidance for future migrations. Also
      de-referenced the deleted `00008_amount_cents.sql` in the rolling-
      upgrades example (generic description of the same hazard instead)
- [x] 4.2 `docs/operations.md`: removed the image-storage "Data migration
      note" / legacy-BYTEA-backfill caveat
- [x] 4.3 `openspec/changes/move-images-to-object-storage/tasks.md` §8:
      marked "Backfill existing BYTEA to S3" superseded by this change
- [x] 4.4 (found during implementation) `docs/gdpr-data-subject-rights.md`
      still named the dropped `photo_data` column twice (updated to
      `photo_object_key`) and cited the now-deleted `00003_user_erasure.sql`
      by filename in two places (reworded to describe the schema directly)
- [x] 4.5 (found during implementation) `backend/internal/teams/repository_test.go`
      had a test (`TestBackfillMemberRolePermissionsMigration` +
      its `runMigration00023Up` helper) that read `00023_backfill_...sql`
      directly off disk by filename — deleted along with its now-unused
      imports (`os`, `path/filepath`, `runtime`, `strings`), since the
      one-time data backfill it tested is moot for a fresh install (no
      pre-existing broken rows can exist)
- [x] 4.6 (found during implementation) `backend/internal/finances/repository_test.go`
      and `backend/internal/jobs/retention.go` had comments citing the
      now-deleted `00025_penalty_assignment_amount_snapshot.sql`/
      `00004_audit_log.sql` by filename; reworded to describe the invariant
      directly instead. Also fixed the underlying inconsistency
      `retention.go`'s comment was working around: the squashed
      `00001_init.sql`'s `audit_log` table comment used to say retention is
      "handled at the infrastructure layer" while `RetentionWorker` actually
      owns it — since this is a fresh, not-yet-applied migration (unlike its
      predecessor, editing it isn't a goose-checksum hazard), corrected the
      comment to match reality instead of documenting around the mismatch

## 5. Verification
- [x] 5.1 `cd backend && make generate` produces no diff on a second run
      (confirmed)
- [x] 5.2 `cd backend && make test` (`go build ./...`, `go vet ./...`,
      `gofmt -l .`, `go test ./...`) all clean/green. `golangci-lint` itself
      could not be installed in this sandbox (`go install .../golangci-lint@v2.12.2`
      fails to resolve — an environment/proxy quirk unrelated to this change;
      CI installs it via `golangci-lint-action`, a different path) — left for
      CI's `golangci-lint` job to confirm.
- [x] 5.3 `goose up` → `down-to 0` → `up` (the exact sequence
      `backend-migration-rollback` runs) executed locally against a real
      `postgres:16` cluster (no Docker daemon in this sandbox, so a real
      `postgres:17` container as CI uses wasn't available, but the DDL this
      migration runs has no version-16-vs-17 dependency) — all three steps
      succeeded
- [x] 5.4 `backend-migration-safety` lint's rules manually walked against the
      rewritten `00001_init.sql`: no bare `CREATE INDEX` (all
      `CONCURRENTLY`), no `ALTER TABLE ... ADD CONSTRAINT`/`ALTER COLUMN ...
      SET NOT NULL`/`... TYPE` statements (constraints are inline `CREATE
      TABLE` column/table constraints, which the lint's `ADD `-prefixed
      patterns don't match), `CONCURRENTLY` usage has the required
      `-- +goose NO TRANSACTION` annotation
- [x] 5.5 `internal/gen/api.gen.go`/`rbac_table.gen.go` unaffected (openapi.yaml
      wasn't touched); confirmed byte-identical after `make generate`
- [ ] 5.6 Frontend `npm run typecheck`, `npm test`, `npm run lint` green for
      the `Login.tsx`/`NavSheets.tsx` edits — being verified together with
      the parallel lint/TS-strictness work in `tooling-and-docs-hardening`
