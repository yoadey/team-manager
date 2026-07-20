-- +goose NO TRANSACTION

-- +goose Up

-- Repository.FindEmailVerificationToken looks up by token_hash (already
-- indexed via the UNIQUE constraint), but Service.Register's
-- taken-unverified-pending path and the retention job's cleanup both need to
-- find/delete tokens by user_id and by expiry -- neither is covered by the
-- UNIQUE(token_hash) index alone.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email_verification_tokens_user_id ON email_verification_tokens (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email_verification_tokens_expires_at ON email_verification_tokens (expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_email_verification_tokens_expires_at;
DROP INDEX IF EXISTS idx_email_verification_tokens_user_id;
