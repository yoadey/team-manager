package finances_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// ─── mock repository ────────────────────────────────────────────────────────

// mockRepo satisfies the unexported financeRepo interface via structural typing.
type mockRepo struct {
	listTransactionsFn       func(ctx context.Context, teamID uuid.UUID) ([]finances.TransactionRow, error)
	listTransactionsPageFn   func(ctx context.Context, teamID uuid.UUID, limit int, cur *finances.TxCursor) ([]finances.TransactionRow, error)
	sumTransactionsFn        func(ctx context.Context, teamID uuid.UUID) (int64, int64, error)
	createTransactionFn      func(ctx context.Context, teamID uuid.UUID, txType, title string, amount int64, date time.Time, category *string) (*finances.TransactionRow, error)
	updateTransactionFn      func(ctx context.Context, id, teamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error)
	deleteTransactionFn      func(ctx context.Context, id, teamID uuid.UUID) error
	listPenaltiesFn          func(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyRow, error)
	countPenaltiesFn         func(ctx context.Context, teamID uuid.UUID) (int, error)
	createPenaltyFn          func(ctx context.Context, teamID uuid.UUID, label string, amount int64) (*finances.PenaltyRow, error)
	updatePenaltyFn          func(ctx context.Context, id, teamID uuid.UUID, patch finances.PenaltyPatch) (*finances.PenaltyRow, error)
	deletePenaltyFn          func(ctx context.Context, id, teamID uuid.UUID) error
	penaltyBelongsToTeamFn   func(ctx context.Context, penaltyID, teamID uuid.UUID) (bool, error)
	listAssignmentsFn        func(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyAssignmentRow, error)
	getAssignmentByIDFn      func(ctx context.Context, id, teamID uuid.UUID) (*finances.PenaltyAssignmentRow, error)
	createAssignmentFn       func(ctx context.Context, teamID, userID, penaltyID uuid.UUID) (*finances.PenaltyAssignmentRow, error)
	deleteAssignmentFn       func(ctx context.Context, id, teamID uuid.UUID) error
	setAssignmentPaidFn      func(ctx context.Context, id, teamID uuid.UUID, paid bool) (*finances.PenaltyAssignmentRow, error)
	userIsMemberOfTeamFn     func(ctx context.Context, userID, teamID uuid.UUID) (bool, error)
	listContributionsFn      func(ctx context.Context, teamID uuid.UUID) ([]finances.ContributionRow, error)
	countOpenContributionsFn func(ctx context.Context, teamID uuid.UUID) (int, error)
	updateContributionFn     func(ctx context.Context, id, teamID uuid.UUID, patch finances.ContributionPatch) (*finances.ContributionRow, error)
	setContributionPaidFn    func(ctx context.Context, id, teamID uuid.UUID, paid bool) (*finances.ContributionRow, error)
	listOpenPenaltiesFn      func(ctx context.Context, teamID uuid.UUID) ([]finances.OpenPenaltyAggregate, error)
	withReadTxFn             func(ctx context.Context, fn func(finances.OverviewReader) error) error
	countTransactionsFn      func(ctx context.Context, teamID uuid.UUID) (int, error)
	countAssignmentsFn       func(ctx context.Context, teamID uuid.UUID) (int, error)
}

func (m *mockRepo) ListTransactions(ctx context.Context, teamID uuid.UUID) ([]finances.TransactionRow, error) {
	return m.listTransactionsFn(ctx, teamID)
}

// ListTransactionsPage is optional; when unset it returns no rows, so tests
// that don't exercise pagination don't all need to set it.
func (m *mockRepo) ListTransactionsPage(ctx context.Context, teamID uuid.UUID, limit int, cur *finances.TxCursor) ([]finances.TransactionRow, error) {
	if m.listTransactionsPageFn != nil {
		return m.listTransactionsPageFn(ctx, teamID, limit, cur)
	}
	return nil, nil
}

func (m *mockRepo) SumTransactions(ctx context.Context, teamID uuid.UUID) (income, expense int64, err error) {
	return m.sumTransactionsFn(ctx, teamID)
}

