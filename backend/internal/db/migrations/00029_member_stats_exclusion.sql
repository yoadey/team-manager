-- +goose Up

-- Removes a member from personal-quota-oriented attendance statistics
-- (overview, single-member view, attendance matrix) while their historical
-- event-level responses still count toward per-event turnout aggregates
-- (see stats.Repository -- EventStats' roster join is deliberately left
-- unfiltered). Team-scoped like the existing "group" column, not on users,
-- since the same person may be excluded on one team and not another.
ALTER TABLE memberships ADD COLUMN exclude_from_stats BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE memberships DROP COLUMN IF EXISTS exclude_from_stats;
