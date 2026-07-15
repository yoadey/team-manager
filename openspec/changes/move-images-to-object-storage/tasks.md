## 1. Config & dependency
- [ ] 1.1 Add pinned `github.com/minio/minio-go/v7`
- [ ] 1.2 Add S3 env vars to `config/config.go` (`S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_USE_PATH_STYLE`, optional `S3_PUBLIC_BASE_URL`); require them when `COOKIE_SECURE=true`
- [ ] 1.3 Document the new vars in `CLAUDE.md`

## 2. Storage package
- [ ] 2.1 Create `internal/storage/store.go` with an `ObjectStore` interface (`Put`/`PresignGet`/`Delete`)
- [ ] 2.2 Implement `s3store.go` (minio-go) and `fake.go` (in-memory, for tests)

## 3. Schema
- [ ] 3.1 Add migration `00026_image_object_keys.sql`: nullable `*_object_key` columns on `users`/`teams`; do NOT drop `*_data` yet; write the down migration

## 4. Handlers & repositories
- [ ] 4.1 Persist/read `*_object_key` instead of `*_data` (teams + user photo)
- [ ] 4.2 Upload: validate/scale (keep `readMultipartImage`) → `store.Put` → store key; on DB error delete the orphaned object
- [ ] 4.3 GET endpoints: verify membership, then 302-redirect to `store.PresignGet` (short TTL)
- [ ] 4.4 Delete endpoints: `store.Delete` + null the column

## 5. Spec & frontend
- [ ] 5.1 Update `openapi/openapi.yaml` image operations (302 + `Location`); run `make generate` and repo-root `make generate-ts`, commit generated output
- [ ] 5.2 Update `src/api/map.ts` + photo components; enable showing other members' photos

## 6. Ops
- [ ] 6.1 Add MinIO service to `docker-compose.yml` (dev-only)
- [ ] 6.2 Wire S3 config in Helm `values*.yaml` from `existingSecret`; verify NetworkPolicy S3 egress
- [ ] 6.3 Add an object-storage section to `docs/operations.md`

## 7. Verification
- [ ] 7.1 `make generate` + `make generate-ts` produce no diff (drift gates green)
- [ ] 7.2 `make lint`, `make test` (incl. fake-store tests), `make vuln` green
- [ ] 7.3 Migration up→down→up green; migration-safety lint green
- [ ] 7.4 Local MinIO smoke: upload → object present + key stored; GET redirects and image loads; delete removes object and nulls column
- [ ] 7.5 Frontend `typecheck`/`test`/`build` green; other members' photos display

## 8. Deferred (separate follow-up change)
- [ ] 8.1 Backfill existing BYTEA to S3; then migration `00027` drops the `*_data` columns