// CountTransactions is optional; when unset, existing tests exercising
// CreateTransaction get a default of 0 (well under maxTransactionsPerTeam)
// so they don't all need updating just to set this new field.
func (m *mockRepo) CountTransactions(ctx context.Context, teamID uuid.UUID) (int, error) {
	if m.countTransactionsFn != nil {
		return m.countTransactionsFn(ctx, teamID)
	}
	return 0, nil
}

func (m *mockRepo) CreateTransaction(ctx context.Context, teamID uuid.UUID, txType, title string, amount int64, date time.Time, category *string) (*finances.TransactionRow, error) {
	return m.createTransactionFn(ctx, teamID, txType, title, amount, date, category)
}

func (m *mockRepo) UpdateTransaction(ctx context.Context, id, teamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error) {
	return m.updateTransactionFn(ctx, id, teamID, patch)
}

func (m *mockRepo) DeleteTransaction(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteTransactionFn(ctx, id, teamID)
}

func (m *mockRepo) ListPenalties(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyRow, error) {
	return m.listPenaltiesFn(ctx, teamID)
}

// CountPenalties is optional; when unset, existing tests exercising
// CreatePenalty get a default of 0 (well under maxPenaltiesPerTeam) so they
// don't all need updating just to set this new field.
func (m *mockRepo) CountPenalties(ctx context.Context, teamID uuid.UUID) (int, error) {
	if m.countPenaltiesFn != nil {
		return m.countPenaltiesFn(ctx, teamID)
	}
	return 0, nil
}

func (m *mockRepo) CreatePenalty(ctx context.Context, teamID uuid.UUID, label string, amount int64) (*finances.PenaltyRow, error) {
	return m.createPenaltyFn(ctx, teamID, label, amount)
}

func (m *mockRepo) UpdatePenalty(ctx context.Context, id, teamID uuid.UUID, patch finances.PenaltyPatch) (*finances.PenaltyRow, error) {
	return m.updatePenaltyFn(ctx, id, teamID, patch)
}

func (m *mockRepo) DeletePenalty(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deletePenaltyFn(ctx, id, teamID)
}

func (m *mockRepo) PenaltyBelongsToTeam(ctx context.Context, penaltyID, teamID uuid.UUID) (bool, error) {
	return m.penaltyBelongsToTeamFn(ctx, penaltyID, teamID)
}

func (m *mockRepo) ListAssignments(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyAssignmentRow, error) {
	return m.listAssignmentsFn(ctx, teamID)
}

func (m *mockRepo) GetAssignmentByID(ctx context.Context, id, teamID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
	return m.getAssignmentByIDFn(ctx, id, teamID)
}

// CountAssignments is optional; when unset, existing tests exercising
// CreateAssignment get a default of 0 (well under maxAssignmentsPerTeam) so
// they don't all need updating just to set this new field.
func (m *mockRepo) CountAssignments(ctx context.Context, teamID uuid.UUID) (int, error) {
	if m.countAssignmentsFn != nil {
		return m.countAssignmentsFn(ctx, teamID)
	}
	return 0, nil
}

func (m *mockRepo) CreateAssignment(ctx context.Context, teamID, userID, penaltyID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
	return m.createAssignmentFn(ctx, teamID, userID, penaltyID)
}

func (m *mockRepo) DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteAssignmentFn(ctx, id, teamID)
}

func (m *mockRepo) SetAssignmentPaid(ctx context.Context, id, teamID uuid.UUID, paid bool) (*finances.PenaltyAssignmentRow, error) {
	return m.setAssignmentPaidFn(ctx, id, teamID, paid)
}

func (m *mockRepo) UserIsMemberOfTeam(ctx context.Context, userID, teamID uuid.UUID) (bool, error) {
	return m.userIsMemberOfTeamFn(ctx, userID, teamID)
}

func (m *mockRepo) ListContributions(ctx context.Context, teamID uuid.UUID) ([]finances.ContributionRow, error) {
	return m.listContributionsFn(ctx, teamID)
}

