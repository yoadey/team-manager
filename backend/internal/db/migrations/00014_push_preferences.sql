-- +goose Up

-- Per-(team, user) opt-out of Web Push delivery for one or more notification
-- categories. A missing row means every category is enabled -- see
-- push.DefaultCategoryPreferences -- so existing subscribers (all of whom
-- predate this table) keep receiving push exactly as before until they
-- explicitly change a preference. The primary key already serves the only
-- lookup pattern (WHERE team_id = $1 AND user_id = $2), so no secondary
-- index is needed.
CREATE TABLE push_preferences (
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attendance BOOLEAN     NOT NULL DEFAULT true,
    events     BOOLEAN     NOT NULL DEFAULT true,
    news       BOOLEAN     NOT NULL DEFAULT true,
    polls      BOOLEAN     NOT NULL DEFAULT true,
    absence    BOOLEAN     NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);

-- +goose Down

DROP TABLE IF EXISTS push_preferences;
