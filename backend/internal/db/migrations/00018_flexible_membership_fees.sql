-- +goose Up

-- Membership fees are no longer a fixed "one row per member per calendar
-- month" concept (see openspec/changes/flexible-membership-fees): the
-- treasurer now defines each fee individually with a free-text name and an
-- optional due date, and creates a new one by hand for every recurring
-- period instead of the app auto-generating one per month. `label` becomes
-- the fee's name; existing rows with no label fall back to the generic name
-- they were always displayed under in the seed/demo data. Requiredness is
-- enforced at the API layer, not with a DB-level NOT NULL -- see
-- design.md's "name stays nullable at the DB level" decision.
ALTER TABLE contributions RENAME COLUMN label TO name;
UPDATE contributions SET name = 'Mitgliedsbeitrag' WHERE name IS NULL;

ALTER TABLE contributions ADD COLUMN due_date DATE;

-- Dropping `month` also drops the UNIQUE(team_id, user_id, month)
-- constraint and idx_contributions_team_month index that were defined on it.
ALTER TABLE contributions DROP COLUMN month;

-- `status` is no longer stored: whether a fee is open/partially paid/fully
-- paid is now derived from the sum of `transactions` rows linked to it via
-- the new transactions.contribution_id below, so the two can never drift
-- apart the way a manually-toggled boolean and a separately booked income
-- transaction could. Dropping the column also drops its
-- CHECK(status IN ('open','paid')) constraint.
ALTER TABLE contributions DROP COLUMN status;

-- Links an income transaction to the fee it pays, in full or in part; a
-- fee's paid amount is SUM(transactions.amount) over its linked rows (see
-- finances.Repository.ListContributions). ON DELETE SET NULL (not CASCADE):
-- deleting a contribution must never delete income that was genuinely
-- received, only the courtesy link describing which fee it paid.
ALTER TABLE transactions ADD COLUMN contribution_id UUID REFERENCES contributions(id) ON DELETE SET NULL;

-- +goose Down

-- Irreversible data loss on Up (dropped month/status values and any
-- transaction<->contribution links), same as any DROP COLUMN -- see
-- 00012_remove_event_rsvp_deadline.sql.
ALTER TABLE transactions DROP COLUMN IF EXISTS contribution_id;
ALTER TABLE contributions ADD COLUMN status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'paid'));
ALTER TABLE contributions ADD COLUMN month CHAR(7);
ALTER TABLE contributions DROP COLUMN IF EXISTS due_date;
ALTER TABLE contributions RENAME COLUMN name TO label;
