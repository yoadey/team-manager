-- +goose NO TRANSACTION

-- +goose Up

-- ListByTeam filters on team_id and keyset-orders on (from_date DESC, id
-- DESC), but the only index touching those columns is idx_absences_team_dates
-- (team_id, from_date, to_date) -- no id column, and to_date is never used in
-- any WHERE/ORDER BY anywhere in the codebase (select-list/CHECK-constraint
-- only), so it contributes nothing a plain (team_id, from_date) prefix
-- wouldn't already give. Migration 00020 fixed the equivalent gap for
-- ListByUser specifically; this closes it for the team-wide list too, and
-- drops the now-fully-redundant old index rather than keeping it alongside.
DROP INDEX IF EXISTS idx_absences_team_dates;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_absences_team_date_id
    ON absences (team_id, from_date DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_absences_team_date_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_absences_team_dates ON absences (team_id, from_date, to_date);
