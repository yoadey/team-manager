package absences_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

const (
	repoTestTeamID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	repoTestUserID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func TestAbsenceRepository_CreateAndList(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Absent User', 'abs@example.com', '#ff0000')`,
		repoTestUserID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Absence Team')`,
		repoTestTeamID)
	require.NoError(t, err)

	teamID := uuid.MustParse(repoTestTeamID)
	userID := uuid.MustParse(repoTestUserID)

	from := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	to := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	reason := "holiday"

	ab, err := repo.Create(ctx, teamID, userID, from, to, &reason)
	require.NoError(t, err)
	require.NotNil(t, ab)
	assert.Equal(t, teamID, ab.TeamId)
	assert.Equal(t, userID, ab.UserId)
	assert.Equal(t, &reason, ab.Reason)
	assert.Equal(t, "Absent User", *ab.MemberName)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, ab.Id, all[0].Id)

	mine, err := repo.ListByUser(ctx, teamID, userID, 50, nil)
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, ab.Id, mine[0].Id)
}

func TestAbsenceRepository_Update(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Update User', 'upd@example.com', '#aaaaaa')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Upd Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	from := "2025-03-01"
	to := "2025-03-07"
	ab, err := repo.Create(ctx, teamID, userID, from, to, nil)
	require.NoError(t, err)

	newTo := "2025-03-14"
	newReason := "extended vacation"
	updated, err := repo.Update(ctx, ab.Id, teamID, userID, nil, &newTo, &newReason)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, newTo, updated.ToDate.Format("2006-01-02"))
	assert.Equal(t, &newReason, updated.Reason)
}

func TestAbsenceRepository_Update_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Cross Team User', 'crossteam@example.com', '#123456')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Owning Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Attacker Team')`,
		otherTid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	otherTeamID := uuid.MustParse(otherTid)
	userID := uuid.MustParse(uid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-03-01", "2025-03-07", nil)
	require.NoError(t, err)

	newReason := "attacker-supplied reason"
	_, err = repo.Update(ctx, ab.Id, otherTeamID, userID, nil, nil, &newReason)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestAbsenceRepository_Update_WrongUser_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	otherUid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Owner User', 'owner@example.com', '#111111')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Other Member', 'othermember@example.com', '#222222')`,
		otherUid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Shared Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	otherUserID := uuid.MustParse(otherUid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-03-01", "2025-03-07", nil)
	require.NoError(t, err)

	// Another member of the same team must not be able to update this absence.
	newReason := "teammate-supplied reason"
	_, err = repo.Update(ctx, ab.Id, teamID, otherUserID, nil, nil, &newReason)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestAbsenceRepository_Delete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del User', 'del@example.com', '#bbbbbb')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.NoError(t, err)

	err = repo.Delete(ctx, ab.Id, teamID, userID)
	require.NoError(t, err)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestAbsenceRepository_Delete_WrongTeam_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	otherTid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Cross User', 'delcross@example.com', '#654321')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Owning Team')`,
		tid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Attacker Team')`,
		otherTid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	otherTeamID := uuid.MustParse(otherTid)
	userID := uuid.MustParse(uid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.NoError(t, err)

	err = repo.Delete(ctx, ab.Id, otherTeamID, userID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	// Absence must still exist under the real team.
	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestAbsenceRepository_Delete_WrongUser_ReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := absences.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	otherUid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Owner User', 'delowner@example.com', '#333333')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Other Member', 'delothermember@example.com', '#444444')`,
		otherUid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Shared Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	otherUserID := uuid.MustParse(otherUid)
	ab, err := repo.Create(ctx, teamID, userID, "2025-05-01", "2025-05-05", nil)
	require.NoError(t, err)

	// Another member of the same team must not be able to delete this absence.
	err = repo.Delete(ctx, ab.Id, teamID, otherUserID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	all, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
