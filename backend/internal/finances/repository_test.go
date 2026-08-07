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
	tx, err := repo.CreateTransaction(ctx, teamID, "income", "Membership Fee", 5000, time.Now().UTC(), &category, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, "Membership Fee", tx.Title)
	assert.Equal(t, int64(5000), tx.Amount)

	expenseCategory := "gear"
	_, err = repo.CreateTransaction(ctx, teamID, "expense", "New Balls", 2000, time.Now().UTC(), &expenseCategory, nil, nil, nil)
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

func TestFinancesRepository_ListTransactionsPage_KeysetPaginatesWholeHistory(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	// Create 5 transactions on distinct, ascending dates so the newest-first
	// ordering is deterministic and easy to assert against.
	const total = 5
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < total; i++ {
		cat := "dues"
		_, err := repo.CreateTransaction(ctx, teamID, "income", "tx", int64(100+i), base.AddDate(0, 0, i), &cat, nil, nil, nil)
		require.NoError(t, err)
	}

	// Page through 2 at a time, following the cursor, and collect every row.
	var seen []string
	var cur *finances.TxCursor
	pages := 0
	for {
		page, err := repo.ListTransactionsPage(ctx, teamID, 2, cur)
		require.NoError(t, err)
		if len(page) == 0 {
			break
		}
		pages++
		for _, row := range page {
			seen = append(seen, row.ID.String())
		}
		if len(page) < 2 {
			break
		}
		last := page[len(page)-1]
		cur = &finances.TxCursor{Date: last.Date, CreatedAt: last.CreatedAt, ID: last.ID}
		require.LessOrEqual(t, pages, total+1, "pagination must terminate")
	}

	assert.Len(t, seen, total, "every transaction must be reachable by paging")
	// No row appears twice across pages (keyset correctness).
	unique := map[string]struct{}{}
	for _, id := range seen {
		_, dup := unique[id]
		assert.False(t, dup, "keyset pagination must not repeat a row across pages")
		unique[id] = struct{}{}
	}

	// First page must be newest-first: the latest date (base+4) comes first.
	first, err := repo.ListTransactionsPage(ctx, teamID, 2, nil)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.True(t, first[0].Date.After(first[1].Date) || first[0].Date.Equal(first[1].Date),
		"page must be ordered newest-first")
	assert.Equal(t, int64(104), first[0].Amount, "newest transaction (base+4 days) must sort first")
}

func TestFinancesRepository_ListTransactionsPage_ScopedToTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	cat := "dues"
	_, err := repo.CreateTransaction(ctx, teamID, "income", "tx", 100, time.Now().UTC(), &cat, nil, nil, nil)
	require.NoError(t, err)

	// A different team sees none of this team's rows.
	other, err := repo.ListTransactionsPage(ctx, uuid.New(), 50, nil)
	require.NoError(t, err)
	assert.Empty(t, other)
}

func TestFinancesRepository_UpdateTransaction_SetsDate(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	cat := "dues"
	tx, err := repo.CreateTransaction(ctx, teamID, "income", "tx", 100, time.Now().UTC(), &cat, nil, nil, nil)
	require.NoError(t, err)

	want := time.Date(2022, 6, 30, 0, 0, 0, 0, time.UTC)
	updated, err := repo.UpdateTransaction(ctx, tx.ID, teamID, finances.TransactionPatch{Date: &want})
	require.NoError(t, err)
	assert.True(t, want.Equal(updated.Date), "the date patch must be persisted")
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

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
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

	payCategory := "fine"
	_, err = repo.CreateTransaction(ctx, teamID, "income", "Fine payment", 500, time.Now().UTC(), &payCategory, nil, &assign.ID, nil)
	require.NoError(t, err)

	toggled, err := repo.GetAssignmentByID(ctx, assign.ID, teamID)
	require.NoError(t, err)
	assert.Equal(t, int64(500), toggled.PaidAmount, "PaidAmount must be the live sum of linked income transactions")
	require.NotNil(t, toggled.PenaltyLabel)
	assert.Equal(t, "Very late", *toggled.PenaltyLabel)
	require.NotNil(t, toggled.PenaltyAmount)
	assert.Equal(t, int64(500), *toggled.PenaltyAmount)

	require.NoError(t, repo.DeleteAssignment(ctx, assign.ID, teamID))

	openPens, err := repo.ListOpenPenaltiesByUser(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, openPens)

	require.NoError(t, repo.DeletePenalty(ctx, pen.ID, teamID))

	pens, err = repo.ListPenalties(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, pens)
}

