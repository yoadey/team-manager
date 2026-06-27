-- +goose Up
-- GDPR Art. 17 erasure by anonymization: mark anonymized accounts with a
-- timestamp so login/validation lookups can exclude them. Personal data on the
-- row itself is overwritten in place by the application (see auth.Repository.EraseUser).
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
