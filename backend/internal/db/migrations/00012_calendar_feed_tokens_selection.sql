-- +goose Up

-- Per-token content selection: which event types the feed renders and
-- whether it includes member birthdays. Stored on the token row itself (not
-- a separate table) so it stays tied to a single subscription URL -- rotate
-- carries the previous selection forward, edit doesn't require re-issuing.
ALTER TABLE calendar_feed_tokens
    ADD COLUMN types             TEXT[]  NOT NULL DEFAULT '{training,auftritt,event}',
    ADD COLUMN include_birthdays BOOLEAN NOT NULL DEFAULT true;

-- +goose Down

ALTER TABLE calendar_feed_tokens
    DROP COLUMN types,
    DROP COLUMN include_birthdays;