// Deleting a penalty catalog entry must NOT remove its assignments (paid or
// unpaid). Migration 00027 changed the FK to ON DELETE SET NULL, so each
// assignment survives with a null penalty_id and is still fully rendered from
// its snapshot label/amount -- preserving the immutable financial record 00025
// established (the old ON DELETE CASCADE erased paid history on catalog tidy-up).
func TestFinancesRepository_DeletePenalty_PreservesAssignments(t *testing.T) {
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

	pen, err := repo.CreatePenalty(ctx, teamID, "Late fee", 500)
	require.NoError(t, err)

	// One paid and one unpaid assignment of the same penalty.
	paid, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
	require.NoError(t, err)
	payCategory := "fine"
	_, err = repo.CreateTransaction(ctx, teamID, "income", "Fine payment", 500, time.Now().UTC(), &payCategory, nil, &paid.ID, nil)
	require.NoError(t, err)
	_, err = repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
	require.NoError(t, err)

	// Delete the catalog penalty.
	require.NoError(t, repo.DeletePenalty(ctx, pen.ID, teamID))
	pens, err := repo.ListPenalties(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, pens, "the penalty catalog entry itself is gone")

	// Both assignments must survive, detached (null penalty_id) but with their
	// snapshot label/amount intact.
	assignments, err := repo.ListAssignments(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, assignments, 2, "assignments must survive deletion of their penalty, not cascade away")
	for _, a := range assignments {
		assert.Nil(t, a.PenaltyID, "penalty_id must be nulled (SET NULL), not the row deleted")
		require.NotNil(t, a.PenaltyLabel)
		assert.Equal(t, "Late fee", *a.PenaltyLabel, "snapshot label must remain the record")
		require.NotNil(t, a.PenaltyAmount)
		assert.Equal(t, int64(500), *a.PenaltyAmount, "snapshot amount must remain the record")
	}
}

// CreateAssignment's snapshot read is team-scoped in the query itself
// (WHERE id = $1 AND team_id = $2), so a penalty belonging to another team can
// never be assigned even if the id is known -- defense-in-depth matching the
// repository's otherwise-uniform in-query tenant scoping.
func TestFinancesRepository_CreateAssignment_RejectsPenaltyFromAnotherTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamA := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)
	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	// A penalty owned by a different team B.
	otherTid := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Other Team')`, otherTid)
	require.NoError(t, err)
	penB, err := repo.CreatePenalty(ctx, uuid.MustParse(otherTid), "Team B fee", 500)
	require.NoError(t, err)

	// Assigning team B's penalty within team A must be rejected.
	_, err = repo.CreateAssignment(ctx, teamA, userID, penB.ID, time.Now(), nil)
	assert.ErrorIs(t, err, finances.ErrPenaltyNotInTeam)
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

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
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
	assign2, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
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

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
	require.NoError(t, err)
	require.NotNil(t, assign.PenaltyAmount)
	assert.Equal(t, int64(maxAmountCents), *assign.PenaltyAmount)
}

// TestFinancesRepository_CreateAssignment_PersistsPastDate verifies that a
// caller-supplied earned-date in the past is stored as given, not silently
// replaced with today's date (penalty_assignments.date used to always default
// to CURRENT_DATE since CreateAssignment never accepted an explicit value).
func TestFinancesRepository_CreateAssignment_PersistsPastDate(t *testing.T) {
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

	pastDate := time.Now().UTC().AddDate(0, -1, 0).Truncate(24 * time.Hour)
	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, pastDate, nil)
	require.NoError(t, err)
	assert.True(t, pastDate.Equal(assign.Date), "expected date %v, got %v", pastDate, assign.Date)
	assert.NotEqual(t, time.Now().UTC().Truncate(24*time.Hour), assign.Date, "must not silently default to today")

	fetched, err := repo.GetAssignmentByID(ctx, assign.ID, teamID)
	require.NoError(t, err)
	assert.True(t, pastDate.Equal(fetched.Date))
}

