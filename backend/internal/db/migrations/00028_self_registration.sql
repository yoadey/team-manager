-- +goose Up

-- Self-service registration (email + password) needs an explicit verified
-- flag: every account today is provisioned out-of-band (manual DB insert),
-- so nothing pre-existing should retroactively become "unverified" and risk
-- being rejected at login or swept up by the retention job this change adds.
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

-- Verification tokens are stored hashed (SHA-256 hex), mirroring
-- sessions.token_hash -- the raw token is only ever held in memory and in the
-- emailed link, never persisted. consumed_at makes a token single-use.
CREATE TABLE email_verification_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        UNIQUE NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS email_verification_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
