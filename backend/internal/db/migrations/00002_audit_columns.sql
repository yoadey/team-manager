-- +goose Up
-- Add updated_at audit columns to core mutable tables.
-- created_at already exists; updated_at tracks the last write for compliance logging.

ALTER TABLE events
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE members
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE roles
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE news
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE polls
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE finances
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE absences
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE absences    DROP COLUMN IF EXISTS updated_at;
ALTER TABLE finances    DROP COLUMN IF EXISTS updated_at;
ALTER TABLE polls       DROP COLUMN IF EXISTS updated_at;
ALTER TABLE news        DROP COLUMN IF EXISTS updated_at;
ALTER TABLE roles       DROP COLUMN IF EXISTS updated_at;
ALTER TABLE members     DROP COLUMN IF EXISTS updated_at;
ALTER TABLE events      DROP COLUMN IF EXISTS updated_at;
