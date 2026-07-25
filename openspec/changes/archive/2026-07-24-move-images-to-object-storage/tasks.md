## 1. Config & dependency
- [x] 1.1 Add pinned `github.com/minio/minio-go/v7`
- [x] 1.2 Add S3 env vars to `config/config.go` (`S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_USE_PATH_STYLE`, optional `S3_PUBLIC_BASE_URL`); require them when `COOKIE_SECURE=true`
- [x] 1.3 Document the new vars in `CLAUDE.md`

## 2. Storage package
- [x] 2.1 Create `internal/storage/store.go` with an `ObjectStore` interface (`Put`/`PresignGet`/`Delete`)
- [x] 2.2 Implement `s3store.go` (minio-go) and `fake.go` (in-memory, for tests)

## 3. Schema
- [x] 3.1 Add migration `00026_image_object_keys.sql`: nullable `*_object_key` columns on `users`/`teams`; do NOT drop `*_data` yet; write the down migration

## 4. Handlers & repositories
- [x] 4.1 Persist/read `*_object_key` instead of `*_data` (teams + user photo). Also added a new `getMemberPhoto`
      endpoint (`GET /teams/{teamId}/members/{membershipId}/photo`, membership-gated via the `members` RBAC module)
      so any team member can resolve another member's photo through the same `users/{userID}/photo` key scheme --
      closing the "cannot show other members' photos" gap named in the proposal's Why.
- [x] 4.2 Upload: validate/scale (kept `readMultipartImage`) → `store.Put` → store key; on DB error delete the
      orphaned object. Also best-effort deletes the *previous* object on replace (upload/delete) and on GDPR
      erasure (`auth.Service.EraseAccount`), so old objects don't leak in the store -- not explicitly called out in
      the design but a direct consequence of "the DB holds only an object key reference".
- [x] 4.3 GET endpoints: verify membership, then 302-redirect to `store.PresignGet` (15 min TTL, `storage.PresignTTL`)
- [x] 4.4 Delete endpoints: `store.Delete` + null the column

## 5. Spec & frontend
- [x] 5.1 Update `openapi/openapi.yaml` image operations (302 + `Location`, via a shared `PhotoRedirect` response
      component) plus the new `getMemberPhoto` operation; ran `make generate` and repo-root `make generate-ts`,
      committed generated output (`internal/gen/api.gen.go`, `internal/middleware/rbac_table.gen.go`,
      `internal/db/gen/*` picked up the new columns via sqlc, `frontend/src/api/types.gen.ts`/`zod.gen.ts`).
- [x] 5.2 Updated `src/api/map.ts`'s `mapMember` (now takes `teamId`) to resolve `hasPhoto` to a
      `/teams/{teamId}/members/{membershipId}/photo` URL; wired through `serviceLayerReal.ts`'s three `mapMember`
      call sites (list/update/setRoles). Existing photo-rendering components (`Av` in `MembersPage.tsx`/
      `MemberSheets.tsx`) needed no changes since they already render `member.photo` generically. Other
      member-photo consumers (attendance rows, comments, poll voters, finance rows, stats) still map to `null` --
      their API schemas carry no `hasPhoto` flag, which is a separate, larger change outside this proposal's scope.

## 6. Ops
- [x] 6.1 Added a `minio` service plus a one-shot `minio-init` (`minio/mc`) bucket-creation job to
      `docker-compose.yml` (dev-only); `backend` now depends on `minio-init` completing and gets `S3_*` env pointed
      at it, with `S3_PUBLIC_BASE_URL=http://localhost:9000` so browser-facing presigned URLs don't reference the
      in-network `minio:9000` hostname.
- [x] 6.2 Wired `S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY` as `existingSecret` keys in
      `helm/team-manager/templates/deployment.yaml` (both the migrate initContainer and the main container, matching
      the JWT-key pattern); added commented `S3_ENDPOINT`/`S3_REGION`/`S3_BUCKET`/`S3_USE_PATH_STYLE` placeholders
      under `values.yaml`'s `env`. Extended `networkpolicy.yaml`'s existing S3 egress rule (previously gated only on
      `backup.enabled && backup.s3.enabled`) to also open whenever `env.S3_ENDPOINT` is set, since the main app pod
      now needs S3 egress too, not just the backup CronJob.
- [x] 6.3 Added an "Object storage (image uploads)" section to `docs/operations.md` covering configuration,
      local-dev MinIO, the Kubernetes/NetworkPolicy wiring, and the legacy-data caveat (see 8.1).

## 7. Verification
- [x] 7.1 `make generate` + `make generate-ts` produce no diff (re-run twice, byte-identical output; committed)
- [x] 7.2 `make lint` (`golangci-lint run ./...`, 0 issues) and `make test` (`go test ./...`, all packages green,
      including `internal/storage`'s fake-store tests) verified locally. `make vuln`/`govulncheck` could NOT be run
      in this sandbox -- its vulnerability-DB fetch (`vuln.go.dev`) is blocked by the environment's egress proxy
      (403 Forbidden) -- so this remains to be confirmed by CI's `govulncheck` job.
- [ ] 7.3 Migration up→down→up and the migration-safety lint could not be exercised locally -- no Docker daemon is
      available in this sandbox, so `testutil.NewTestDB`-based integration tests (incl. the updated
      `TestTeamRepository_DeleteTeamPhoto_ClearsStoredPhoto`/`...DeleteTeamLogo_ClearsStoredLogo`) skip automatically
      rather than running. The migration itself only adds nullable columns (no rewrite, no lock escalation), the
      same shape CI's migration-safety lint already accepts for the immediately-preceding migration (00025). Left
      unchecked pending confirmation from CI's `backend-migration-rollback`/`backend-migration-safety` jobs.
- [ ] 7.4 Local MinIO smoke test not run for the same reason (no Docker daemon in this sandbox to run
      `docker compose up`). The equivalent path is covered by unit tests against `storage.FakeStore` (Put →
      PresignGet → Delete round-trip) and service-layer tests asserting the real upload/delete/redirect flow against
      the fake store, but a live MinIO round-trip has not been exercised.
- [x] 7.5 Frontend `npm run typecheck`, `npm test` (1135 tests, all passing), `npm run build` +
      `npm run check:bundle` (253.1 KB gzipped total, well under the 600 KB budget) all green. Manually verified via
      `map.test.ts`'s new `mapMember` cases that a member's `hasPhoto` now resolves to a per-team-member photo URL
      instead of always `null`.

## 8. Deferred (separate follow-up change)
- [x] 8.1 ~~Backfill existing BYTEA to S3; then migration `00027` drops the
      `*_data` columns~~ — superseded by `openspec/changes/alpha-initial-setup`:
      since this project has only ever shipped under an `alpha` tag and has
      never been deployed anywhere with real image data to preserve, that
      change drops `photo_data`/`photo_mime`/`logo_data`/`logo_mime` outright
      (no backfill) as part of squashing the migration history into a single
      initial-setup migration.
