-- +goose NO TRANSACTION

-- +goose Up

-- statsprefs.Repository.ListPresets looks up by (team_id, user_id).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_stats_view_presets_team_user ON stats_view_presets (team_id, user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_stats_view_presets_team_user;
