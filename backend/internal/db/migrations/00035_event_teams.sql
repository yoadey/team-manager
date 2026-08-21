-- +goose Up

-- One row per (event_id, team_id) an event targets, including its own
-- "owning" team (events.team_id is retained for back-compat -- every read
-- path that scoped on events.team_id = $N is relaxed to an EXISTS check
-- against this table instead; see internal/events/repository.go). A
-- single-team event simply has exactly one row here, matching its own
-- events.team_id -- every EXISTS-join relaxation is then a strict superset
-- of the old team_id = $N check for that case, so single-team behavior is
-- unchanged. Creating/editing the target set requires events:write in every
-- targeted team (internal/events/service.go); reading/RSVPing an event only
-- requires membership in any one of them. Deleting an event stays restricted
-- to the owning team only (events.team_id), deliberately not extended to
-- this table, to avoid ambiguity over which target team may delete a shared
-- event.
CREATE TABLE event_teams (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id  UUID NOT NULL REFERENCES teams(id)  ON DELETE CASCADE,
    PRIMARY KEY (event_id, team_id)
);

-- Every existing event predates this table and has no row here yet. Every
-- relaxed read/RSVP path (GetEvent, ListEvents, ListAttendance, comments,
-- SetAttendance/SetNomination, SetStatus, GetMyEffectiveAttendance/-s) now
-- scopes via `EXISTS (SELECT 1 FROM event_teams ...)` instead of
-- `team_id = $N` -- without this backfill that EXISTS is false for every
-- pre-existing event, hiding it from its own owning team the moment this
-- ships (same class of bug 00032_stats_rbac_module.sql's backfill avoids
-- for the "stats" RBAC module).
INSERT INTO event_teams (event_id, team_id)
SELECT id, team_id FROM events
ON CONFLICT DO NOTHING;

-- The reverse lookup direction ("which cross-team events target team X",
-- used by ListEvents/ListAttendance/etc. to resolve visibility for a
-- non-owning viewer) isn't served by the PK's leading column -- see the
-- follow-up indexes migration for that index, built without holding a
-- table-level lock for the duration.

-- +goose Down

DROP TABLE IF EXISTS event_teams;
