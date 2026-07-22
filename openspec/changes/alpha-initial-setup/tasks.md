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
- [ ] 2.1 Read and tally every `ADD COLUMN`/`ADD CONSTRAINT`/`CREATE INDEX`/
      `CREATE TABLE`/backfill `UPDATE` across `00001`-`00029`
- [ ] 2.2 Rewrite `00001_init.sql` as the single migration producing that
      exact end schema, baking `photo_object_key`/`teams.photo_object_key`/
      `teams.logo_object_key` in directly and omitting
      `photo_data`/`photo_mime`/`logo_data`/`logo_mime` entirely; non-PK
      indexes as `CREATE INDEX CONCURRENTLY IF NOT EXISTS` (matching `00006`'s
      existing pattern) so the `backend-migration-safety` lint stays green
- [ ] 2.3 Write the matching `-- +goose Down` (drop everything the Up section
      creates)
- [ ] 2.4 Delete `00002_audit_columns.sql` through
      `00029_email_verification_tokens_indexes.sql`
- [ ] 2.5 Cross-check: every column/index/constraint present across the
      original 29 files also exists in the rewritten `00001_init.sql` (or was
      deliberately dropped per §3 below)

## 3. Drop legacy image columns and their fallback reads
- [ ] 3.1 `internal/polls/repository.go`, `internal/finances/repository.go`,
      `internal/events/repository.go`, `internal/notifications/repository.go`,
      `internal/stats/repository.go`, `internal/absences/repository.go`,
      `internal/db/queries/news.sql`: drop the `OR photo_data IS NOT NULL`
      (and `length(photo_data) > 0`) fallback from every `has_photo`
      computation — key off `*_object_key IS NOT NULL` only
- [ ] 3.2 `internal/auth/repository.go`: drop the `photo_data = NULL,
      photo_mime = NULL` clause from the photo-delete statement (column no
      longer exists)
- [ ] 3.3 `sqlc generate` (via `make generate`) to drop
      `PhotoData`/`PhotoMime`/`LogoData`/`LogoMime` from
      `internal/db/gen/models.go`; update `internal/teams/model.go` if it
      hand-declares the same fields
- [ ] 3.4 `make generate` (oapi-codegen + genrbac) produces no unrelated diff

## 4. Docs: drop the now-moot migration/backfill narrative
- [ ] 4.1 `docs/operations.md`: remove the migration-rollback recovery
      example tied to historical migration numbers (e.g. "every existing
      deployment already has 00004 applied") — no longer applicable with a
      single initial migration and no prior deployment
- [ ] 4.2 `docs/operations.md`: remove the image-storage "Data migration
      note" / legacy-BYTEA-backfill caveat
- [ ] 4.3 `openspec/changes/move-images-to-object-storage/tasks.md` §8: mark
      "Backfill existing BYTEA to S3" as superseded by this change (columns
      dropped outright instead)

## 5. Verification
- [ ] 5.1 `cd backend && make generate` produces no diff on a second run
- [ ] 5.2 `cd backend && make lint` (golangci-lint) and `make test` green
- [ ] 5.3 `backend-migration-rollback`-equivalent (`goose up` →
      `down-to 0` → `up`) exercised in CI (no local Docker daemon available
      in this sandbox to run it locally)
- [ ] 5.4 `backend-migration-safety` lint passes on the rewritten
      `00001_init.sql` (no unsafe unconcurrent `CREATE INDEX`, no `ALTER
      TABLE ... ADD CONSTRAINT`/`SET NOT NULL` patterns)
- [ ] 5.5 `cd backend && make generate` (openapi-drift) still clean if
      `internal/gen/api.gen.go` is affected by any generated-type change
- [ ] 5.6 Frontend `npm run typecheck`, `npm test`, `npm run lint` green for
      the `Login.tsx`/`NavSheets.tsx` edits
