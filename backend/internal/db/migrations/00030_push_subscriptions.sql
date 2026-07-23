-- +goose Up

-- One row per browser/device a user has opted in to Web Push from -- keyed
-- on user_id alone (not team-scoped), since a user with several teams
-- expects a single "enable push" toggle to cover all of them. endpoint is
-- unique so re-subscribing the same browser (e.g. after a key rotation)
-- upserts in place instead of accumulating duplicate rows.
CREATE TABLE push_subscriptions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint      TEXT        NOT NULL UNIQUE,
    p256dh        TEXT        NOT NULL,
    auth_key      TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at  TIMESTAMPTZ
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions (user_id);

-- +goose Down

DROP TABLE IF EXISTS push_subscriptions;
