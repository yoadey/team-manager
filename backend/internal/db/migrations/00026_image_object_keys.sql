-- +goose Up

-- Move team/user photos and team logos out of Postgres BYTEA columns into an
-- S3-compatible object store (internal/storage). Going forward, uploads write
-- an object key here instead of bytes into *_data -- HasPhoto/HasLogo (in the
-- API response) and image delivery both switch to keying off *_object_key
-- being non-empty rather than *_data.
--
-- *_data/*_mime are deliberately NOT dropped or backfilled in this migration
-- (see openspec/changes/move-images-to-object-storage's design.md Non-Goals):
-- any team/user whose photo predates this release keeps its bytes in *_data
-- with no *_object_key, and will appear photo-less until a follow-up
-- migration backfills *_data into the object store and sets *_object_key
-- (tasks.md section 8, deferred). No data is lost -- *_data still holds the
-- original bytes, ready for that backfill.
ALTER TABLE users ADD COLUMN photo_object_key TEXT;
ALTER TABLE teams ADD COLUMN photo_object_key TEXT;
ALTER TABLE teams ADD COLUMN logo_object_key TEXT;

-- +goose Down

ALTER TABLE teams DROP COLUMN IF EXISTS logo_object_key;
ALTER TABLE teams DROP COLUMN IF EXISTS photo_object_key;
ALTER TABLE users DROP COLUMN IF EXISTS photo_object_key;
