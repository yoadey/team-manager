package roles_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/yoadey/team-manager/backend/internal/roles"
	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// seedRoleAdminCaller creates a user with a full-permission role in tid,
// suitable as UpdateRole's callerUserID in tests that aren't specifically
// exercising the permission-escalation ceiling (enforceNoRoleEscalation).
func seedRoleAdminCaller(t *testing.T, pool *pgxpool.Pool, tid string) string {
	t.Helper()
	ctx := context.Background()
	uid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Role Admin Caller', $2, '#abcdef')`,
		uid, uid+"@example.com")
	require.NoError(t, err)
	var mid string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid).Scan(&mid))
	adminRole, err := roles.NewRepository(pool).CreateRole(ctx, tid, "Caller Admin Role", nil, teams.PermissionsJSON{
		Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write",
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid, adminRole.Id.String())
	require.NoError(t, err)
	return uid
}

func TestRolesRepository_CreateAndList(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Roles Team')`, tid)
	require.NoError(t, err)

	perms := teams.PermissionsJSON{
		Events: "write", Members: "read", Finances: "none",
		News: "read", Polls: "write", Settings: "none",
	}
	color := "#ff6600"
	role, err := repo.CreateRole(ctx, tid, "Coach", &color, perms)
	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Equal(t, "Coach", role.Name)
	assert.Equal(t, &color, role.Color)
	assert.Equal(t, "write", role.Permissions.Events)
	assert.False(t, role.System)

	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, role.Id, list[0].Id)
}

func TestRolesRepository_Update(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Update Roles Team')`, tid)
	require.NoError(t, err)

	role, err := repo.CreateRole(ctx, tid, "Player", nil, teams.PermissionsJSON{})
	require.NoError(t, err)
	callerUserID := seedRoleAdminCaller(t, pool, tid)

	newName := "Captain"
	newColor := "#0000ff"
	newPerms := teams.PermissionsJSON{Events: "write", Members: "write"}
	updated, err := repo.UpdateRole(ctx, role.Id.String(), tid, callerUserID, roles.RolePatch{
		Name: &newName, Color: &newColor, Permissions: &newPerms,
	})
	require.NoError(t, err)
	assert.Equal(t, "Captain", updated.Name)
	assert.Equal(t, &newColor, updated.Color)
	assert.Equal(t, "write", updated.Permissions.Events)
}

func TestRolesRepository_Update_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	otherTid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Owning Roles Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Attacker Roles Team')`, otherTid)
	require.NoError(t, err)

	role, err := roles.NewRepository(pool).CreateRole(ctx, tid, "Player", nil, teams.PermissionsJSON{})
	require.NoError(t, err)

	newName := "Attacker Renamed"
	_, err = repo.UpdateRole(ctx, role.Id.String(), otherTid, uuid.New().String(), roles.RolePatch{Name: &newName})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestRolesRepository_Delete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Roles Team')`, tid)
	require.NoError(t, err)

	role, err := repo.CreateRole(ctx, tid, "Temp Role", nil, teams.PermissionsJSON{})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteRole(ctx, role.Id.String(), tid))

	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// Deleting the team's only settings:write-granting role must be blocked —
// membership_roles cascades on role deletion, so this would otherwise
// silently strip every member holding it of admin access in one step, the
// same unrecoverable lockout members.SetRoles/RemoveMember already guard
// against.
func TestRolesRepository_DeleteRole_LastSettingsAdmin_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	uid := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Last Admin Role Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Sole Admin', 'sole-role-admin@example.com', '#123456')`, uid)
	require.NoError(t, err)
	var mid string
	err = pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid).Scan(&mid)
	require.NoError(t, err)

	adminRole, err := repo.CreateRole(ctx, tid, "Custom Admin", nil, teams.PermissionsJSON{
		Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write",
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid, adminRole.Id.String())
	require.NoError(t, err)

	err = repo.DeleteRole(ctx, adminRole.Id.String(), tid)
	require.ErrorIs(t, err, roles.ErrLastSettingsAdmin)

	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	assert.Len(t, list, 1, "role must still exist")
}