// TestFinancesRepository_CreateAssignment_PersistsNote verifies that a
// caller-supplied note is stored and can be read back.
func TestFinancesRepository_CreateAssignment_PersistsNote(t *testing.T) {
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

	note := "Missed training without excuse"
	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), &note)
	require.NoError(t, err)
	require.NotNil(t, assign.Note)
	assert.Equal(t, note, *assign.Note)

	fetched, err := repo.GetAssignmentByID(ctx, assign.ID, teamID)
	require.NoError(t, err)
	require.NotNil(t, fetched.Note)
	assert.Equal(t, note, *fetched.Note)
}

// TestFinancesRepository_CreateAssignment_WithoutNote_Succeeds verifies that
// omitting the note (nil) creates the assignment successfully with no note
// recorded, rather than failing or storing an empty string.
func TestFinancesRepository_CreateAssignment_WithoutNote_Succeeds(t *testing.T) {
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

	assign, err := repo.CreateAssignment(ctx, teamID, userID, pen.ID, time.Now(), nil)
	require.NoError(t, err)
	assert.Nil(t, assign.Note)
}

// TestFinancesRepository_CreateAssignment_ToleratesPreSnapshotInsert is a
// regression test for a rolling-deploy hazard: penalty_assignments.amount/
// label (see 00001_init.sql's comment) are deliberately nullable rather than
// NOT NULL, since under Kubernetes' RollingUpdate strategy (the default with
// replicaCount > 1) an old-version pod's binary could still issue
// "INSERT INTO penalty_assignments (team_id, user_id, penalty_id) ..." with
// no amount/label during a rollout window that adds a NOT NULL constraint --
// every one of those concurrent old-pod inserts would then fail. This
// directly exercises that exact old-binary INSERT shape against the current
// schema and asserts it still succeeds, leaving amount/label NULL --
// tolerated end-to-end since PenaltyAssignmentRow.PenaltyAmount/PenaltyLabel
// and the generated OpenAPI type are already nullable (*int64/*string).
func TestFinancesRepository_CreateAssignment_ToleratesPreSnapshotInsert(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	pen, err := repo.CreatePenalty(ctx, teamID, "Pre-snapshot Fine", 500)
	require.NoError(t, err)

	var assignmentID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO penalty_assignments (team_id, user_id, penalty_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, tid, uid, pen.ID).Scan(&assignmentID)
	require.NoError(t, err, "an old-binary INSERT omitting amount/label must not be rejected by a NOT NULL constraint during a rolling deploy")

	var amount *int64
	var label *string
	err = pool.QueryRow(ctx, `SELECT amount, label FROM penalty_assignments WHERE id = $1`, assignmentID).Scan(&amount, &label)
	require.NoError(t, err)
	assert.Nil(t, amount)
	assert.Nil(t, label)
}

