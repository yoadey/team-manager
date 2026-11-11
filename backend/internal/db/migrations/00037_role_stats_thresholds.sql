-- +goose Up

-- Per-group (role) attendance threshold for the statistics page's per-event
-- line-up: an incomplete group is highlighted as "too low" when its
-- attendance percentage (mode 'percent') or its number of attendees (mode
-- 'count') is below value. A role with no row here resolves to the default
-- (percent / 50, see stats.DefaultThresholdMode/Value) -- so no backfill is
-- needed and a freshly created role needs no companion insert.
--
-- Keyed by role_id alone, deliberately without a team_id column: the role
-- already pins the team, and every query reaches this table through
-- `roles r ... WHERE r.team_id = $N` (see stats.Repository), which is what
-- keeps the per-team scoping invariant. ON DELETE CASCADE removes the
-- threshold with its role, so -- unlike teams.reason_visibility_role_ids --
-- roles.DeleteRole needs no scrub step for it.
--
-- The CHECKs are a backstop for stats.Service's own validation (mode enum,
-- 0..100 for percent, 0..10000 for count), not the primary enforcement.
CREATE TABLE role_stats_thresholds (
    role_id    uuid PRIMARY KEY REFERENCES roles(id) ON DELETE CASCADE,
    mode       text NOT NULL CHECK (mode IN ('percent', 'count')),
    value      integer NOT NULL CHECK (value >= 0 AND value <= 10000),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (mode <> 'percent' OR value <= 100)
);

-- +goose Down

DROP TABLE IF EXISTS role_stats_thresholds;
