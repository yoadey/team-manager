package finances_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	tx, err := repo.CreateTransaction(ctx, teamID, "income", "Membership Fee", 5000, time.Now().UTC(), &category)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, "Membership Fee", tx.Title)
	assert.Equal(t, int64(5000), tx.Amount)

	expenseCategory := "gear"
	_, err = repo.CreateTransaction(ctx, teamID, "expense", "New Balls", 2000, time.Now().UTC(), &expenseCategory)
	require.NoError(t, err)

	list, err := repo.ListTransactions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "New Balls", list[0].Title) // ordered by date desc, created_at desc

	income, expense, err := repo.SumTransactions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), income)
	assert.Equal(t, int64(2000), expense)

	newTitle := "Updated Fee"
	updated, err := repo.UpdateTransaction(ctx, tx.ID, teamID, finances.TransactionPatch{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Updated Fee", updated.Title)

	// Empty patch (no-op path) must still be scoped to teamID -- a member of
	// another team must not be able to read this transaction via an empty PATCH body.
	otherTeamID := uuid.New()
	_, err = repo.UpdateTransaction(ctx, tx.ID, otherTeamID, finances.TransactionPatch{})
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	// The no-op path scoped to the correct team should still return the transaction.
	noop, err := repo.UpdateTransaction(ctx, tx.ID, teamID, finances.TransactionPatch{})
	require.NoError(t, err)
	assert.Equal(t, "Updated Fee", noop.Title)

	require.NoError(t, repo.DeleteTransaction(ctx, tx.ID, teamID))

	list, err = repo.ListTransactions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	income, expense, err = repo.SumTransactions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), income)
	assert.Equal(t, int64(2000), expense)
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

	pen, err := repo.CreatePenalty(ctx, teamID, "Late arrival", 500)
	require.NoError(t, err)
	require.NotNil(t, pen)
	assert.Equal(t, "Late arrival", pen.Label)
	assert.Equal(t, int64(500), pen.Amount)

	pens, err := repo.ListPenalties(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, pens, 1)

	newLabel := "Very late"
	updated, err := repo.UpdatePenalty(ctx, pen.ID, teamID, finances.PenaltyPatch{Label: &newLabel})
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

	fetched, err := repo.GetAssignmentByID(ctx, assign.ID, teamID)
	require.NoError(t, err)
	assert.Equal(t, assign.ID, fetched.ID)
	require.NotNil(t, fetched.PenaltyLabel)
	assert.Equal(t, "Very late", *fetched.PenaltyLabel)
	assert.NotEmpty(t, fetched.MemberName)

	_, err = repo.GetAssignmentByID(ctx, uuid.New(), teamID)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	toggled, err := repo.ToggleAssignmentPaid(ctx, assign.ID, teamID)
	require.NoError(t, err)
	assert.True(t, toggled.Paid)

	require.NoError(t, repo.DeleteAssignment(ctx, assign.ID, teamID))

	openPens, err := repo.ListOpenPenaltiesByUser(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, openPens)

	require.NoError(t, repo.DeletePenalty(ctx, pen.ID, teamID))

	pens, err = repo.ListPenalties(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, pens)
}

func TestFinancesRepository_Assignment_KeepsAmountSnapshotAfterPenaltyEdited(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	pen, err := repo.CreatePenalty(ctx, teamID, "Late arrival", 500)
	require.NoError(t, err)

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID)
	require.NoError(t, err)
	require.NotNil(t, assign.PenaltyAmount)
	assert.Equal(t, int64(500), *assign.PenaltyAmount)

	// Editing the penalty definition after the assignment was created must
	// not retroactively change what the assignment shows -- it's a
	// historical record of what was actually assigned, not a live view.
	newLabel := "Very late"
	newAmount := int64(5000)
	_, err = repo.UpdatePenalty(ctx, pen.ID, teamID, finances.PenaltyPatch{Label: &newLabel, Amount: &newAmount})
	require.NoError(t, err)

	fetched, err := repo.GetAssignmentByID(ctx, assign.ID, teamID)
	require.NoError(t, err)
	require.NotNil(t, fetched.PenaltyLabel)
	require.NotNil(t, fetched.PenaltyAmount)
	assert.Equal(t, "Late arrival", *fetched.PenaltyLabel)
	assert.Equal(t, int64(500), *fetched.PenaltyAmount)

	// A new assignment created after the edit does pick up the new amount.
	assign2, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID)
	require.NoError(t, err)
	require.NotNil(t, assign2.PenaltyAmount)
	assert.Equal(t, int64(5000), *assign2.PenaltyAmount)

	openPens, err := repo.ListOpenPenaltiesByUser(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, openPens, 1)
	assert.Equal(t, int64(5500), openPens[0].TotalAmount)
}