// TestFinancesRepository_ListOpenPenaltiesByUser_TreatsNullAmountAsZero is a
// companion regression test to
// TestFinancesRepository_CreateAssignment_ToleratesPreSnapshotInsert: an
// old-binary INSERT during a rolling deploy can leave a user's ONLY unpaid
// penalty_assignments row with amount = NULL. SUM() over a Postgres group
// where every contributing row is NULL returns SQL NULL, not 0 -- unlike the
// row-level read paths (ListAssignments etc.), which already tolerate a NULL
// amount via a nullable *int64 field, ListOpenPenaltiesByUser scans straight
// into a non-pointer int64 TotalAmount, so an unguarded SUM(pa.amount) would
// fail that scan and take down the whole finance overview endpoint for the
// team (GetOverview runs every read, including this one, inside a single
// transaction that returns an error on any failure). Mirrors the
// COALESCE(SUM(...), 0) pattern already used by SumTransactions.
func TestFinancesRepository_ListOpenPenaltiesByUser_TreatsNullAmountAsZero(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	pen, err := repo.CreatePenalty(ctx, teamID, "Pre-snapshot Fine", 500)
	require.NoError(t, err)

	// The exact old-binary INSERT shape from the sibling test above: no
	// amount/label at all, leaving this the user's only open assignment.
	_, err = pool.Exec(ctx, `
		INSERT INTO penalty_assignments (team_id, user_id, penalty_id)
		VALUES ($1, $2, $3)
	`, tid, uid, pen.ID)
	require.NoError(t, err)

	openPens, err := repo.ListOpenPenaltiesByUser(ctx, teamID)
	require.NoError(t, err, "must not fail scanning a NULL SUM() when every row in the group has a NULL amount")
	require.Len(t, openPens, 1)
	assert.Equal(t, int64(0), openPens[0].TotalAmount)
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
	_, err := pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	created, err := repo.CreateContributions(ctx, teamID, "Mitgliedsbeitrag Juni", nil, 2500, nil, []uuid.UUID{userID})
	require.NoError(t, err)
	require.Len(t, created, 1)
	contribID := created[0].ID
	assert.Equal(t, "Mitgliedsbeitrag Juni", created[0].Name)
	assert.Equal(t, int64(2500), created[0].Amount)
	assert.Nil(t, created[0].DueDate)
	assert.Equal(t, int64(0), created[0].PaidAmount)

	list, err := repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, contribID, list[0].ID)
	assert.Equal(t, int64(0), list[0].PaidAmount)

	openCount, err := repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 1, openCount)

	// UpdateContribution: change name, amount, and due date.
	newName := "Monthly Fee"
	var newAmount int64 = 3000
	dueDate := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	updated, err := repo.UpdateContribution(ctx, contribID, teamID, finances.ContributionPatch{
		Name:    &newName,
		Amount:  &newAmount,
		DueDate: &dueDate,
	})
	require.NoError(t, err)
	assert.Equal(t, "Monthly Fee", updated.Name)
	assert.Equal(t, int64(3000), updated.Amount)
	require.NotNil(t, updated.DueDate)
	assert.True(t, dueDate.Equal(*updated.DueDate))

	// Paid amount/status are derived from linked income transactions, not a
	// settable field: booking a partial payment moves the contribution to
	// "partial", a second one covering the rest moves it to "paid".
	category := "Beiträge"
	tx1, err := repo.CreateTransaction(ctx, teamID, "income", "Teilzahlung 1", 1000, time.Now().UTC(), &category, &contribID, nil, nil)
	require.NoError(t, err)
	list, err = repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, int64(1000), list[0].PaidAmount)

	openCount, err = repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 1, openCount) // still open (partial < amount)

	_, err = repo.CreateTransaction(ctx, teamID, "income", "Teilzahlung 2", 2000, time.Now().UTC(), &category, &contribID, nil, nil)
	require.NoError(t, err)
	list, err = repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, int64(3000), list[0].PaidAmount) // 1000 + 2000 >= 3000

	openCount, err = repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 0, openCount)

	// Deleting a linked transaction reduces the derived paid amount again.
	require.NoError(t, repo.DeleteTransaction(ctx, tx1.ID, teamID))
	list, err = repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), list[0].PaidAmount)

	// Cross-team access must be rejected, including via the empty-patch no-op path.
	otherTeamID := uuid.New()

	_, err = repo.UpdateContribution(ctx, contribID, otherTeamID, finances.ContributionPatch{})
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	newerName := "Hijacked"
	_, err = repo.UpdateContribution(ctx, contribID, otherTeamID, finances.ContributionPatch{Name: &newerName})
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	err = repo.DeleteContribution(ctx, contribID, otherTeamID)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	// The contribution must be untouched by the rejected attempts.
	list, err = repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Monthly Fee", list[0].Name)

	// Deleting a contribution unlinks (not cascades to) its linked
	// transaction -- the booked income must survive.
	require.NoError(t, repo.DeleteContribution(ctx, contribID, teamID))
	list, err = repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Empty(t, list)
	txs, err := repo.ListTransactions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Nil(t, txs[0].ContributionID)
}

// TestFinancesRepository_UpdateContribution_DescriptionAndArchived verifies
// that description and archived are independently patchable, persist across
// a reload, and archiving never touches the row's other fields or any
// linked transaction.
func TestFinancesRepository_UpdateContribution_DescriptionAndArchived(t *testing.T) {
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

	created, err := repo.CreateContributions(ctx, teamID, "Turniergebühr", nil, 1500, nil, []uuid.UUID{userID})
	require.NoError(t, err)
	require.Len(t, created, 1)
	contribID := created[0].ID
	assert.Nil(t, created[0].Description)
	assert.False(t, created[0].Archived)

	desc := "Deckt Startgeld und Transport ab."
	updated, err := repo.UpdateContribution(ctx, contribID, teamID, finances.ContributionPatch{Description: &desc})
	require.NoError(t, err)
	require.NotNil(t, updated.Description)
	assert.Equal(t, desc, *updated.Description)
	assert.False(t, updated.Archived)
	// Untouched by the description-only patch.
	assert.Equal(t, "Turniergebühr", updated.Name)
	assert.Equal(t, int64(1500), updated.Amount)

	archived := true
	updated, err = repo.UpdateContribution(ctx, contribID, teamID, finances.ContributionPatch{Archived: &archived})
	require.NoError(t, err)
	assert.True(t, updated.Archived)
	require.NotNil(t, updated.Description)
	assert.Equal(t, desc, *updated.Description) // untouched by the archived-only patch

	list, err := repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].Archived)

	unarchived := false
	updated, err = repo.UpdateContribution(ctx, contribID, teamID, finances.ContributionPatch{Archived: &unarchived})
	require.NoError(t, err)
	assert.False(t, updated.Archived)
}

