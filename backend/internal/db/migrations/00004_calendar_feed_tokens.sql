-- +goose Up

-- One active token per (user, team): issuing a new one revokes the previous
-- row rather than deleting it, so history/debugging can still see when a
-- link was rotated. token is stored in the clear (unlike session tokens,
-- which are hashed) -- mirroring invites.code, a leaked feed link is meant
-- to be trivially revocable and replaceable, not something the DB needs to
-- protect against its own compromise the way a password or session token
-- does.
CREATE TABLE calendar_feed_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id     UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    token       TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ
);

-- The feed handler looks up by bare token on every request; the UNIQUE(token)
-- constraint above already provides a btree index for this lookup. The
-- "one active token per user+team" index is added in a follow-up migration,
-- since building it without blocking writes needs a mode of index creation
-- that cannot run inside this migration's own transaction.

-- +goose Down

DROP TABLE IF EXISTS calendar_feed_tokens;
