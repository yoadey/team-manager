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
- [x] 4.1 Persist/read `*_object_key` instead of `*_data` (teams + user photo). `EraseUser` also nulls `photo_object_key` and best-effort deletes the S3 object on account erasure (GDPR Art. 17 — not explicitly called out in tasks.md, but required by the same invariant `photo_data`'s nulling already served).
- [x] 4.2 Upload: validate/scale (keep `readMultipartImage`) → `store.Put` → store key; on DB error delete the orphaned object
- [x] 4.3 GET endpoints: verify membership, then 302-redirect to `store.PresignGet` (short TTL)
- [x] 4.4 Delete endpoints: `store.Delete` + null the column
- [x] 4.5 (Added beyond the original list, needed to satisfy 5.2's "enable showing other members' photos"): new `GET /teams/{teamId}/users/{userId}/photo` endpoint (`members` package, `x-rbac-module: members`) — the generic per-member photo lookup that closes the proposal's "cannot show other members' photos" gap.

## 5. Spec & frontend
- [x] 5.1 Update `openapi/openapi.yaml` image operations (302 + `Location`); run `make generate` and repo-root `make generate-ts`, commit generated output
- [x] 5.2 Update `src/api/map.ts` + photo components; enable showing other members' photos. Implemented via `memberPhotoUrl()` in `map.ts`, threaded through `mapMember`/`mapAttendanceRow`/`mapEventComment`/`mapAbsence`/`mapMemberStat` (explicit `teamId` param) and `mapNewsItem`/`mapPenaltyAssignment`/`mapContribution`/`mapNotification` (their schemas already embed `teamId`), calling the new `GET /teams/{teamId}/users/{userId}/photo` from 4.5. One deliberate exception: `PollOption.voters` carries no `userId` (anonymous-poll privacy) and so still resolves to `null` — closing that would need its own schema change with privacy implications, out of scope here.

## 6. Ops
- [x] 6.1 Add MinIO service to `docker-compose.yml` (dev-only)
- [x] 6.2 Wire S3 config in Helm `values*.yaml` from `existingSecret`; verify NetworkPolicy S3 egress. The egress rule was previously gated on `backup.enabled && backup.s3.enabled` only; broadened to also fire on the new `images.s3.enabled`, since the app's own image-storage S3 client is independent of the backup CronJob.
- [x] 6.3 Add an object-storage section to `docs/operations.md`

## 7. Verification
- [x] 7.1 `make generate` + `make generate-ts` produce no diff (drift gates green) — verified by regenerating twice in a row and diffing (stable/idempotent); `zod.gen.ts` (also emitted by `generate-ts` but not covered by the CI drift gate, and not consumed by any frontend code) was additionally re-run through `prettier` to match the repo's committed style, same as the pre-commit hook would do.
- [~] 7.2 `make lint` (ran via the pre-installed `golangci-lint` directly — the pinned-version `go install` in `make tools` failed to resolve in this sandbox, an environment issue unrelated to this change) and `make test -short` (unit tests) are green. `make vuln`/`govulncheck` could not run: this sandbox's egress policy blocks `vuln.go.dev` (403) — untested here, expected to run normally in real CI.
- [ ] 7.3 Migration up→down→up: **not run**. This sandbox's egress policy blocks Docker Hub image pulls (403 from `production.cloudfront.docker.com`), so `testutil.NewTestDB`'s testcontainers-backed integration tests (and thus this check) could not execute. The migration itself is a plain nullable `ADD COLUMN`/`DROP COLUMN IF EXISTS` — the same class already exercised by 00016/00018/00023 — and passes CI's static migration-safety lint (verified: no unsafe DDL patterns). Real CI should confirm this end-to-end.
- [ ] 7.4 Local MinIO smoke test: **not run**, same Docker Hub access block as 7.3 (`docker compose up` needs to pull `postgres`/`minio`/`node` images). `docker compose config` was used instead to confirm the compose file parses and resolves correctly.
- [x] 7.5 Frontend `typecheck`/`test`/`build` green; other members' photos resolve to a URL wherever `hasPhoto`+`teamId`+`userId` are available (verified via new `map.test.ts` cases, not a live browser smoke test).

## 8. Deferred (separate follow-up change)
- [ ] 8.1 Backfill existing BYTEA to S3; then migration `00027` drops the `*_data` columns
