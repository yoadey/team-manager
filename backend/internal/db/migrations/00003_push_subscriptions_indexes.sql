-- +goose NO TRANSACTION

-- +goose Up

-- NotificationWorker.enqueuePushDeliveries and the push-subscription CRUD
-- handlers both look up by user_id; the UNIQUE(endpoint) constraint on
-- push_subscriptions already covers endpoint lookups.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_push_subscriptions_user_id ON push_subscriptions (user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_push_subscriptions_user_id;
