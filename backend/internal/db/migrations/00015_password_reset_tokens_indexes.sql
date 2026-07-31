-- +goose NO TRANSACTION

-- +goose Up

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_password_reset_tokens_user_id    ON password_reset_tokens (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens (expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;
DROP INDEX IF EXISTS idx_password_reset_tokens_expires_at;
