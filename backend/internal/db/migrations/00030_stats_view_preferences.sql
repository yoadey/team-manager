-- +goose Up

-- Named, reusable date ranges a member saves for themselves (e.g. "Saison
-- 2026/27"), private to the creator -- see design.md's Non-Goals for why
-- team-shared presets are out of scope here. Created before
-- stats_last_selection so the latter's preset_id can reference it inline.
-- CHECK (to_date >= from_date) mirrors absences' identical backstop
-- (migration 00001) for the same date-ordering concern: the handler only
-- validates ordering when both bounds arrive in the same request (see
-- statsprefs/handler.go), so a PATCH patching just one bound needs this
-- constraint to still reject an inverted range against the stored other
-- bound.
CREATE TABLE stats_view_presets (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       text NOT NULL,
    from_date  date NOT NULL,
    to_date    date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (to_date >= from_date)
);

-- Stores each member's last-selected statistics date range per team, so the
-- Stats page can restore it on the next visit instead of always resetting to
-- the default last-3-months window. One row per (team, user); upserted on
-- every selection change. preset_id is nullable and ON DELETE SET NULL:
-- deleting a preset degrades a saved selection back to its raw from/to dates
-- instead of failing.
-- CHECK (to_date >= from_date) is satisfied whenever either bound is NULL
-- (Postgres treats a NULL comparison as not violating a CHECK), so this
-- doesn't interfere with "nothing saved yet" (both NULL); it only rejects
-- an inverted range once both bounds are actually set.
CREATE TABLE stats_last_selection (
    team_id    uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_date  date,
    to_date    date,
    preset_id  uuid REFERENCES stats_view_presets(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id),
    CHECK (to_date >= from_date)
);

-- +goose Down

DROP TABLE IF EXISTS stats_last_selection;
DROP TABLE IF EXISTS stats_view_presets;
