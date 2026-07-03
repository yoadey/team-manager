package teams_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestTeamRepository_CreateTeam(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Test User', 'create@example.com', '#aabbcc')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "My Team", userID)
	require.NoError(t, err)
	assert.NotEmpty(t, tr.Id.String())
	assert.Equal(t, "My Team", tr.Name)
	assert.False(t, tr.CreatedAt.IsZero())
}

func TestTeamRepository_ListForUser(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('List User', 'list@example.com', '#ccddee')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	_, err = repo.CreateTeam(ctx, "List Team", userID)
	require.NoError(t, err)

	result, err := repo.ListTeamsForUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "List Team", result[0].Name)
}

func TestTeamRepository_UpdateTeam(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Update User', 'update@example.com', '#112233')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Original Name", userID)
	require.NoError(t, err)

	newName := "Updated Name"
	updated, err := repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, tr.Id, updated.Id)
}

func TestTeamRepository_UpdateTeam_ReasonVisibilityRoleIDs_ValidatesOwnership(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Reason Vis User', 'reason-vis@example.com', '#445566')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Reason Vis Team", userID)
	require.NoError(t, err)

	// A role from a different team must be rejected.
	otherTeam, err := repo.CreateTeam(ctx, "Other Team", userID)
	require.NoError(t, err)
	var foreignRoleID string
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Foreign Role', '{}') RETURNING id`,
		otherTeam.Id.String(),
	).Scan(&foreignRoleID)
	require.NoError(t, err)

	_, err = repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{
		ReasonVisibilityRoleIDs: []string{foreignRoleID},
	})
	require.ErrorIs(t, err, teams.ErrRoleNotInTeam)

	// A role belonging to the team is accepted.
	var ownRoleID string
	err = pool.QueryRow(ctx,
		`INSERT INTO roles (team_id, name, permissions) VALUES ($1, 'Own Role', '{}') RETURNING id`,
		tr.Id.String(),
	).Scan(&ownRoleID)
	require.NoError(t, err)

	updated, err := repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{
		ReasonVisibilityRoleIDs: []string{ownRoleID},
	})
	require.NoError(t, err)
	require.Len(t, updated.ReasonVisibilityRoleIDs, 1)
	assert.Equal(t, uuid.MustParse(ownRoleID), updated.ReasonVisibilityRoleIDs[0])
}