// TestFinancesRepository_CountOpenContributions_ExcludesArchived verifies
// that an archived contribution is excluded from the open-contributions
// count regardless of its paid state, per Contribution.archived's contract.
func TestFinancesRepository_CountOpenContributions_ExcludesArchived(t *testing.T) {
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

	created, err := repo.CreateContributions(ctx, teamID, "Mitgliedsbeitrag", nil, 2000, nil, []uuid.UUID{userID})
	require.NoError(t, err)
	contribID := created[0].ID

	openCount, err := repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 1, openCount)

	archived := true
	_, err = repo.UpdateContribution(ctx, contribID, teamID, finances.ContributionPatch{Archived: &archived})
	require.NoError(t, err)

	openCount, err = repo.CountOpenContributions(ctx, teamID)
	require.NoError(t, err)
	assert.Equal(t, 0, openCount, "archived contribution must not count as open even though it's unpaid")
}

// TestFinancesRepository_TransactionNote_RoundTrips verifies that an
// optional transaction note persists through create, list, and update.
func TestFinancesRepository_TransactionNote_RoundTrips(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	note := "Bar erhalten, Quittung Nr. 12"
	created, err := repo.CreateTransaction(ctx, teamID, "income", "Spende", 500, time.Now().UTC(), nil, nil, nil, &note)
	require.NoError(t, err)
	require.NotNil(t, created.Note)
	assert.Equal(t, note, *created.Note)

	list, err := repo.ListTransactions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].Note)
	assert.Equal(t, note, *list[0].Note)

	newNote := "Korrektur: eigentlich Quittung Nr. 13"
	updated, err := repo.UpdateTransaction(ctx, created.ID, teamID, finances.TransactionPatch{Note: &newNote})
	require.NoError(t, err)
	require.NotNil(t, updated.Note)
	assert.Equal(t, newNote, *updated.Note)
}

