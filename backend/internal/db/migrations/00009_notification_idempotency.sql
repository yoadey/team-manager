-- +goose Up

-- River guarantees at-least-once delivery, not exactly-once: if the process
-- crashes after the INSERT below commits but before River records the job as
-- complete, the job is retried and would otherwise insert a duplicate
-- notification row. river_job_id lets the worker use ON CONFLICT DO NOTHING
-- to make the insert actually idempotent, matching the guarantee the worker's
-- doc comment already (incorrectly) claimed.
ALTER TABLE notifications ADD COLUMN river_job_id BIGINT;
CREATE UNIQUE INDEX notifications_river_job_id_idx ON notifications (river_job_id) WHERE river_job_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS notifications_river_job_id_idx;
ALTER TABLE notifications DROP COLUMN IF EXISTS river_job_id;
