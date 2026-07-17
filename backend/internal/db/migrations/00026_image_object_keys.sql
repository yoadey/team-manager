-- +goose Up

-- Adds an object-store key column alongside each existing *_data BYTEA
-- column, per move-images-to-object-storage. The application switches to
-- reading/writing these columns exclusively; *_data stays untouched here
-- (nullable, no backfill) -- a follow-up change backfills existing BYTEA
-- into the object store and then drops *_data/*_mime in a later migration
-- (00027), per the same expand/contract precedent as 00016/00018/00023.
-- Under a RollingUpdate deploy, old-version pods keep reading/writing
-- *_data until they're gone, so nothing here can be NOT NULL or otherwise
-- assume every row already has an object key.
ALTER TABLE users ADD COLUMN photo_object_key TEXT;
ALTER TABLE teams ADD COLUMN photo_object_key TEXT;
ALTER TABLE teams ADD COLUMN logo_object_key TEXT;

-- +goose Down

ALTER TABLE teams DROP COLUMN IF EXISTS logo_object_key;
ALTER TABLE teams DROP COLUMN IF EXISTS photo_object_key;
ALTER TABLE users DROP COLUMN IF EXISTS photo_object_key;
