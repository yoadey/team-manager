package roles_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/yoadey/team-manager/backend/internal/roles"
	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

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

	newName := "Captain"
	newColor := "#0000ff"
	newPerms := teams.PermissionsJSON{Events: "write", Members: "write"}
	updated, err := repo.UpdateRole(ctx, role.Id.String(), tid, roles.RolePatch{
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
	_, err = repo.UpdateRole(ctx, role.Id.String(), otherTid, roles.RolePatch{Name: &newName})
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
	_, err = repo.UpdateRole(ctx, adminRole.Id.String(), tid, roles.RolePatch{Permissions: &downgraded})
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
	updated, err := repo.UpdateRole(ctx, roleA.Id.String(), tid, roles.RolePatch{Permissions: &downgraded})
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
	_, err = repo.UpdateRole(ctx, roleID, tid, roles.RolePatch{Name: &newName})
	require.ErrorIs(t, err, roles.ErrSystemRole)

	// ...nor rewrite its permissions (the actual privilege-escalation vector).
	newPerms := teams.PermissionsJSON{Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write"}
	_, err = repo.UpdateRole(ctx, roleID, tid, roles.RolePatch{Permissions: &newPerms})
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
	updated, err := repo.UpdateRole(ctx, roleID, tid, roles.RolePatch{Color: &newColor})
	require.NoError(t, err)
	assert.Equal(t, &newColor, updated.Color)
	assert.Equal(t, "Admin", updated.Name)
}
