-- +goose NO TRANSACTION

-- +goose Up

-- jobs.deleteUnverifiedUsers scans WHERE email_verified_at IS NULL AND
-- created_at < $1 daily inside retentionPhaseTimeout; this partial index
-- covers that exact predicate so the scan stays index-driven as the users
-- table grows, instead of a full sequential scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_unverified_created_at ON users (created_at) WHERE email_verified_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_users_unverified_created_at;
