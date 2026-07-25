## Why

Images are stored as `BYTEA` in Postgres (`users.photo_data`, `teams.photo_data`, `teams.logo_data`, migration `00001_init.sql`). This bloats database size, WAL and the daily `pg_dump` backups; every image delivery ties up a pgx pool connection instead of a CDN/object store; and there is no generic member-photo endpoint, so `frontend/src/api/map.ts` cannot display other members' photos. It is a known scaling dead-end, and the Helm NetworkPolicy already reserves the S3 egress port for this step.

## What Changes

- Store image bytes in an S3-compatible object store (AWS S3 / MinIO); keep only an object key in the DB.
- Deliver images via short-lived presigned GET URLs (302 redirect), gated by team membership.
- Add config, a `storage` package with an `ObjectStore` interface, migration `00026`, and MinIO for local dev.
- Update the OpenAPI image operations and regenerate both clients; close the member-photo feature gap.
- Column drop of the old `*_data` is deferred to a follow-up change after backfill.

## Capabilities

### New Capabilities
- `image-storage`: how team/user photos and logos are stored, delivered, and access-controlled.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: `internal/storage/*` (new), `config/config.go`, `teams/handler.go`, `teams/repository.go`, auth user-photo path, `openapi/openapi.yaml` + generated code, migration `00026`.
- Frontend: `src/api/map.ts` + photo components.
- Ops: Helm `networkpolicy.yaml` + `values*.yaml`, `docs/operations.md`, `docker-compose.yml` (MinIO), `CLAUDE.md` env table.
- CI: openapi-drift, migration-safety, migration-rollback, govulncheck.