func (m *mockRepo) CountOpenContributions(ctx context.Context, teamID uuid.UUID) (int, error) {
	return m.countOpenContributionsFn(ctx, teamID)
}

func (m *mockRepo) UpdateContribution(ctx context.Context, id, teamID uuid.UUID, patch finances.ContributionPatch) (*finances.ContributionRow, error) {
	return m.updateContributionFn(ctx, id, teamID, patch)
}

func (m *mockRepo) SetContributionPaid(ctx context.Context, id, teamID uuid.UUID, paid bool) (*finances.ContributionRow, error) {
	return m.setContributionPaidFn(ctx, id, teamID, paid)
}

func (m *mockRepo) ListOpenPenaltiesByUser(ctx context.Context, teamID uuid.UUID) ([]finances.OpenPenaltyAggregate, error) {
	return m.listOpenPenaltiesFn(ctx, teamID)
}

// WithReadTx runs fn directly against the mock itself (which already
// implements finances.OverviewReader), since unit tests have no live
// transaction to hand out.
func (m *mockRepo) WithReadTx(ctx context.Context, fn func(finances.OverviewReader) error) error {
	if m.withReadTxFn != nil {
		return m.withReadTxFn(ctx, fn)
	}
	return fn(m)
}

// ─── GetOverview ─────────────────────────────────────────────────────────────

