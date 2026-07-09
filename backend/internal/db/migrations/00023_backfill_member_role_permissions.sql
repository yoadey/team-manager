-- +goose Up

-- Round-44 (commit 80734aa) fixed CreateTeam's seeded "Member" role from
-- {events:read, members:none, finances:none, news:read, polls:read,
-- settings:none} to {..., members:read, ..., settings:read} -- a module set
-- to "none" hides GET access too, not just writes (see authz.go), so the old
-- default 403'd every ordinary member's own dashboard load (AppContext's
-- afterLoginLoad unconditionally fetches the member roster and role catalog
-- for every team member, not just admins). That fix only changed the Go
-- default applied at CreateTeam time -- it never touched already-persisted
-- "Member" role rows, and AcceptInvite assigns whatever Member role row
-- already exists for a team rather than re-deriving it. So every team
-- created before this fix shipped, and every member who joined (or still
-- joins) one of those teams via an old invite link, remains locked out
-- indefinitely with no in-app path to self-service repair -- if the
-- original creator already left, no admin capable of fixing the role via
-- the UI may even remain.
--
-- Backfill only rows that still exactly match the old broken default, so a
-- team where an admin has since deliberately re-locked members/settings
-- back down to "none" is left untouched.
UPDATE roles
SET permissions = jsonb_set(jsonb_set(permissions, '{members}', '"read"'), '{settings}', '"read"')
WHERE system = true
  AND name = 'Member'
  AND permissions = '{"events":"read","members":"none","finances":"none","news":"read","polls":"read","settings":"none"}'::jsonb;

-- +goose Down

-- Not reversible: rows this backfill touched are indistinguishable from
-- rows CreateTeam has seeded with the new default since round 44, so
-- reverting would also break brand-new teams created after this migration
-- ran. Deliberately a no-op, matching 00018's precedent for a
-- data-only migration.