// A settings:write role can be deleted once another role/member already
// provides equivalent coverage.
func TestRolesRepository_DeleteRole_NotLastSettingsAdmin_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	uid1 := uuid.New().String()
	uid2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Two Admin Roles Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Admin One', 'role-admin1@example.com', '#111111')`, uid1)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Admin Two', 'role-admin2@example.com', '#222222')`, uid2)
	require.NoError(t, err)
	var mid1, mid2 string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid1).Scan(&mid1))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid2).Scan(&mid2))

	adminPerms := teams.PermissionsJSON{Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write"}
	roleA, err := repo.CreateRole(ctx, tid, "Admin A", nil, adminPerms)
	require.NoError(t, err)
	roleB, err := repo.CreateRole(ctx, tid, "Admin B", nil, adminPerms)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid1, roleA.Id.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid2, roleB.Id.String())
	require.NoError(t, err)

	require.NoError(t, repo.DeleteRole(ctx, roleA.Id.String(), tid))

	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, roleB.Id, list[0].Id)
}

// Regression test: reason_visibility_role_ids (a plain UUID[] column on
// teams, set via teams.Repository.UpdateTeam) has no FK to roles, so
// deleting a role that a team references there used to leave a permanently
// dangling ID with no error or indication. DeleteRole must scrub it.
func TestRolesRepository_DeleteRole_ScrubsReasonVisibilityRoleIDs(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Reason Vis Cleanup Team')`, tid)
	require.NoError(t, err)

	trainerRole, err := repo.CreateRole(ctx, tid, "Trainer", nil, teams.PermissionsJSON{
		Events: "read", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none",
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`UPDATE teams SET reason_visibility_role_ids = $1 WHERE id = $2`,
		[]string{trainerRole.Id.String()}, tid,
	)
	require.NoError(t, err)

	require.NoError(t, repo.DeleteRole(ctx, trainerRole.Id.String(), tid))

	var remaining []string
	err = pool.QueryRow(ctx, `SELECT reason_visibility_role_ids FROM teams WHERE id = $1`, tid).Scan(&remaining)
	require.NoError(t, err)
	assert.Empty(t, remaining, "deleted role's ID must be scrubbed from reason_visibility_role_ids")
}

// Regression test: events.nominated_role_ids and event_series.nominated_role_ids
// are the same kind of plain UUID[] column with no FK to roles -- deleting a
// role referenced there used to leave a permanently dangling ID in every
// future ListEvents/GetEvent response for that event/series.
func TestRolesRepository_DeleteRole_ScrubsNominatedRoleIDs(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Nominated Roles Cleanup Team')`, tid)
	require.NoError(t, err)

	trainerRole, err := repo.CreateRole(ctx, tid, "Trainer", nil, teams.PermissionsJSON{
		Events: "read", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "none",
	})
	require.NoError(t, err)

	var seriesID string
	err = pool.QueryRow(ctx, `
		INSERT INTO event_series (team_id, type, title, nominated_role_ids)
		VALUES ($1, 'training', 'Weekly Training', $2)
		RETURNING id`,
		tid, []string{trainerRole.Id.String()},
	).Scan(&seriesID)
	require.NoError(t, err)

	var eventID string
	err = pool.QueryRow(ctx, `
		INSERT INTO events (team_id, series_id, type, title, date, nominated_role_ids)
		VALUES ($1, $2, 'training', 'Training', CURRENT_DATE, $3)
		RETURNING id`,
		tid, seriesID, []string{trainerRole.Id.String()},
	).Scan(&eventID)
	require.NoError(t, err)

	require.NoError(t, repo.DeleteRole(ctx, trainerRole.Id.String(), tid))

	var eventRemaining, seriesRemaining []string
	require.NoError(t, pool.QueryRow(ctx, `SELECT nominated_role_ids FROM events WHERE id = $1`, eventID).Scan(&eventRemaining))
	require.NoError(t, pool.QueryRow(ctx, `SELECT nominated_role_ids FROM event_series WHERE id = $1`, seriesID).Scan(&seriesRemaining))
	assert.Empty(t, eventRemaining, "deleted role's ID must be scrubbed from events.nominated_role_ids")
	assert.Empty(t, seriesRemaining, "deleted role's ID must be scrubbed from event_series.nominated_role_ids")
}

// Regression test: UpdateRole previously had no last-settings-admin guard at
// all, so revoking settings:write via an edit (instead of deleting the role
// outright) could lock a team out of settings/role management — exactly the
// state DeleteRole's ErrLastSettingsAdmin guard exists to prevent.
func TestRolesRepository_UpdateRole_RevokingLastSettingsWrite_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	uid := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Last Admin Update Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Sole Admin', 'sole-update-admin@example.com', '#654321')`, uid)
	require.NoError(t, err)
	var mid string
	err = pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid).Scan(&mid)
	require.NoError(t, err)

	adminRole, err := repo.CreateRole(ctx, tid, "Custom Admin", nil, teams.PermissionsJSON{
		Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write",
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid, adminRole.Id.String())
	require.NoError(t, err)

	downgraded := teams.PermissionsJSON{
		Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "read",
	}
	_, err = repo.UpdateRole(ctx, adminRole.Id.String(), tid, uid, roles.RolePatch{Permissions: &downgraded})
	require.ErrorIs(t, err, roles.ErrLastSettingsAdmin)

	// The role's permissions must be untouched by the rejected update.
	got, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "write", got[0].Permissions.Settings)
}

