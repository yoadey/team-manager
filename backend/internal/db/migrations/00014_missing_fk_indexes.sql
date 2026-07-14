-- +goose NO TRANSACTION

-- +goose Up

-- membership_roles.role_id has no supporting index -- it's only the
-- trailing column of the composite PK (membership_id, role_id), which can't
-- be used for an index scan keyed on role_id alone. Every role deletion
-- (roles.Repository.DeleteRole's "DELETE FROM roles WHERE id = $1 ...")
-- forces a sequential scan of the entire membership_roles table to find its
-- ON DELETE CASCADE rows, growing worse as teams accumulate roles/members.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_membership_roles_role_id ON membership_roles (role_id);

-- event_comments has no index on event_id at all, unlike attendance (which
-- has both event_id and user_id indexes from migration 00006). ListComments
-- and DeleteComment both filter on event_id, and it's the ON DELETE CASCADE
-- FK target when an event is deleted -- this is a hot path (every event
-- detail page view) that degrades to a full table scan as comment volume
-- grows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_comments_event_id ON event_comments (event_id);

-- invites had no index at all beyond the implicit unique index on `code`
-- and the PK on `id`. jobs/retention.go now deletes rows past
-- expires_at + a grace period (the table previously grew unboundedly, since
-- CreateInvite is called every time the invite sheet is opened with no
-- reuse of unexpired codes) -- that DELETE's WHERE clause needs this index
-- for the same reason sessions.expires_at has one.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_invites_expires_at ON invites (expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_membership_roles_role_id;
DROP INDEX IF EXISTS idx_event_comments_event_id;
DROP INDEX IF EXISTS idx_invites_expires_at;
