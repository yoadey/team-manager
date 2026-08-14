-- +goose Up

-- A short, self-service, purely cosmetic label a member can give
-- themselves (e.g. "Witzbeauftragter") -- display-only, never interpreted
-- by RBAC. Nullable TEXT, no default, same shape as the existing
-- memberships."group" column.
ALTER TABLE memberships ADD COLUMN title TEXT;

-- +goose Down

ALTER TABLE memberships DROP COLUMN IF EXISTS title;