// A settings:write role's permissions can be downgraded once another
// role/member already provides equivalent coverage.
func TestRolesRepository_UpdateRole_NotLastSettingsAdmin_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	uid1 := uuid.New().String()
	uid2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Two Admin Roles Update Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Admin One', 'role-update-admin1@example.com', '#111112')`, uid1)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Admin Two', 'role-update-admin2@example.com', '#222223')`, uid2)
	require.NoError(t, err)
	var mid1, mid2 string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid1).Scan(&mid1))
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, uid2).Scan(&mid2))

	adminPerms := teams.PermissionsJSON{Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write"}
	roleA, err := repo.CreateRole(ctx, tid, "Admin A", nil, adminPerms)
	require.NoError(t, err)
	roleB, err := repo.CreateRole(ctx, tid, "Admin B", nil, adminPerms)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid1, roleA.Id.String())
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, mid2, roleB.Id.String())
	require.NoError(t, err)

	downgraded := teams.PermissionsJSON{Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "read"}
	updated, err := repo.UpdateRole(ctx, roleA.Id.String(), tid, uid1, roles.RolePatch{Permissions: &downgraded})
	require.NoError(t, err)
	assert.Equal(t, "read", updated.Permissions.Settings)
}

func TestRolesRepository_Delete_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	otherTid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Owning Roles Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Attacker Roles Team')`, otherTid)
	require.NoError(t, err)

	role, err := repo.CreateRole(ctx, tid, "Temp Role", nil, teams.PermissionsJSON{})
	require.NoError(t, err)

	err = repo.DeleteRole(ctx, role.Id.String(), otherTid)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestRolesRepository_DeleteSystemRole_Fails(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Sys Role Team')`, tid)
	require.NoError(t, err)

	var roleID string
	err = pool.QueryRow(
		ctx,
		`INSERT INTO roles (team_id, name, system) VALUES ($1, 'Admin', true) RETURNING id`, tid,
	).Scan(&roleID)
	require.NoError(t, err)

	err = repo.DeleteRole(ctx, roleID, tid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "system")
}

func TestRolesRepository_UpdateSystemRole_NameOrPermissions_Fails(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Sys Role Update Team')`, tid)
	require.NoError(t, err)

	var roleID string
	err = pool.QueryRow(
		ctx,
		`INSERT INTO roles (team_id, name, system, permissions) VALUES ($1, 'Admin', true, '{}') RETURNING id`, tid,
	).Scan(&roleID)
	require.NoError(t, err)

	// A settings:write holder must not be able to rename a system role...
	newName := "Renamed Admin"
	_, err = repo.UpdateRole(ctx, roleID, tid, uuid.New().String(), roles.RolePatch{Name: &newName})
	require.ErrorIs(t, err, roles.ErrSystemRole)

	// ...nor rewrite its permissions (the actual privilege-escalation vector).
	newPerms := teams.PermissionsJSON{Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write"}
	_, err = repo.UpdateRole(ctx, roleID, tid, uuid.New().String(), roles.RolePatch{Permissions: &newPerms})
	require.ErrorIs(t, err, roles.ErrSystemRole)

	// Verify nothing was actually changed by the rejected attempts.
	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Admin", list[0].Name)
	assert.Empty(t, list[0].Permissions.Events)
}

func TestRolesRepository_UpdateSystemRole_ColorOnly_Succeeds(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Sys Role Color Team')`, tid)
	require.NoError(t, err)

	var roleID string
	err = pool.QueryRow(
		ctx,
		`INSERT INTO roles (team_id, name, system, permissions) VALUES ($1, 'Admin', true, '{}') RETURNING id`, tid,
	).Scan(&roleID)
	require.NoError(t, err)

	// Color is cosmetic-only and must remain editable even for system roles.
	newColor := "#123456"
	updated, err := repo.UpdateRole(ctx, roleID, tid, uuid.New().String(), roles.RolePatch{Color: &newColor})
	require.NoError(t, err)
	assert.Equal(t, &newColor, updated.Color)
	assert.Equal(t, "Admin", updated.Name)
}

// Regression test for the privilege-escalation path this fix closes:
// UpdateRole previously had no ceiling check at all, so a settings:write-only
// caller (able to reach both POST/PATCH .../roles under RequirePermission)
// could create a role granting only what they already hold, assign it to
// themselves -- passing SetRoles's ceiling trivially, since it grants
// nothing beyond what they have -- and then PATCH that same role's
// permissions upward afterward, escalating their own effective permissions
// with no assignment step left to catch it.
func TestRolesRepository_UpdateRole_EscalationBeyondCallersOwnPermissions_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	callerUID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Escalation Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Settings Only Caller', 'settings-only@example.com', '#333333')`, callerUID)
	require.NoError(t, err)
	var callerMid string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, callerUID).Scan(&callerMid))

	// Caller holds only settings:write, granted via a role assigned to them.
	settingsOnlyPerms := teams.PermissionsJSON{Events: "none", Members: "none", Finances: "none", News: "none", Polls: "none", Settings: "write"}
	settingsOnlyRole, err := repo.CreateRole(ctx, tid, "Settings Only", nil, settingsOnlyPerms)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, callerMid, settingsOnlyRole.Id.String())
	require.NoError(t, err)

	// The role the caller wants to patch upward: currently grants nothing
	// beyond what the caller already holds (settings:write), so ceiling (b)
	// doesn't cover the escalation attempt either.
	targetRole, err := repo.CreateRole(ctx, tid, "Target", nil, settingsOnlyPerms)
	require.NoError(t, err)

	escalated := teams.PermissionsJSON{Finances: "write", Settings: "write"}
	_, err = repo.UpdateRole(ctx, targetRole.Id.String(), tid, callerUID, roles.RolePatch{Permissions: &escalated})
	require.ErrorIs(t, err, roles.ErrInsufficientPermissionToGrant)

	// The role's permissions must be untouched by the rejected update.
	got, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	for _, rr := range got {
		if rr.Id == targetRole.Id {
			assert.Equal(t, "none", rr.Permissions.Finances)
		}
	}
}

