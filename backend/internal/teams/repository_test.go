package teams_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	// Regression: `COUNT(*) FROM roles WHERE id = ANY($1)` counts matching
	// rows once per distinct role, so comparing it against len(ids) directly
	// used to wrongly reject a request that legitimately repeats the same
	// valid role ID. This only asserts the (valid) request is accepted, not
	// that storage deduplicates -- deduping the stored array is a separate,
	// unrelated concern.
	updated, err = repo.UpdateTeam(ctx, tr.Id.String(), teams.TeamPatch{
		ReasonVisibilityRoleIDs: []string{ownRoleID, ownRoleID},
	})
	require.NoError(t, err)
	require.Len(t, updated.ReasonVisibilityRoleIDs, 2)
	assert.Equal(t, uuid.MustParse(ownRoleID), updated.ReasonVisibilityRoleIDs[0])
	assert.Equal(t, uuid.MustParse(ownRoleID), updated.ReasonVisibilityRoleIDs[1])
}

func TestTeamRepository_DeleteTeamPhoto_ClearsStoredPhoto(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Photo Delete User', 'photo-delete@example.com', '#778899')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Photo Delete Team", userID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateTeamPhoto(ctx, tr.Id.String(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"))
	withPhoto, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	require.NotEmpty(t, withPhoto.PhotoData)

	require.NoError(t, repo.DeleteTeamPhoto(ctx, tr.Id.String()))
	cleared, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.Empty(t, cleared.PhotoData)
	assert.Nil(t, cleared.PhotoMime)
}

func TestTeamRepository_DeleteTeamPhoto_UnknownTeam_ReturnsNoRows(t *testing.T) {
	pool := testutil.NewTestDB(t)
	repo := teams.NewRepository(pool)
	err := repo.DeleteTeamPhoto(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestTeamRepository_DeleteTeamLogo_ClearsStoredLogo(t *testing.T) {
	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, avatar_color)
		VALUES ('Logo Delete User', 'logo-delete@example.com', '#998877')
		RETURNING id
	`).Scan(&userID)
	require.NoError(t, err)

	repo := teams.NewRepository(pool)
	tr, err := repo.CreateTeam(ctx, "Logo Delete Team", userID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateTeamLogo(ctx, tr.Id.String(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"))
	withLogo, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	require.NotEmpty(t, withLogo.LogoData)

	require.NoError(t, repo.DeleteTeamLogo(ctx, tr.Id.String()))
	cleared, err := repo.GetTeam(ctx, tr.Id.String())
	require.NoError(t, err)
	assert.Empty(t, cleared.LogoData)
	assert.Nil(t, cleared.LogoMime)
}

func TestTeamRepository_DeleteTeamLogo_UnknownTeam_ReturnsNoRows(t *testing.T) {
	pool := testutil.NewTestDB(t)
	repo := teams.NewRepository(pool)
	err := repo.DeleteTeamLogo(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, pgx.ErrNoRows)
}
