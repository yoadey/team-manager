-- +goose Up

-- Password reset tokens: structurally mirrors email_verification_tokens
-- (hashed token at rest, single-use via consumed_at, TTL via expires_at),
-- but kept as its own table rather than a shared one with a "purpose"
-- column so the two token kinds can never be cross-redeemed by construction
-- -- see openspec/changes/password-reset/design.md.
CREATE TABLE password_reset_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        UNIQUE NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE IF EXISTS password_reset_tokens;
