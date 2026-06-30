package finances_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func seedFinanceFixtures(t *testing.T, pool *pgxpool.Pool, uid, tid string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Finance User', 'finance@example.com', '#996633')`, uid)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Finance Team')`, tid)
	if err != nil {
		t.Fatalf("seed team: %v", err)
	}
}

func TestFinancesRepository_Transactions(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	category := "income"
	tx, err := repo.CreateTransaction(ctx, teamID, "income", "Membership Fee", 50.0, time.Now().UTC(), &category)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, "Membership Fee", tx.Title)
	assert.Equal(t, 50.0, tx.Amount)

	list, err := repo.ListTransactions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, tx.ID, list[0].ID)

	newTitle := "Updated Fee"
	updated, err := repo.UpdateTransaction(ctx, tx.ID, finances.TransactionPatch{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Updated Fee", updated.Title)

	require.NoError(t, repo.DeleteTransaction(ctx, tx.ID))

	list, err = repo.ListTransactions(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestFinancesRepository_Penalties(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pen, err := repo.CreatePenalty(ctx, teamID, "Late arrival", 5.0)
	require.NoError(t, err)
	require.NotNil(t, pen)
	assert.Equal(t, "Late arrival", pen.Label)
	assert.Equal(t, 5.0, pen.Amount)

	pens, err := repo.ListPenalties(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, pens, 1)

	newLabel := "Very late"
	updated, err := repo.UpdatePenalty(ctx, pen.ID, finances.PenaltyPatch{Label: &newLabel})
	require.NoError(t, err)
	assert.Equal(t, "Very late", updated.Label)

	// Need membership + assignment to test listing.
	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID)
	require.NoError(t, err)
	require.NotNil(t, assign)

	assignments, err := repo.ListAssignments(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)

	toggled, err := repo.ToggleAssignmentPaid(ctx, assign.ID)
	require.NoError(t, err)
	assert.True(t, toggled.Paid)

	require.NoError(t, repo.DeleteAssignment(ctx, assign.ID))

	openPens, err := repo.ListOpenPenaltiesByUser(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, openPens)

	require.NoError(t, repo.DeletePenalty(ctx, pen.ID))

	pens, err = repo.ListPenalties(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, pens)
}

func TestFinancesRepository_Contributions(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	// Seed a contribution directly (no Create method in the repo — contributions
	// are generated via a separate admin flow or seeded by the service layer).
	var contribID uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO contributions (team_id, user_id, month, amount, status)
		 VALUES ($1, $2, '2024-06', 25.00, 'open') RETURNING id`,
		teamID, userID,
	).Scan(&contribID)
	require.NoError(t, err)

	list, err := repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, contribID, list[0].ID)
	assert.Equal(t, "2024-06", list[0].Month)
	assert.Equal(t, 25.0, list[0].Amount)
	assert.Equal(t, "open", list[0].Status)

	// UpdateContribution: change label and amount.
	newLabel := "Monthly Fee"
	newAmount := 30.0
	updated, err := repo.UpdateContribution(ctx, contribID, finances.ContributionPatch{
		Label:  &newLabel,
		Amount: &newAmount,
	})
	require.NoError(t, err)
	require.NotNil(t, updated.Label)
	assert.Equal(t, "Monthly Fee", *updated.Label)
	assert.Equal(t, 30.0, updated.Amount)

	// ToggleContributionStatus: open → paid.
	toggled, err := repo.ToggleContributionStatus(ctx, contribID)
	require.NoError(t, err)
	assert.Equal(t, "paid", toggled.Status)

	// Toggle again: paid → open.
	toggled, err = repo.ToggleContributionStatus(ctx, contribID)
	require.NoError(t, err)
	assert.Equal(t, "open", toggled.Status)
}