func TestService_GetOverview_ComputesBalanceAndOpenPenaltySum(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		listTransactionsFn:       func(context.Context, uuid.UUID) ([]finances.TransactionRow, error) { return nil, nil },
		sumTransactionsFn:        func(context.Context, uuid.UUID) (int64, int64, error) { return 50000, 20000, nil },
		listPenaltiesFn:          func(context.Context, uuid.UUID) ([]finances.PenaltyRow, error) { return nil, nil },
		listAssignmentsFn:        func(context.Context, uuid.UUID) ([]finances.PenaltyAssignmentRow, error) { return nil, nil },
		listContributionsFn:      func(context.Context, uuid.UUID) ([]finances.ContributionRow, error) { return nil, nil },
		countOpenContributionsFn: func(context.Context, uuid.UUID) (int, error) { return 3, nil },
		listOpenPenaltiesFn: func(context.Context, uuid.UUID) ([]finances.OpenPenaltyAggregate, error) {
			return []finances.OpenPenaltyAggregate{
				{UserID: uuid.New(), TotalAmount: 1500},
				{UserID: uuid.New(), TotalAmount: 550},
			}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	overview, err := svc.GetOverview(context.Background(), teamID)
	require.NoError(t, err)
	assert.Equal(t, int64(50000), overview.Income)
	assert.Equal(t, int64(20000), overview.Expense)
	assert.Equal(t, int64(30000), overview.Balance, "balance must be income - expense")
	assert.Equal(t, 3, overview.ContribOpen)
	assert.Equal(t, int64(2050), overview.OpenPenaltySum, "open penalty sum must total all users' open amounts")
	assert.Len(t, overview.OpenPenalties, 2)
}

func TestService_GetOverview_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		listTransactionsFn: func(context.Context, uuid.UUID) ([]finances.TransactionRow, error) {
			return nil, wantErr
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	_, err := svc.GetOverview(context.Background(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

// ─── Transactions ────────────────────────────────────────────────────────────

func TestService_CreateTransaction(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	category := "equipment"
	var capturedAmount int64
	repo := &mockRepo{
		createTransactionFn: func(_ context.Context, gotTeamID uuid.UUID, txType, title string, amount int64, _ time.Time, gotCategory *string) (*finances.TransactionRow, error) {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, "expense", txType)
			assert.Equal(t, &category, gotCategory)
			capturedAmount = amount
			return &finances.TransactionRow{ID: uuid.New(), TeamID: teamID, Type: txType, Title: title, Amount: amount, Category: gotCategory}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateTransactionJSONRequestBody{
		Type:     gen.Expense,
		Title:    "Balls",
		Amount:   4250,
		Category: &category,
	}
	result, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(4250), capturedAmount)
	assert.Equal(t, "Balls", result.Title)
}

func TestService_CreateTransaction_UsesClientDateWhenProvided(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	want := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	var gotDate time.Time
	repo := &mockRepo{
		createTransactionFn: func(_ context.Context, _ uuid.UUID, txType, title string, amount int64, date time.Time, cat *string) (*finances.TransactionRow, error) {
			gotDate = date
			return &finances.TransactionRow{ID: uuid.New(), TeamID: teamID, Type: txType, Title: title, Amount: amount, Date: date, Category: cat}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateTransactionJSONRequestBody{
		Type:   gen.Income,
		Title:  "Back-dated dues",
		Amount: 1000,
		Date:   &openapi_types.Date{Time: want},
	}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.NoError(t, err)
	assert.Equal(t, want, gotDate, "client-provided date must be passed through to the repository")
}

func TestService_UpdateTransaction_PassesDatePatch(t *testing.T) {
	t.Parallel()

	want := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	var gotPatch finances.TransactionPatch
	repo := &mockRepo{
		updateTransactionFn: func(_ context.Context, id, teamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error) {
			gotPatch = patch
			return &finances.TransactionRow{ID: id, TeamID: teamID, Type: "income", Title: "x", Amount: 1, Date: want}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.UpdateTransactionJSONRequestBody{Date: &openapi_types.Date{Time: want}}
	_, err := svc.UpdateTransaction(context.Background(), uuid.New(), uuid.New(), body)
	require.NoError(t, err)
	require.NotNil(t, gotPatch.Date)
	assert.Equal(t, want, *gotPatch.Date)
}

// ─── ListTransactions (keyset pagination) ────────────────────────────────────

func TestService_ListTransactions_ReturnsNextCursorWhenMorePages(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	// Repo is asked for limit+1 to detect a further page; return exactly that
	// many so the service trims to `limit` and emits a next cursor.
	var gotLimit int
	repo := &mockRepo{
		listTransactionsPageFn: func(_ context.Context, _ uuid.UUID, limit int, cur *finances.TxCursor) ([]finances.TransactionRow, error) {
			gotLimit = limit
			assert.Nil(t, cur, "first page must decode to a nil cursor")
			rows := make([]finances.TransactionRow, limit)
			for i := range rows {
				rows[i] = finances.TransactionRow{ID: uuid.New(), TeamID: teamID, Type: "income", Title: "t", Amount: 1, Date: time.Now()}
			}
			return rows, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	items, next, err := svc.ListTransactions(context.Background(), teamID, 2, "")
	require.NoError(t, err)
	assert.Equal(t, 3, gotLimit, "service must over-fetch by one to detect a further page")
	assert.Len(t, items, 2, "the page must be trimmed back to the requested limit")
	require.NotNil(t, next, "a next cursor must be returned when a further page exists")
	assert.NotEmpty(t, *next)
}

func TestService_ListTransactions_NoCursorOnLastPage(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		listTransactionsPageFn: func(_ context.Context, _ uuid.UUID, _ int, _ *finances.TxCursor) ([]finances.TransactionRow, error) {
			return []finances.TransactionRow{
				{ID: uuid.New(), Type: "income", Title: "t", Amount: 1, Date: time.Now()},
			}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	items, next, err := svc.ListTransactions(context.Background(), uuid.New(), 50, "")
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Nil(t, next, "no next cursor when the last page fits under the limit")
}

func TestService_ListTransactions_DecodesIncomingCursor(t *testing.T) {
	t.Parallel()

	pager := pagination.New(nil)
	want := finances.TxCursor{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now().UTC().Truncate(time.Second), ID: uuid.New()}
	token, err := pager.Encode(want)
	require.NoError(t, err)

	var got *finances.TxCursor
	repo := &mockRepo{
		listTransactionsPageFn: func(_ context.Context, _ uuid.UUID, _ int, cur *finances.TxCursor) ([]finances.TransactionRow, error) {
			got = cur
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pager, slog.Default())
	_, _, err = svc.ListTransactions(context.Background(), uuid.New(), 50, token)
	require.NoError(t, err)
	require.NotNil(t, got, "a valid cursor token must be decoded and forwarded to the repository")
	assert.Equal(t, want.ID, got.ID)
	assert.True(t, want.Date.Equal(got.Date))
}

// Regression test: with no per-team cap, a member holding only
// finances:write could flood the transactions table past what the
// unbounded aggregate queries behind the finance overview (SumTransactions)
// can scan within their fixed 5s timeout, degrading or hard-failing the
// overview for the whole team. CreateTransaction must refuse once the team
// is at maxTransactionsPerTeam, without ever reaching the repo's insert.
func TestService_CreateTransaction_RejectsAtCap(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		countTransactionsFn: func(context.Context, uuid.UUID) (int, error) { return 100_000, nil },
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string) (*finances.TransactionRow, error) {
			t.Fatal("CreateTransaction must not be called once the team is at the transaction cap")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Expense, Title: "Balls", Amount: 100}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrTooManyTransactions)
}

// Regression test: unlike CreateTransaction/CreateAssignment, CreatePenalty
// used to have no CountPenalties check at all -- a team member with
// finances:write could flood the penalties table without bound, and
// GetOverview reads ListPenalties unconditionally inside the same 5s query
// timeout as every other overview list. CreatePenalty must refuse once the
// team is at maxPenaltiesPerTeam, without ever reaching the repo's insert.
func TestService_CreatePenalty_RejectsAtCap(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		countPenaltiesFn: func(context.Context, uuid.UUID) (int, error) { return 500, nil },
		createPenaltyFn: func(context.Context, uuid.UUID, string, int64) (*finances.PenaltyRow, error) {
			t.Fatal("CreatePenalty must not be called once the team is at the penalty cap")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyJSONRequestBody{Label: "Zu spät", Amount: 500}
	_, err := svc.CreatePenalty(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrTooManyPenalties)
}

func TestService_UpdateTransaction_OnlySetsProvidedFields(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	newTitle := "Renamed"
	var capturedPatch finances.TransactionPatch
	repo := &mockRepo{
		updateTransactionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error) {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			capturedPatch = patch
			return &finances.TransactionRow{ID: id, TeamID: teamID, Title: newTitle}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.UpdateTransactionJSONRequestBody{Title: &newTitle}
	_, err := svc.UpdateTransaction(context.Background(), id, teamID, body)
	require.NoError(t, err)

	require.NotNil(t, capturedPatch.Title)
	assert.Equal(t, newTitle, *capturedPatch.Title)
	assert.Nil(t, capturedPatch.Amount, "amount should stay nil when not provided in the request body")
	assert.Nil(t, capturedPatch.Type)
	assert.Nil(t, capturedPatch.Category)
}

func TestService_DeleteTransaction(t *testing.T) {
	t.Parallel()

	called := false
	repo := &mockRepo{
		deleteTransactionFn: func(context.Context, uuid.UUID, uuid.UUID) error {
			called = true
			return nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	err := svc.DeleteTransaction(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.True(t, called)
}

// ─── Assignments ─────────────────────────────────────────────────────────────

func TestService_CreateAssignment_RejectsPenaltyFromAnotherTeam(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil },
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: uuid.New(), UserId: uuid.New()}
	_, err := svc.CreateAssignment(context.Background(), uuid.New(), body)
	require.ErrorIs(t, err, finances.ErrPenaltyNotInTeam)
}

func TestService_CreateAssignment_RejectsUserNotInTeam(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil },
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: uuid.New(), UserId: uuid.New()}
	_, err := svc.CreateAssignment(context.Background(), uuid.New(), body)
	require.ErrorIs(t, err, finances.ErrUserNotInTeam)
}

// Regression test: same unbounded-growth risk as
// TestService_CreateTransaction_RejectsAtCap, but for penalty_assignments,
// which ListOpenPenaltiesByUser scans on every finance overview. The cap
// check must run after the existing penalty/user validation (so those still
// report their own specific errors) but before the insert.
func TestService_CreateAssignment_RejectsAtCap(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		countAssignmentsFn:     func(context.Context, uuid.UUID) (int, error) { return 100_000, nil },
		createAssignmentFn: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			t.Fatal("CreateAssignment must not be called once the team is at the assignment cap")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	_, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrTooManyAssignments)
}

func TestService_CreateAssignment_ReloadsEnrichedRowOnSuccess(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	createdID := uuid.New()
	label := "Late arrival"
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(_ context.Context, gotTeamID, gotUserID, gotPenaltyID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, userID, gotUserID)
			assert.Equal(t, penaltyID, gotPenaltyID)
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID}, nil
		},
		getAssignmentByIDFn: func(_ context.Context, gotID, gotTeamID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			assert.Equal(t, createdID, gotID)
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, PenaltyLabel: &label}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	result, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err)
	require.NotNil(t, result.Label)
	assert.Equal(t, label, *result.Label, "result should use the enriched row from GetAssignmentByID, not the bare insert result")
}

func TestService_CreateAssignment_FallsBackToUnenrichedRowWhenReloadFails(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	createdID := uuid.New()
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return nil, errors.New("reload failed")
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	result, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err, "a reload failure after a successful create should not fail the request")
	assert.Equal(t, createdID, result.Id)
	assert.Nil(t, result.Label, "unenriched fallback row has no penalty label")
}

// Regression test: a concurrent DeletePenalty that cascades the just-created
// assignment away between the insert and the reload used to be
// indistinguishable from a merely transient reload failure -- both fell
// through to the same "return the bare, unenriched row with a 200 OK"
// fallback, silently reporting success for a row that no longer exists in
// the database. pgx.ErrNoRows specifically must propagate instead, so the
// handler's existing "not found" mapping applies.
func TestService_CreateAssignment_PropagatesErrNoRowsWhenRowDeletedBeforeReload(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	createdID := uuid.New()
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	_, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.ErrorIs(t, err, pgx.ErrNoRows, "must not silently return a 200 OK for a row deleted before the reload")
}

func TestService_SetPenaltyPaid_ReloadsEnrichedRow(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	label := "Yellow card"
	repo := &mockRepo{
		setAssignmentPaidFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, paid bool) (*finances.PenaltyAssignmentRow, error) {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			assert.True(t, paid, "the requested paid value must be passed through")
			return &finances.PenaltyAssignmentRow{ID: id, TeamID: teamID, Paid: true}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: id, TeamID: teamID, Paid: true, PenaltyLabel: &label}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	result, err := svc.SetPenaltyPaid(context.Background(), teamID, id, true)
	require.NoError(t, err)
	assert.True(t, result.Paid)
	require.NotNil(t, result.Label)
	assert.Equal(t, label, *result.Label)
}

// Regression test: same class as
// TestService_CreateAssignment_PropagatesErrNoRowsWhenRowDeletedBeforeReload,
// for the toggle-paid path.
func TestService_SetPenaltyPaid_PropagatesErrNoRowsWhenRowDeletedBeforeReload(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	repo := &mockRepo{
		setAssignmentPaidFn: func(context.Context, uuid.UUID, uuid.UUID, bool) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: id, TeamID: teamID, Paid: true}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	_, err := svc.SetPenaltyPaid(context.Background(), teamID, id, true)
	require.ErrorIs(t, err, pgx.ErrNoRows, "must not silently return a 200 OK for a row deleted before the reload")
}

// ─── Contributions ───────────────────────────────────────────────────────────

func TestService_SetContributionPaid(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	repo := &mockRepo{
		setContributionPaidFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, paid bool) (*finances.ContributionRow, error) {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			assert.True(t, paid, "the requested paid value must be passed through")
			return &finances.ContributionRow{ID: id, TeamID: teamID, Status: "paid"}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	result, err := svc.SetContributionPaid(context.Background(), id, teamID, true)
	require.NoError(t, err)
	assert.Equal(t, gen.ContributionStatus("paid"), result.Status)
}
