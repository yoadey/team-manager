-- +goose Up

-- Event-reminder push settings, per (team, user) -- lives on the same
-- push_preferences row as the existing category toggles rather than a
-- separate table, since it's saved through the same whole-object PUT
-- /teams/{teamId}/push-preferences endpoint. A missing row still means
-- "enabled, 6 hours" via push.DefaultCategoryPreferences, same as every
-- other column on this table defaulting to enabled.
ALTER TABLE push_preferences
    ADD COLUMN event_reminder_enabled     BOOLEAN  NOT NULL DEFAULT true,
    ADD COLUMN event_reminder_hours_before SMALLINT NOT NULL DEFAULT 6
        CHECK (event_reminder_hours_before BETWEEN 1 AND 72);

-- Idempotency marker: records that a reminder push has already been
-- enqueued for a given (event, member) pair, so jobs.EventReminderWorker's
-- periodic re-scan of upcoming events never sends the same reminder twice.
-- The primary key is the only lookup pattern this table needs (an
-- ON CONFLICT DO NOTHING upsert per candidate pair on every tick).
CREATE TABLE event_reminders_sent (
    event_id UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sent_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);

-- +goose Down

DROP TABLE IF EXISTS event_reminders_sent;

ALTER TABLE push_preferences
    DROP COLUMN IF EXISTS event_reminder_enabled,
    DROP COLUMN IF EXISTS event_reminder_hours_before;