// Regression test: penalty_assignments.amount must be BIGINT (integer
// cents), matching penalties.amount, not the legacy NUMERIC(10,2) every
// other amount column moved off in migration 00008. NUMERIC(10,2) can't
// hold 100_000_000 cents (the app-allowed maxAmountCents, i.e. $1,000,000.00)
// without a numeric field overflow, so a penalty at the top of the allowed
// range could never actually be assigned to anyone.
func TestFinancesRepository_CreateAssignment_MaxAmountPenalty_Succeeds(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	const maxAmountCents = 100_000_000
	pen, err := repo.CreatePenalty(ctx, teamID, "Max Fine", maxAmountCents)
	require.NoError(t, err)

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID)
	require.NoError(t, err)
	require.NotNil(t, assign.PenaltyAmount)
	assert.Equal(t, int64(maxAmountCents), *assign.PenaltyAmount)
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
		 VALUES ($1, $2, '2024-06', 2500, 'open') RETURNING id`,
		teamID, userID,
	).Scan(&contribID)
	require.NoError(t, err)

	list, err := repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, contribID, list[0].ID)
	assert.Equal(t, "2024-06", list[0].Month)
	assert.Equal(t, int64(2500), list[0].Amount)
	assert.Equal(t, "open", list[0].Status)

	openCount, err := repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 1, openCount)

	// UpdateContribution: change label and amount.
	newLabel := "Monthly Fee"
	var newAmount int64 = 3000
	updated, err := repo.UpdateContribution(ctx, contribID, teamID, finances.ContributionPatch{
		Label:  &newLabel,
		Amount: &newAmount,
	})
	require.NoError(t, err)
	require.NotNil(t, updated.Label)
	assert.Equal(t, "Monthly Fee", *updated.Label)
	assert.Equal(t, int64(3000), updated.Amount)

	// ToggleContributionStatus: open → paid.
	toggled, err := repo.ToggleContributionStatus(ctx, contribID, teamID)
	require.NoError(t, err)
	assert.Equal(t, "paid", toggled.Status)

	openCount, err = repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 0, openCount)

	// Toggle again: paid → open.
	toggled, err = repo.ToggleContributionStatus(ctx, contribID, teamID)
	require.NoError(t, err)
	assert.Equal(t, "open", toggled.Status)

	// Cross-team access must be rejected, including via the empty-patch no-op path.
	otherTeamID := uuid.New()

	_, err = repo.UpdateContribution(ctx, contribID, otherTeamID, finances.ContributionPatch{})
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	newerLabel := "Hijacked"
	_, err = repo.UpdateContribution(ctx, contribID, otherTeamID, finances.ContributionPatch{Label: &newerLabel})
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	_, err = repo.ToggleContributionStatus(ctx, contribID, otherTeamID)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	// The contribution must be untouched by the rejected attempts.
	list, err = repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "open", list[0].Status)
}

// TestFinancesRepository_ToggleContributionStatus_ConcurrentTogglesDontLoseUpdates
// guards against a read-then-write race: ToggleContributionStatus previously
// read the current status, computed the flip in Go, then wrote it back in a
// separate statement — two concurrent toggles could both read "open", both
// compute "paid", and both write "paid", losing one of the two toggle
// intents (net effect of two toggles should be back to "open"). The fix
// flips the status atomically in a single UPDATE ... CASE statement.
func TestFinancesRepository_ToggleContributionStatus_ConcurrentTogglesDontLoseUpdates(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	var contribID uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO contributions (team_id, user_id, month, amount, status)
		 VALUES ($1, $2, '2024-07', 1500, 'open') RETURNING id`,
		teamID, userID,
	).Scan(&contribID)
	require.NoError(t, err)

	const n = 20
	errs := make(chan error, n)
	for range n {
		go func() {
			_, err := repo.ToggleContributionStatus(ctx, contribID, teamID)
			errs <- err
		}()
	}
	for range n {
		require.NoError(t, <-errs)
	}

	// An even number of toggles must land back on the original status —
	// a lost update would instead leave it stuck on "paid".
	list, err := repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "open", list[0].Status)
}

