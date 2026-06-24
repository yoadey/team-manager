package roles_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/yoadey/team-manager/backend/internal/gen"
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
	assert.Equal(t, gen.PermLevel("write"), role.Permissions.Events)
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
	updated, err := repo.UpdateRole(ctx, role.Id.String(), roles.RolePatch{
		Name: &newName, Color: &newColor, Permissions: &newPerms,
	})
	require.NoError(t, err)
	assert.Equal(t, "Captain", updated.Name)
	assert.Equal(t, &newColor, updated.Color)
	assert.Equal(t, gen.PermLevel("write"), updated.Permissions.Events)
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

	require.NoError(t, repo.DeleteRole(ctx, role.Id.String()))

	list, err := repo.ListRoles(ctx, tid)
	require.NoError(t, err)
	assert.Empty(t, list)
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

	err = repo.DeleteRole(ctx, roleID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "system")
}
