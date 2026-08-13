package statsprefs_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/statsprefs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func seedUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, $2, $3, '#aaaaaa')`, id, name, email)
	require.NoError(t, err)
	return id
}

func seedTeam(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, $2)`, id, name)
	require.NoError(t, err)
	return id
}

func TestRepository_GetLastSelection_DefaultsWhenNoRow(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := statsprefs.NewRepository(pool)
	ctx := context.Background()

	teamID := seedTeam(t, ctx, pool, "Selection Default Team")
	userID := seedUser(t, ctx, pool, "Selection User", "selection@example.com")

	got, err := repo.GetLastSelection(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, statsprefs.LastSelection{}, got)
}

func TestRepository_UpsertLastSelection_RoundTrips(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := statsprefs.NewRepository(pool)
	ctx := context.Background()

	teamID := seedTeam(t, ctx, pool, "Selection Roundtrip Team")
	userID := seedUser(t, ctx, pool, "Selection User 2", "selection2@example.com")

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	sel := statsprefs.LastSelection{FromDate: &from, ToDate: &to}
	require.NoError(t, repo.UpsertLastSelection(ctx, teamID, userID, sel))

	got, err := repo.GetLastSelection(ctx, teamID, userID)
	require.NoError(t, err)
	require.NotNil(t, got.FromDate)
	require.NotNil(t, got.ToDate)
	assert.True(t, from.Equal(*got.FromDate))
	assert.True(t, to.Equal(*got.ToDate))
	assert.Nil(t, got.PresetID)

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM stats_last_selection WHERE team_id = $1 AND user_id = $2`, teamID, userID).Scan(&count))
	assert.Equal(t, 1, count)

	// Saving again updates the same row rather than duplicating it.
	to2 := to.AddDate(0, 1, 0)
	sel.ToDate = &to2
	require.NoError(t, repo.UpsertLastSelection(ctx, teamID, userID, sel))
	got, err = repo.GetLastSelection(ctx, teamID, userID)
	require.NoError(t, err)
	assert.True(t, to2.Equal(*got.ToDate))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM stats_last_selection WHERE team_id = $1 AND user_id = $2`, teamID, userID).Scan(&count))
	assert.Equal(t, 1, count, "saving a selection again must update, not duplicate")
}

func TestRepository_PresetCRUD(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := statsprefs.NewRepository(pool)
	ctx := context.Background()

	teamID := seedTeam(t, ctx, pool, "Preset CRUD Team")
	userID := seedUser(t, ctx, pool, "Preset User", "preset@example.com")

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 5, 31, 0, 0, 0, 0, time.UTC)

	count, err := repo.CountPresets(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	created, err := repo.CreatePreset(ctx, teamID, userID, "Saison 2026/27", from, to)
	require.NoError(t, err)
	assert.Equal(t, "Saison 2026/27", created.Name)
	assert.True(t, from.Equal(created.FromDate))
	assert.True(t, to.Equal(created.ToDate))
	require.NotEqual(t, uuid.Nil, created.ID)

	count, err = repo.CountPresets(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	presets, err := repo.ListPresets(ctx, teamID, userID)
	require.NoError(t, err)
	require.Len(t, presets, 1)
	assert.Equal(t, created.ID, presets[0].ID)

	newName := "Saison 2026/27 (final)"
	updated, err := repo.UpdatePreset(ctx, teamID, userID, created.ID, &newName, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.True(t, from.Equal(updated.FromDate), "from must be unchanged when not patched")

	require.NoError(t, repo.DeletePreset(ctx, teamID, userID, created.ID))
	presets, err = repo.ListPresets(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Empty(t, presets)

	// Deleting again (already gone) is idempotent, not an error.
	require.NoError(t, repo.DeletePreset(ctx, teamID, userID, created.ID))
}

func TestRepository_DeletePreset_ScopedToOwner(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := statsprefs.NewRepository(pool)
	ctx := context.Background()

	teamID := seedTeam(t, ctx, pool, "Preset Owner Team")
	owner := seedUser(t, ctx, pool, "Preset Owner", "preset-owner@example.com")
	attacker := seedUser(t, ctx, pool, "Preset Attacker", "preset-attacker@example.com")

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	created, err := repo.CreatePreset(ctx, teamID, owner, "Owner's Preset", from, to)
	require.NoError(t, err)

	// The attacker naming the owner's preset must not delete or modify it.
	require.NoError(t, repo.DeletePreset(ctx, teamID, attacker, created.ID))
	presets, err := repo.ListPresets(ctx, teamID, owner)
	require.NoError(t, err)
	require.Len(t, presets, 1, "a user must not be able to delete another user's preset")

	newName := "Hijacked"
	_, err = repo.UpdatePreset(ctx, teamID, attacker, created.ID, &newName, nil, nil)
	require.Error(t, err, "a user must not be able to update another user's preset")
}

// TestRepository_DeletingActivePreset_ClearsSelectionPresetID verifies the
// ON DELETE SET NULL behavior: deleting a preset that is the current
// selection's active preset degrades the selection row to its raw
// from/to dates instead of erroring or cascading the delete.
func TestRepository_DeletingActivePreset_ClearsSelectionPresetID(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := statsprefs.NewRepository(pool)
	ctx := context.Background()

	teamID := seedTeam(t, ctx, pool, "Preset Cascade Team")
	userID := seedUser(t, ctx, pool, "Preset Cascade User", "preset-cascade@example.com")

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	preset, err := repo.CreatePreset(ctx, teamID, userID, "Active Preset", from, to)
	require.NoError(t, err)

	presetID := preset.ID
	require.NoError(t, repo.UpsertLastSelection(ctx, teamID, userID, statsprefs.LastSelection{
		FromDate: &from, ToDate: &to, PresetID: &presetID,
	}))

	require.NoError(t, repo.DeletePreset(ctx, teamID, userID, presetID))

	got, err := repo.GetLastSelection(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Nil(t, got.PresetID, "deleting the active preset must clear preset_id, not error")
	require.NotNil(t, got.FromDate, "the selection row itself must survive the preset delete")
	assert.True(t, from.Equal(*got.FromDate))
}