// TestFinancesRepository_CreateContributions_FanOutIsAtomic verifies the
// membership-fee fan-out create: one row per userID, and if any id fails
// the atomic membership re-check, the whole batch rolls back rather than
// being left partially applied.
func TestFinancesRepository_CreateContributions_FanOutIsAtomic(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid1 := uuid.New().String()
	uid2 := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid1, tid)
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Second User', 'second@example.com', '#336699')`, uid2)
	require.NoError(t, err)
	teamID := uuid.MustParse(tid)
	userID1 := uuid.MustParse(uid1)
	userID2 := uuid.MustParse(uid2)
	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid1)
	require.NoError(t, err)
	// userID2 is deliberately NOT a member of teamID.

	// Fan-out to a real member succeeds and creates exactly one row.
	created, err := repo.CreateContributions(ctx, teamID, "Turnier", nil, 1000, nil, []uuid.UUID{userID1})
	require.NoError(t, err)
	require.Len(t, created, 1)

	// A batch containing a non-member is rejected in full: the whole
	// transaction rolls back, so the member-only row from a mixed batch is
	// never left behind either.
	_, err = repo.CreateContributions(ctx, teamID, "Batch", nil, 1000, nil, []uuid.UUID{userID1, userID2})
	require.Error(t, err)
	assert.ErrorIs(t, err, finances.ErrUserNotInTeam)

	list, err := repo.ListContributions(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, list, 1, "the rejected batch must not have left a partial row behind")
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
	_, err = repo.CreateAssignment(ctx, teamID, userID, penalty.ID, time.Now(), nil)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "CreateAssignment must reject a userID that is not a member of teamID")

	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2)`, tid, uid)
	require.NoError(t, err)

	a, err := repo.CreateAssignment(ctx, teamID, userID, penalty.ID, time.Now(), nil)
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

	_, err = repo.CreateAssignment(ctx, teamID, userID, penalty.ID, time.Now(), nil)
	assert.ErrorIs(t, err, finances.ErrPenaltyNotInTeam, "CreateAssignment must map a penalty FK violation to ErrPenaltyNotInTeam")
}

// Regression test: ListTransactions ordered by date DESC, created_at DESC
// with no further tiebreaker, unlike its sibling ListTransactionsPage/
// ListAssignments, which both include id as a final tiebreaker. Two
// transactions sharing the exact same date AND created_at (bulk import,
// migration backfill, or simply coincidence at low timestamp resolution)
// left Postgres free to return them in either order across otherwise
// identical calls -- capped at maxOverviewRows, that non-determinism could
// silently swap which rows fall inside vs. outside the cap between reloads
// of the same, unchanged data.
func TestFinancesRepository_ListTransactions_DeterministicOrderOnTie(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	tiedDate := "2024-06-01"
	tiedCreatedAt := "2024-06-01T12:00:00Z"
	lowID := "aaaaaaaa-0000-0000-0000-000000000001"
	highID := "ffffffff-0000-0000-0000-000000000002"
	for _, id := range []string{lowID, highID} {
		_, err := pool.Exec(ctx, `
			INSERT INTO transactions (id, team_id, type, title, amount, date, created_at)
			VALUES ($1, $2, 'income', 'Tied', 100, $3, $4)
		`, id, tid, tiedDate, tiedCreatedAt)
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		list, err := repo.ListTransactions(ctx, teamID)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, highID, list[0].ID.String(), "call %d: id DESC must break the date/created_at tie deterministically", i)
		assert.Equal(t, lowID, list[1].ID.String(), "call %d: id DESC must break the date/created_at tie deterministically", i)
	}
}

// Same regression as above, for ListPenalties (ORDER BY label with no
// tiebreaker).
func TestFinancesRepository_ListPenalties_DeterministicOrderOnTie(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()
	seedFinanceFixtures(t, pool, uid, tid)
	teamID := uuid.MustParse(tid)

	lowID := "aaaaaaaa-0000-0000-0000-000000000001"
	highID := "ffffffff-0000-0000-0000-000000000002"
	for _, id := range []string{lowID, highID} {
		_, err := pool.Exec(ctx, `INSERT INTO penalties (id, team_id, label, amount) VALUES ($1, $2, 'Same Label', 500)`, id, tid)
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		list, err := repo.ListPenalties(ctx, teamID)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, lowID, list[0].ID.String(), "call %d: id must break the label tie deterministically", i)
		assert.Equal(t, highID, list[1].ID.String(), "call %d: id must break the label tie deterministically", i)
	}
}

// Same regression as above, for ListContributions (ORDER BY due_date,
// contribution name, user name with no tiebreaker). Two different users
// sharing the same name (a real possibility, not something the app
// prevents) each with a same-named, same-due-date fee produces the tie.
func TestFinancesRepository_ListContributions_DeterministicOrderOnTie(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := finances.NewRepository(pool)
	ctx := context.Background()

	tid := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Tie Team')`, tid)
	require.NoError(t, err)

	user1 := "11111111-0000-0000-0000-000000000001"
	user2 := "22222222-0000-0000-0000-000000000002"
	for _, uid := range []string{user1, user2} {
		_, err := pool.Exec(ctx,
			`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Same Name', $2, '#123456')`,
			uid, uid+"@example.com")
		require.NoError(t, err)
	}

	lowID := "aaaaaaaa-0000-0000-0000-000000000001"
	highID := "ffffffff-0000-0000-0000-000000000002"
	_, err = pool.Exec(ctx, `INSERT INTO contributions (id, team_id, user_id, name, amount) VALUES ($1, $2, $3, 'Beitrag', 2500)`, highID, tid, user1)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO contributions (id, team_id, user_id, name, amount) VALUES ($1, $2, $3, 'Beitrag', 2500)`, lowID, tid, user2)
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		list, err := repo.ListContributions(ctx, uuid.MustParse(tid))
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, lowID, list[0].ID.String(), "call %d: id must break the (dueDate, name) tie deterministically", i)
		assert.Equal(t, highID, list[1].ID.String(), "call %d: id must break the (dueDate, name) tie deterministically", i)
	}
}
