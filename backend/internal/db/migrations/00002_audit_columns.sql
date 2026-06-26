-- +goose Up
-- Add updated_at audit columns to core mutable tables.
-- created_at already exists; updated_at tracks the last write for compliance logging.

ALTER TABLE events
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE memberships
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE roles
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE news
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE polls
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE transactions
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE penalties
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE contributions
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE absences
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE absences      DROP COLUMN IF EXISTS updated_at;
ALTER TABLE contributions DROP COLUMN IF EXISTS updated_at;
ALTER TABLE penalties     DROP COLUMN IF EXISTS updated_at;
ALTER TABLE transactions   DROP COLUMN IF EXISTS updated_at;
ALTER TABLE polls         DROP COLUMN IF EXISTS updated_at;
ALTER TABLE news          DROP COLUMN IF EXISTS updated_at;
ALTER TABLE roles         DROP COLUMN IF EXISTS updated_at;
ALTER TABLE memberships   DROP COLUMN IF EXISTS updated_at;
ALTER TABLE events        DROP COLUMN IF EXISTS updated_at;
