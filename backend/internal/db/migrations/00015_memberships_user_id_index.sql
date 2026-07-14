-- +goose NO TRANSACTION

-- +goose Up

-- memberships only has a UNIQUE(team_id, user_id) composite index, with
-- team_id leading -- not usable for an index scan keyed on user_id alone.
-- teams.Repository.ListTeamsForUser ("GET /teams", hit on essentially every
-- session) and auth.Repository.EraseUser's per-team advisory-lock step both
-- filter memberships purely by user_id, so both degrade to a sequential scan
-- of the whole table as total membership rows grow.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_memberships_user_id ON memberships (user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_memberships_user_id;
