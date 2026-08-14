-- +goose Up

-- Introduces the "stats" RBAC module. Every existing role row predates it
-- and is missing the "stats" key in its permissions JSONB; without a
-- backfill, teams.PermissionsJSON.Stats would decode to "" for every role,
-- which authz.go/roles' permission checks treat identically to "none" --
-- silently hiding the entire statistics area from every existing team,
-- including their own Admins, the moment this ships. Backfill explicitly
-- instead of relying on that implicit fallback, so every role row stays
-- self-describing for all modules.

-- System Admin roles keep full access, matching every other module they
-- already hold "write" on.
UPDATE roles SET permissions = permissions || '{"stats":"write"}'::jsonb
WHERE system = true AND name = 'Admin' AND NOT (permissions ? 'stats');

-- System Member roles default to "read", matching events/members/news/
-- polls/settings -- the default a member gets to see attendance
-- statistics without any role reconfiguration.
UPDATE roles SET permissions = permissions || '{"stats":"read"}'::jsonb
WHERE system = true AND name = 'Member' AND NOT (permissions ? 'stats');

-- Every other existing role (custom, or any system role not matched
-- above) defaults to "none" -- explicit rather than left as a missing
-- key, adjustable afterwards via the role editor like any other module.
-- Runs last so it only catches what the two updates above didn't.
UPDATE roles SET permissions = permissions || '{"stats":"none"}'::jsonb
WHERE NOT (permissions ? 'stats');

-- +goose Down

UPDATE roles SET permissions = permissions - 'stats';