// TestFinancesRepository_UserIsMemberOfTeam guards against regressing to a
// nonexistent table name (the query must target `memberships`, the table
// every other module uses — a prior version queried a nonexistent
// `team_members` table, which made CreateAssignment always fail its
// membership check in production).
func TestFinancesRepository_UserIsMemberOfTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	isMember, err := repo.UserIsMemberOfTeam(ctx, userID, teamID)
	require.NoError(t, err)
	assert.False(t, isMember, "user has no membership row yet")

	_, err = pool.Exec(ctx,
		`INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	isMember, err = repo.UserIsMemberOfTeam(ctx, userID, teamID)
	require.NoError(t, err)
	assert.True(t, isMember)

	// A user who is not a member of this team must not be reported as one.
	otherUserID := uuid.New()
	isMember, err = repo.UserIsMemberOfTeam(ctx, otherUserID, teamID)
	require.NoError(t, err)
	assert.False(t, isMember)
}

// TestFinancesRepository_CreateAssignment_RejectsNonMemberUser regression-tests
// a TOCTOU gap where CreateAssignment's plain INSERT trusted the service
// layer's earlier, separate UserIsMemberOfTeam check -- penalty_assignments.
// user_id only references users(id), not memberships, so nothing at the DB
// level stopped an assignment being created for a user who was removed from
// the team in the narrow window between that check and this insert. The
// INSERT now re-checks membership atomically via WHERE EXISTS.
func TestFinancesRepository_CreateAssignment_RejectsNonMemberUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	penalty, err := repo.CreatePenalty(ctx, teamID, "Missed practice", 500)
	require.NoError(t, err)

	// uid was never given a membership row by seedFinanceFixtures.
	_, err = repo.CreateAssignment(ctx, teamID, userID, penalty.ID)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "CreateAssignment must reject a userID that is not a member of teamID")

	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	a, err := repo.CreateAssignment(ctx, teamID, userID, penalty.ID)
	require.NoError(t, err, "CreateAssignment scoped to an actual team member must succeed")
	assert.Equal(t, userID, a.UserID)
}

// TestFinancesRepository_CreateAssignment_RejectsDeletedPenalty regression-tests
// the sibling TOCTOU race on the penalty side: penalty_id does have a real FK
// (unlike user_id), so a penalty deleted concurrently between the service
// layer's PenaltyBelongsToTeam check and this insert surfaces as a Postgres
// foreign-key violation (23503). CreateAssignment must map that to
// ErrPenaltyNotInTeam so the handler returns 404/422 instead of a generic 500.
func TestFinancesRepository_CreateAssignment_RejectsDeletedPenalty(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	penalty, err := repo.CreatePenalty(ctx, teamID, "Missed practice", 500)
	require.NoError(t, err)

	require.NoError(t, repo.DeletePenalty(ctx, penalty.ID, teamID))

	_, err = repo.CreateAssignment(ctx, teamID, userID, penalty.ID)
	assert.ErrorIs(t, err, finances.ErrPenaltyNotInTeam, "CreateAssignment must map a penalty FK violation to ErrPenaltyNotInTeam")
}