// A caller may grant a role a permission level they hold themselves.
func TestRolesRepository_UpdateRole_WithinCallersOwnPermissions_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Within Ceiling Team')`, tid)
	require.NoError(t, err)
	callerUID := seedRoleAdminCaller(t, pool, tid)

	targetRole, err := repo.CreateRole(ctx, tid, "Target", nil, teams.PermissionsJSON{})
	require.NoError(t, err)

	newPerms := teams.PermissionsJSON{Finances: "write"}
	updated, err := repo.UpdateRole(ctx, targetRole.Id.String(), tid, callerUID, roles.RolePatch{Permissions: &newPerms})
	require.NoError(t, err)
	assert.Equal(t, "write", updated.Permissions.Finances)
}

// A caller with only settings:write may still reorganize/demote a role that
// ALREADY grants more than they hold -- ceiling (b) exists precisely so a
// settings:write-only caller isn't locked out of managing role assignments
// for permissions someone else legitimately granted, as long as the result
// doesn't exceed what the role already granted.
func TestRolesRepository_UpdateRole_DemotingRoleAboveCallersOwnPermissions_Allowed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := roles.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	callerUID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Demote Above Ceiling Team')`, tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Settings Only Caller 2', 'settings-only-2@example.com', '#444444')`, callerUID)
	require.NoError(t, err)
	var callerMid string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, tid, callerUID).Scan(&callerMid))
	settingsOnlyRole, err := repo.CreateRole(ctx, tid, "Settings Only", nil, teams.PermissionsJSON{Settings: "write"})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, callerMid, settingsOnlyRole.Id.String())
	require.NoError(t, err)

	// A role that already grants finances:write -- legitimately created by
	// someone else with that permission -- before this caller touches it.
	financeRole, err := repo.CreateRole(ctx, tid, "Finance Role", nil, teams.PermissionsJSON{Finances: "write", Settings: "write"})
	require.NoError(t, err)

	// The caller renames it and demotes finances from write to read -- never
	// exceeding what the role already granted, so this must be allowed even
	// though the caller has no finances permission themselves.
	newName := "Finance Role (Read Only)"
	demoted := teams.PermissionsJSON{Finances: "read", Settings: "write"}
	updated, err := repo.UpdateRole(ctx, financeRole.Id.String(), tid, callerUID, roles.RolePatch{Name: &newName, Permissions: &demoted})
	require.NoError(t, err)
	assert.Equal(t, "Finance Role (Read Only)", updated.Name)
	assert.Equal(t, "read", updated.Permissions.Finances)
}
