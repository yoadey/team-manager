-- name: ListRolesByTeam :many
SELECT id, team_id, name, system, color, permissions
FROM roles
WHERE team_id = $1
ORDER BY system DESC, name;

-- name: CreateRole :one
INSERT INTO roles (team_id, name, color, permissions)
VALUES ($1, $2, $3, $4)
RETURNING id, team_id, name, system, color, permissions;

-- name: GetRoleByID :one
SELECT id, team_id, name, system, color, permissions
FROM roles WHERE id = $1 AND team_id = $2;

-- name: GetRoleSystemAndPermissions :one
SELECT system, permissions FROM roles WHERE id = $1 AND team_id = $2;

-- name: GetRoleSystem :one
SELECT system FROM roles WHERE id = $1 AND team_id = $2;

-- name: GetRoleHasSettingsWrite :one
SELECT (permissions->>'settings' = 'write')::boolean FROM roles WHERE id = $1 AND team_id = $2;

-- name: CheckOtherRolesHaveSettingsWrite :one
-- Whether any role other than $2 (roleID), assigned to a still-active member
-- (deleted_at IS NULL excludes GDPR-erased accounts, which keep their
-- membership_roles rows but can never authenticate again), grants
-- settings:write in team $1.
SELECT EXISTS (
    SELECT 1 FROM memberships m
    JOIN membership_roles mr ON mr.membership_id = m.id
    JOIN roles r ON r.id = mr.role_id
    JOIN users u ON u.id = m.user_id
    WHERE m.team_id = $1 AND r.id != $2 AND r.team_id = m.team_id
      AND r.permissions->>'settings' = 'write'
      AND u.deleted_at IS NULL
)::boolean;

-- name: DeleteRole :execrows
DELETE FROM roles WHERE id = $1 AND team_id = $2;

-- name: ScrubTeamReasonVisibilityRoleID :exec
UPDATE teams SET reason_visibility_role_ids = array_remove(reason_visibility_role_ids, sqlc.arg(role_id)::uuid) WHERE id = $1;

-- name: ScrubEventsNominatedRoleID :exec
UPDATE events SET nominated_role_ids = array_remove(nominated_role_ids, sqlc.arg(role_id)::uuid) WHERE team_id = $1;

-- name: ScrubEventSeriesNominatedRoleID :exec
UPDATE event_series SET nominated_role_ids = array_remove(nominated_role_ids, sqlc.arg(role_id)::uuid) WHERE team_id = $1;

-- name: CountRolesForTeam :one
-- Distinct-role count for a candidate ID list, used by RolesExistForTeam to
-- verify every ID in roleIDs belongs to teamID (compares against the caller's
-- own distinct-id count, since COUNT(*) here counts matching rows, not input
-- array elements).
SELECT COUNT(*)::int FROM roles WHERE team_id = $1 AND id = ANY(sqlc.arg(role_ids)::uuid[]);

-- name: GetEffectivePermissionsForUser :many
SELECT r.permissions
FROM roles r
JOIN membership_roles mr ON mr.role_id = r.id
JOIN memberships m ON m.id = mr.membership_id
WHERE m.team_id = $1 AND m.user_id = $2 AND r.team_id = $1;
