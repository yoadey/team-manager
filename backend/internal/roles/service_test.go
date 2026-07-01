package roles_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/roles"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listRolesFn  func(ctx context.Context, teamID string) ([]teams.RoleRow, error)
	createRoleFn func(ctx context.Context, teamID, name string, color *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error)
	updateRoleFn func(ctx context.Context, roleID, teamID string, patch roles.RolePatch) (*teams.RoleRow, error)
	deleteRoleFn func(ctx context.Context, roleID, teamID string) error
}

func (m *mockRepo) ListRoles(ctx context.Context, teamID string) ([]teams.RoleRow, error) {
	return m.listRolesFn(ctx, teamID)
}

func (m *mockRepo) CreateRole(ctx context.Context, teamID, name string, color *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error) {
	return m.createRoleFn(ctx, teamID, name, color, permissions)
}

func (m *mockRepo) UpdateRole(ctx context.Context, roleID, teamID string, patch roles.RolePatch) (*teams.RoleRow, error) {
	return m.updateRoleFn(ctx, roleID, teamID, patch)
}

func (m *mockRepo) DeleteRole(ctx context.Context, roleID, teamID string) error {
	return m.deleteRoleFn(ctx, roleID, teamID)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_ListRoles(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		listRolesFn: func(_ context.Context, gotTeamID string) ([]teams.RoleRow, error) {
			assert.Equal(t, teamID.String(), gotTeamID)
			return []teams.RoleRow{
				{Id: uuid.New(), TeamID: teamID, Name: "Trainer", System: true, Permissions: teams.PermissionsJSON{Events: "write", Members: "read"}},
			}, nil
		},
	}

	svc := roles.NewService(repo)
	result, err := svc.ListRoles(context.Background(), teamID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Trainer", result[0].Name)
	assert.True(t, result[0].System)
	assert.Equal(t, gen.Write, result[0].Permissions.Events)
	assert.Equal(t, gen.Read, result[0].Permissions.Members)
}

func TestService_CreateRole_MapsPermissionsToInternalRepresentation(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	var capturedPerms teams.PermissionsJSON
	repo := &mockRepo{
		createRoleFn: func(_ context.Context, gotTeamID, name string, _ *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error) {
			assert.Equal(t, teamID.String(), gotTeamID)
			assert.Equal(t, "Coach", name)
			capturedPerms = permissions
			return &teams.RoleRow{Id: uuid.New(), TeamID: teamID, Name: name, Permissions: permissions}, nil
		},
	}

	svc := roles.NewService(repo)
	body := &gen.CreateRoleJSONRequestBody{
		Name: "Coach",
		Permissions: gen.Permissions{
			Events:   gen.Write,
			Members:  gen.Read,
			Finances: gen.None,
			News:     gen.Write,
			Polls:    gen.Read,
			Settings: gen.None,
		},
	}
	result, err := svc.CreateRole(context.Background(), teamID, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "write", capturedPerms.Events)
	assert.Equal(t, "read", capturedPerms.Members)
	assert.Equal(t, "none", capturedPerms.Finances)
	assert.Equal(t, "write", capturedPerms.News)
	assert.Equal(t, "read", capturedPerms.Polls)
	assert.Equal(t, "none", capturedPerms.Settings)
}

func TestService_UpdateRole_LeavesPermissionsNilWhenNotProvided(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	teamID := uuid.New()
	newName := "Renamed Role"
	var capturedPatch roles.RolePatch
	repo := &mockRepo{
		updateRoleFn: func(_ context.Context, gotRoleID, gotTeamID string, patch roles.RolePatch) (*teams.RoleRow, error) {
			assert.Equal(t, roleID.String(), gotRoleID)
			assert.Equal(t, teamID.String(), gotTeamID)
			capturedPatch = patch
			return &teams.RoleRow{Id: roleID, Name: newName}, nil
		},
	}

	svc := roles.NewService(repo)
	body := &gen.UpdateRoleJSONRequestBody{Name: &newName}
	_, err := svc.UpdateRole(context.Background(), roleID, teamID, body)
	require.NoError(t, err)

	require.NotNil(t, capturedPatch.Name)
	assert.Equal(t, newName, *capturedPatch.Name)
	assert.Nil(t, capturedPatch.Permissions, "permissions should stay nil when not provided in the request body")
}

func TestService_UpdateRole_MapsPermissionsWhenProvided(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	teamID := uuid.New()
	var capturedPatch roles.RolePatch
	repo := &mockRepo{
		updateRoleFn: func(_ context.Context, _, _ string, patch roles.RolePatch) (*teams.RoleRow, error) {
			capturedPatch = patch
			return &teams.RoleRow{Id: roleID}, nil
		},
	}

	svc := roles.NewService(repo)
	perms := gen.Permissions{Events: gen.Write, Members: gen.None, Finances: gen.None, News: gen.None, Polls: gen.None, Settings: gen.Read}
	body := &gen.UpdateRoleJSONRequestBody{Permissions: &perms}
	_, err := svc.UpdateRole(context.Background(), roleID, teamID, body)
	require.NoError(t, err)

	require.NotNil(t, capturedPatch.Permissions)
	assert.Equal(t, "write", capturedPatch.Permissions.Events)
	assert.Equal(t, "read", capturedPatch.Permissions.Settings)
}

func TestService_UpdateRole_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	wrongTeamID := uuid.New()
	newName := "Attacker Renamed"
	repo := &mockRepo{
		updateRoleFn: func(context.Context, string, string, roles.RolePatch) (*teams.RoleRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := roles.NewService(repo)
	body := &gen.UpdateRoleJSONRequestBody{Name: &newName}
	_, err := svc.UpdateRole(context.Background(), roleID, wrongTeamID, body)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestService_DeleteRole(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	teamID := uuid.New()
	called := false
	repo := &mockRepo{
		deleteRoleFn: func(_ context.Context, gotRoleID, gotTeamID string) error {
			assert.Equal(t, roleID.String(), gotRoleID)
			assert.Equal(t, teamID.String(), gotTeamID)
			called = true
			return nil
		},
	}

	svc := roles.NewService(repo)
	err := svc.DeleteRole(context.Background(), roleID, teamID)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_DeleteRole_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("cannot delete system role")
	repo := &mockRepo{
		deleteRoleFn: func(context.Context, string, string) error { return wantErr },
	}

	svc := roles.NewService(repo)
	err := svc.DeleteRole(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_DeleteRole_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		deleteRoleFn: func(context.Context, string, string) error { return pgx.ErrNoRows },
	}

	svc := roles.NewService(repo)
	err := svc.DeleteRole(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
