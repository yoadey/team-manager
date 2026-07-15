## Context

`teams/handler.go:readMultipartImage` caps uploads at 2 MB, accepts only JPEG/PNG, and scales to max 800×800, then persists bytes into `*_data BYTEA`. GET endpoints stream the bytes as `image/jpeg`. Highest migration is `00025`, so the next is `00026`.

## Goals / Non-Goals

**Goals:**
- Move image bytes out of Postgres into an S3-compatible store; DB holds only `*_object_key`.
- Presigned, membership-gated delivery; no image streaming through the app server.
- Close the "cannot show other members' photos" gap.

**Non-Goals:**
- Changing the 2 MB / MIME / 800×800 validation (kept; only the sink changes).
- Dropping the old `*_data` columns in this change (deferred until after backfill).

## Decisions

- **`github.com/minio/minio-go/v7`** (small surface, S3 + MinIO, presigning built in), pinned.
- Key scheme: `teams/{teamID}/photo`, `teams/{teamID}/logo`, `users/{userID}/photo`.
- GET endpoints return **302 redirect** to a short-TTL presigned URL; membership is checked before the URL is issued (access control stays server-side).
- Upload order: S3 put **before** DB commit; on DB error, best-effort delete the orphaned object.
- With `COOKIE_SECURE=true` (production), S3 config is required at startup, mirroring the existing JWT/cookie-key hard-gating.
- `ObjectStore` is an interface with an in-memory fake for tests.

## Risks / Trade-offs

- Two-phase data migration: backfill existing BYTEA to S3, then a later `00027` drops the columns — never in this change (no data loss).
- Presigned URLs bypass per-request auth, so TTL must be short and issuance must remain membership-gated.
- OpenAPI response shape changes (bytes → 302) require regenerating both clients and updating frontend consumers.
