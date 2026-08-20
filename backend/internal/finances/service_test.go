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
	listTransactionsFn               func(ctx context.Context, teamID uuid.UUID) ([]finances.TransactionRow, error)
	listTransactionsPageFn           func(ctx context.Context, teamID uuid.UUID, limit int, cur *finances.TxCursor) ([]finances.TransactionRow, error)
	sumTransactionsFn                func(ctx context.Context, teamID uuid.UUID) (int64, int64, error)
	createTransactionFn              func(ctx context.Context, teamID uuid.UUID, txType, title string, amount int64, date time.Time, category *string, contributionID, penaltyAssignmentID *uuid.UUID, note *string) (*finances.TransactionRow, error)
	getTransactionFn                 func(ctx context.Context, id, teamID uuid.UUID) (*finances.TransactionRow, error)
	updateTransactionFn              func(ctx context.Context, id, teamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error)
	deleteTransactionFn              func(ctx context.Context, id, teamID uuid.UUID) error
	listPenaltiesFn                  func(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyRow, error)
	countPenaltiesFn                 func(ctx context.Context, teamID uuid.UUID) (int, error)
	createPenaltyFn                  func(ctx context.Context, teamID uuid.UUID, label string, amount int64) (*finances.PenaltyRow, error)
	updatePenaltyFn                  func(ctx context.Context, id, teamID uuid.UUID, patch finances.PenaltyPatch) (*finances.PenaltyRow, error)
	deletePenaltyFn                  func(ctx context.Context, id, teamID uuid.UUID) error
	penaltyBelongsToTeamFn           func(ctx context.Context, penaltyID, teamID uuid.UUID) (bool, error)
	listAssignmentsFn                func(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyAssignmentRow, error)
	getAssignmentByIDFn              func(ctx context.Context, id, teamID uuid.UUID) (*finances.PenaltyAssignmentRow, error)
	createAssignmentFn               func(ctx context.Context, teamID, userID, penaltyID uuid.UUID, date time.Time, note *string) (*finances.PenaltyAssignmentRow, error)
	deleteAssignmentFn               func(ctx context.Context, id, teamID uuid.UUID) error
	penaltyAssignmentBelongsToTeamFn func(ctx context.Context, assignmentID, teamID uuid.UUID) (bool, error)
	userIsMemberOfTeamFn             func(ctx context.Context, userID, teamID uuid.UUID) (bool, error)
	listContributionsFn              func(ctx context.Context, teamID uuid.UUID) ([]finances.ContributionRow, error)
	countOpenContributionsFn         func(ctx context.Context, teamID uuid.UUID) (int, error)
	createContributionsFn            func(ctx context.Context, teamID uuid.UUID, name string, description *string, amount int64, dueDate *time.Time, userIDs []uuid.UUID) ([]finances.ContributionRow, error)
	updateContributionFn             func(ctx context.Context, id, teamID uuid.UUID, patch finances.ContributionPatch) (*finances.ContributionRow, error)
	deleteContributionFn             func(ctx context.Context, id, teamID uuid.UUID) error
	contributionBelongsToTeamFn      func(ctx context.Context, contributionID, teamID uuid.UUID) (bool, error)
	listOpenPenaltiesFn              func(ctx context.Context, teamID uuid.UUID) ([]finances.OpenPenaltyAggregate, error)
	withReadTxFn                     func(ctx context.Context, fn func(finances.OverviewReader) error) error
	countTransactionsFn              func(ctx context.Context, teamID uuid.UUID) (int, error)
	countAssignmentsFn               func(ctx context.Context, teamID uuid.UUID) (int, error)
	countContributionsFn             func(ctx context.Context, teamID uuid.UUID) (int, error)
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

func (m *mockRepo) CreateTransaction(ctx context.Context, teamID uuid.UUID, txType, title string, amount int64, date time.Time, category *string, contributionID, penaltyAssignmentID *uuid.UUID, note *string) (*finances.TransactionRow, error) {
	return m.createTransactionFn(ctx, teamID, txType, title, amount, date, category, contributionID, penaltyAssignmentID, note)
}

// GetTransaction is optional; when unset it returns an empty (unlinked)
// TransactionRow, so tests that don't exercise the linked-type-change guard
// don't all need to set it.
func (m *mockRepo) GetTransaction(ctx context.Context, id, teamID uuid.UUID) (*finances.TransactionRow, error) {
	if m.getTransactionFn != nil {
		return m.getTransactionFn(ctx, id, teamID)
	}
	return &finances.TransactionRow{ID: id, TeamID: teamID}, nil
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

func (m *mockRepo) CreateAssignment(ctx context.Context, teamID, userID, penaltyID uuid.UUID, date time.Time, note *string) (*finances.PenaltyAssignmentRow, error) {
	return m.createAssignmentFn(ctx, teamID, userID, penaltyID, date, note)
}

func (m *mockRepo) DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteAssignmentFn(ctx, id, teamID)
}

func (m *mockRepo) PenaltyAssignmentBelongsToTeam(ctx context.Context, assignmentID, teamID uuid.UUID) (bool, error) {
	return m.penaltyAssignmentBelongsToTeamFn(ctx, assignmentID, teamID)
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

// CountContributions is optional; when unset, existing tests exercising
// CreateContributions get a default of 0 (well under maxContributionsPerTeam)
// so they don't all need updating just to set this new field.
func (m *mockRepo) CountContributions(ctx context.Context, teamID uuid.UUID) (int, error) {
	if m.countContributionsFn != nil {
		return m.countContributionsFn(ctx, teamID)
	}
	return 0, nil
}

func (m *mockRepo) CreateContributions(ctx context.Context, teamID uuid.UUID, name string, description *string, amount int64, dueDate *time.Time, userIDs []uuid.UUID) ([]finances.ContributionRow, error) {
	return m.createContributionsFn(ctx, teamID, name, description, amount, dueDate, userIDs)
}

func (m *mockRepo) UpdateContribution(ctx context.Context, id, teamID uuid.UUID, patch finances.ContributionPatch) (*finances.ContributionRow, error) {
	return m.updateContributionFn(ctx, id, teamID, patch)
}

func (m *mockRepo) DeleteContribution(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteContributionFn(ctx, id, teamID)
}

func (m *mockRepo) ContributionBelongsToTeam(ctx context.Context, contributionID, teamID uuid.UUID) (bool, error) {
	return m.contributionBelongsToTeamFn(ctx, contributionID, teamID)
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
		createTransactionFn: func(_ context.Context, gotTeamID uuid.UUID, txType, title string, amount int64, _ time.Time, gotCategory *string, _, _ *uuid.UUID, _ *string) (*finances.TransactionRow, error) {
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
		createTransactionFn: func(_ context.Context, _ uuid.UUID, txType, title string, amount int64, date time.Time, cat *string, _, _ *uuid.UUID, _ *string) (*finances.TransactionRow, error) {
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

func TestService_UpdateTransaction_PassesNotePatch(t *testing.T) {
	t.Parallel()

	want := "Bar erhalten, Quittung Nr. 12"
	var gotPatch finances.TransactionPatch
	repo := &mockRepo{
		updateTransactionFn: func(_ context.Context, id, teamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error) {
			gotPatch = patch
			return &finances.TransactionRow{ID: id, TeamID: teamID, Type: "income", Title: "x", Amount: 1, Note: patch.Note}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.UpdateTransactionJSONRequestBody{Note: &want}
	result, err := svc.UpdateTransaction(context.Background(), uuid.New(), uuid.New(), body)
	require.NoError(t, err)
	require.NotNil(t, gotPatch.Note)
	assert.Equal(t, want, *gotPatch.Note)
	require.NotNil(t, result.Note)
	assert.Equal(t, want, *result.Note)
}

// Regression test: UpdateTransaction must reject moving a transaction's type
// away from income while it is still linked to a contribution -- otherwise
// the treasurer could silently detach a booked fee payment from its
// contribution by editing the type, with no error or audit trail (see
// ErrCannotChangeTypeOfLinkedTransaction's doc comment). The repository's
// UpdateTransaction must never be reached in this case.
func TestService_UpdateTransaction_RejectsTypeChangeWhenLinkedToContribution(t *testing.T) {
	t.Parallel()

	teamID, id, contributionID := uuid.New(), uuid.New(), uuid.New()
	repo := &mockRepo{
		getTransactionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID) (*finances.TransactionRow, error) {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			return &finances.TransactionRow{ID: id, TeamID: teamID, Type: "income", ContributionID: &contributionID}, nil
		},
		updateTransactionFn: func(context.Context, uuid.UUID, uuid.UUID, finances.TransactionPatch) (*finances.TransactionRow, error) {
			t.Fatal("repository UpdateTransaction must not be called when the type change is rejected")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	expenseType := gen.Expense
	body := &gen.UpdateTransactionJSONRequestBody{Type: &expenseType}
	_, err := svc.UpdateTransaction(context.Background(), id, teamID, body)
	require.ErrorIs(t, err, finances.ErrCannotChangeTypeOfLinkedTransaction)
}

// Same guard, for a transaction linked to a penalty assignment instead of a
// contribution.
func TestService_UpdateTransaction_RejectsTypeChangeWhenLinkedToPenaltyAssignment(t *testing.T) {
	t.Parallel()

	teamID, id, assignmentID := uuid.New(), uuid.New(), uuid.New()
	repo := &mockRepo{
		getTransactionFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.TransactionRow, error) {
			return &finances.TransactionRow{ID: id, TeamID: teamID, Type: "income", PenaltyAssignmentID: &assignmentID}, nil
		},
		updateTransactionFn: func(context.Context, uuid.UUID, uuid.UUID, finances.TransactionPatch) (*finances.TransactionRow, error) {
			t.Fatal("repository UpdateTransaction must not be called when the type change is rejected")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	expenseType := gen.Expense
	body := &gen.UpdateTransactionJSONRequestBody{Type: &expenseType}
	_, err := svc.UpdateTransaction(context.Background(), id, teamID, body)
	require.ErrorIs(t, err, finances.ErrCannotChangeTypeOfLinkedTransaction)
}

// Regression guard for the fix above: changing type on an UNLINKED
// transaction must keep working exactly as before.
func TestService_UpdateTransaction_AllowsTypeChangeWhenUnlinked(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	var updateCalled bool
	repo := &mockRepo{
		getTransactionFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.TransactionRow, error) {
			return &finances.TransactionRow{ID: id, TeamID: teamID, Type: "income"}, nil
		},
		updateTransactionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error) {
			updateCalled = true
			require.NotNil(t, patch.Type)
			assert.Equal(t, "expense", *patch.Type)
			return &finances.TransactionRow{ID: gotID, TeamID: gotTeamID, Type: *patch.Type}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	expenseType := gen.Expense
	body := &gen.UpdateTransactionJSONRequestBody{Type: &expenseType}
	_, err := svc.UpdateTransaction(context.Background(), id, teamID, body)
	require.NoError(t, err)
	assert.True(t, updateCalled, "repository UpdateTransaction must be called when the transaction isn't linked")
}

// Regression guard: changing only the amount (not type) of a transaction
// still linked to a contribution must remain allowed -- the finding this
// fix addresses is specifically about type changes silently detaching a
// linked payment; correcting a typo'd amount on an income transaction that
// stays linked is legitimate and must not be blocked.
func TestService_UpdateTransaction_AllowsAmountChangeWhenLinked(t *testing.T) {
	t.Parallel()

	teamID, id, contributionID := uuid.New(), uuid.New(), uuid.New()
	newAmount := int64(2500)
	var updateCalled bool
	repo := &mockRepo{
		getTransactionFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.TransactionRow, error) {
			t.Fatal("GetTransaction must not be called when the patch doesn't touch type")
			return nil, nil
		},
		updateTransactionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error) {
			updateCalled = true
			require.NotNil(t, patch.Amount)
			assert.Equal(t, newAmount, *patch.Amount)
			return &finances.TransactionRow{ID: gotID, TeamID: gotTeamID, Type: "income", Amount: newAmount, ContributionID: &contributionID}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.UpdateTransactionJSONRequestBody{Amount: &newAmount}
	_, err := svc.UpdateTransaction(context.Background(), id, teamID, body)
	require.NoError(t, err)
	assert.True(t, updateCalled, "repository UpdateTransaction must be called for an amount-only patch on a linked transaction")
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
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string, *uuid.UUID, *uuid.UUID, *string) (*finances.TransactionRow, error) {
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
		createAssignmentFn: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, *string) (*finances.PenaltyAssignmentRow, error) {
			t.Fatal("CreateAssignment must not be called once the team is at the assignment cap")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	_, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrTooManyAssignments)
}

// TestService_CreateAssignment_DefaultsDateToToday verifies that omitting
// body.Date defaults to the current time (mirroring CreateTransaction's
// equivalent default), rather than leaving it zero-valued.
func TestService_CreateAssignment_DefaultsDateToToday(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	before := time.Now()
	var gotDate time.Time
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(_ context.Context, _, _, _ uuid.UUID, date time.Time, _ *string) (*finances.PenaltyAssignmentRow, error) {
			gotDate = date
			return &finances.PenaltyAssignmentRow{ID: uuid.New(), TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, Date: date}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return nil, errors.New("reload not needed for this assertion")
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	_, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err, "a reload failure must not fail the request (see FallsBackToUnenrichedRow test)")
	after := time.Now()
	assert.False(t, gotDate.Before(before) || gotDate.After(after), "expected date to default to now(), got %v (window %v..%v)", gotDate, before, after)
}

// TestService_CreateAssignment_PassesExplicitPastDate verifies that a
// caller-supplied date in the past is passed through to the repository
// unchanged, not overridden by the "defaults to today" behavior.
func TestService_CreateAssignment_PassesExplicitPastDate(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	pastDate := time.Now().AddDate(0, -1, 0)
	var gotDate time.Time
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(_ context.Context, _, _, _ uuid.UUID, date time.Time, _ *string) (*finances.PenaltyAssignmentRow, error) {
			gotDate = date
			return &finances.PenaltyAssignmentRow{ID: uuid.New(), TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, Date: date}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return nil, errors.New("reload not needed for this assertion")
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID, Date: &openapi_types.Date{Time: pastDate}}
	_, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err)
	assert.True(t, pastDate.Equal(gotDate), "the caller-supplied past date must be passed through, not replaced with today")
}

// TestService_CreateAssignment_PassesNoteThrough verifies that a
// caller-supplied note is passed to the repository and appears on the
// enriched result.
func TestService_CreateAssignment_PassesNoteThrough(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	createdID := uuid.New()
	note := "Missed training without excuse"
	var gotNote *string
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(_ context.Context, _, _, _ uuid.UUID, date time.Time, n *string) (*finances.PenaltyAssignmentRow, error) {
			gotNote = n
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, Date: date, Note: n}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, Note: &note}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID, Note: &note}
	result, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err)
	require.NotNil(t, gotNote, "note must be passed through to the repository")
	assert.Equal(t, note, *gotNote)
	require.NotNil(t, result.Note)
	assert.Equal(t, note, *result.Note)
}

// TestService_CreateAssignment_WithoutNote_Succeeds verifies that omitting
// the note creates the assignment successfully with a nil note passed to the
// repository, rather than failing or synthesizing an empty string.
func TestService_CreateAssignment_WithoutNote_Succeeds(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	createdID := uuid.New()
	var noteWasPassed bool
	var gotNote *string
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(_ context.Context, _, _, _ uuid.UUID, date time.Time, n *string) (*finances.PenaltyAssignmentRow, error) {
			noteWasPassed = true
			gotNote = n
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, Date: date, Note: n}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	result, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err)
	assert.True(t, noteWasPassed)
	assert.Nil(t, gotNote)
	assert.Nil(t, result.Note)
}

func TestService_CreateAssignment_ReloadsEnrichedRowOnSuccess(t *testing.T) {
	t.Parallel()

	teamID, penaltyID, userID := uuid.New(), uuid.New(), uuid.New()
	createdID := uuid.New()
	label := "Late arrival"
	repo := &mockRepo{
		penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createAssignmentFn: func(_ context.Context, gotTeamID, gotUserID, gotPenaltyID uuid.UUID, gotDate time.Time, gotNote *string) (*finances.PenaltyAssignmentRow, error) {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, userID, gotUserID)
			assert.Equal(t, penaltyID, gotPenaltyID)
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, Date: gotDate, Note: gotNote}, nil
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
		createAssignmentFn: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, *string) (*finances.PenaltyAssignmentRow, error) {
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
		createAssignmentFn: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time, *string) (*finances.PenaltyAssignmentRow, error) {
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

// TestService_AssignmentPaid_DerivedFromPaidAmount covers the two paid
// buckets a penalty assignment's paidAmount/amount ratio maps to -- paid is
// never a settable field, only ever computed from what CreateAssignment's
// reloaded row reports as paidAmount. Unlike contributions there is no
// partial status: a fine is a small fixed amount, so anything short of the
// full amount is simply unpaid.
func TestService_AssignmentPaid_DerivedFromPaidAmount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		paidAmount int64
		amount     int64
		want       bool
	}{
		{"unpaid", 0, 500, false},
		{"partial payment still counts as unpaid", 200, 500, false},
		{"exact payment", 500, 500, true},
		{"overpayment", 600, 500, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			teamID, penaltyID, userID, id := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			amount := c.amount
			repo := &mockRepo{
				penaltyBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
				userIsMemberOfTeamFn:   func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
				createAssignmentFn: func(_ context.Context, gotTeamID, gotUserID, gotPenaltyID uuid.UUID, _ time.Time, _ *string) (*finances.PenaltyAssignmentRow, error) {
					return &finances.PenaltyAssignmentRow{ID: id, TeamID: gotTeamID, UserID: gotUserID, PenaltyID: &gotPenaltyID}, nil
				},
				getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
					return &finances.PenaltyAssignmentRow{ID: id, TeamID: teamID, UserID: userID, PenaltyID: &penaltyID, PaidAmount: c.paidAmount, PenaltyAmount: &amount}, nil
				},
			}
			svc := finances.NewService(repo, pagination.New(nil), slog.Default())
			body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
			result, err := svc.CreateAssignment(context.Background(), teamID, body)
			require.NoError(t, err)
			assert.Equal(t, c.want, result.Paid)
			assert.Equal(t, c.paidAmount, result.PaidAmount)
		})
	}
}

// ─── Contributions ───────────────────────────────────────────────────────────

// TestService_ContributionStatus_DerivedFromPaidAmount covers the three
// status buckets a contribution's paidAmount/amount ratio maps to -- status
// is never a settable field, only ever computed from what UpdateContribution
// and CreateContributions' underlying rows report as paidAmount.
func TestService_ContributionStatus_DerivedFromPaidAmount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		paidAmount int64
		amount     int64
		want       gen.ContributionStatus
	}{
		{"no payment", 0, 2500, gen.Open},
		{"partial payment", 1000, 2500, gen.Partial},
		{"exact payment", 2500, 2500, gen.Paid},
		{"overpayment", 3000, 2500, gen.Paid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			teamID, id := uuid.New(), uuid.New()
			repo := &mockRepo{
				updateContributionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, _ finances.ContributionPatch) (*finances.ContributionRow, error) {
					return &finances.ContributionRow{ID: gotID, TeamID: gotTeamID, Amount: c.amount, PaidAmount: c.paidAmount}, nil
				},
			}
			svc := finances.NewService(repo, pagination.New(nil), slog.Default())
			result, err := svc.UpdateContribution(context.Background(), id, teamID, &gen.UpdateContributionJSONRequestBody{})
			require.NoError(t, err)
			assert.Equal(t, c.want, result.Status)
			assert.Equal(t, c.paidAmount, result.PaidAmount)
		})
	}
}

// TestService_UpdateContribution_PassesDescriptionAndArchivedPatch verifies
// that description/archived from the request body reach the repository
// patch, independent of name/amount/dueDate.
func TestService_UpdateContribution_PassesDescriptionAndArchivedPatch(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	wantDesc := "Deckt Startgeld ab."
	wantArchived := true
	var gotPatch finances.ContributionPatch
	repo := &mockRepo{
		updateContributionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID, patch finances.ContributionPatch) (*finances.ContributionRow, error) {
			gotPatch = patch
			return &finances.ContributionRow{ID: gotID, TeamID: gotTeamID, Description: patch.Description, Archived: wantArchived}, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.UpdateContributionJSONRequestBody{Description: &wantDesc, Archived: &wantArchived}
	result, err := svc.UpdateContribution(context.Background(), id, teamID, body)
	require.NoError(t, err)
	require.NotNil(t, gotPatch.Description)
	assert.Equal(t, wantDesc, *gotPatch.Description)
	require.NotNil(t, gotPatch.Archived)
	assert.True(t, *gotPatch.Archived)
	assert.True(t, result.Archived)
	require.NotNil(t, result.Description)
	assert.Equal(t, wantDesc, *result.Description)
}

func TestService_CreateContributions_FansOutToOneRowPerUser(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}
	var gotUserIDs []uuid.UUID
	repo := &mockRepo{
		createContributionsFn: func(_ context.Context, gotTeamID uuid.UUID, name string, _ *string, amount int64, dueDate *time.Time, ids []uuid.UUID) ([]finances.ContributionRow, error) {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, "Turnier", name)
			assert.Equal(t, int64(1500), amount)
			gotUserIDs = ids
			rows := make([]finances.ContributionRow, 0, len(ids))
			for _, uid := range ids {
				rows = append(rows, finances.ContributionRow{ID: uuid.New(), TeamID: teamID, UserID: uid, Name: name, Amount: amount})
			}
			return rows, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateContributionsJSONRequestBody{Name: "Turnier", Amount: 1500, UserIds: userIDs}
	result, err := svc.CreateContributions(context.Background(), teamID, body)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, userIDs, gotUserIDs)
}

// Regression test mirroring TestService_CreateTransaction_RejectsAtCap:
// CreateContributions must refuse a fan-out that would push the team over
// maxContributionsPerTeam, without ever reaching the repo's insert.
func TestService_CreateContributions_RejectsAtCap(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		countContributionsFn: func(context.Context, uuid.UUID) (int, error) { return 100_000, nil },
		createContributionsFn: func(context.Context, uuid.UUID, string, *string, int64, *time.Time, []uuid.UUID) ([]finances.ContributionRow, error) {
			t.Fatal("CreateContributions must not be called once the team is at the contribution cap")
			return nil, nil
		},
	}

	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateContributionsJSONRequestBody{Name: "Turnier", Amount: 1500, UserIds: []uuid.UUID{uuid.New()}}
	_, err := svc.CreateContributions(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrTooManyContributions)
}

func TestService_DeleteContribution(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	repo := &mockRepo{
		deleteContributionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID) error {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			return nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	require.NoError(t, svc.DeleteContribution(context.Background(), id, teamID))
}

// ─── Transaction<->contribution linking ──────────────────────────────────────

func TestService_CreateTransaction_RejectsExpenseLinkedToContribution(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string, *uuid.UUID, *uuid.UUID, *string) (*finances.TransactionRow, error) {
			t.Fatal("CreateTransaction must not be called when an expense is linked to a contribution")
			return nil, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	contribID := uuid.New()
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Expense, Title: "Refund", Amount: 500, ContributionId: &contribID}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrContributionRequiresIncome)
}

func TestService_CreateTransaction_RejectsContributionFromAnotherTeam(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		contributionBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil },
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string, *uuid.UUID, *uuid.UUID, *string) (*finances.TransactionRow, error) {
			t.Fatal("CreateTransaction must not be called when the contribution isn't in the team")
			return nil, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	contribID := uuid.New()
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Beitrag", Amount: 500, ContributionId: &contribID}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrContributionNotInTeam)
}

func TestService_CreateTransaction_LinksIncomeToContribution(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	contribID := uuid.New()
	var gotContributionID *uuid.UUID
	repo := &mockRepo{
		contributionBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createTransactionFn: func(_ context.Context, gotTeamID uuid.UUID, txType, title string, amount int64, _ time.Time, _ *string, gotCID, _ *uuid.UUID, _ *string) (*finances.TransactionRow, error) {
			gotContributionID = gotCID
			return &finances.TransactionRow{ID: uuid.New(), TeamID: gotTeamID, Type: txType, Title: title, Amount: amount, ContributionID: gotCID}, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Beitrag", Amount: 500, ContributionId: &contribID}
	result, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.NoError(t, err)
	require.NotNil(t, gotContributionID)
	assert.Equal(t, contribID, *gotContributionID)
	require.NotNil(t, result.ContributionId)
	assert.Equal(t, contribID, *result.ContributionId)
}

// ─── Transaction<->penalty assignment linking ────────────────────────────────

func TestService_CreateTransaction_RejectsExpenseLinkedToPenaltyAssignment(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string, *uuid.UUID, *uuid.UUID, *string) (*finances.TransactionRow, error) {
			t.Fatal("CreateTransaction must not be called when an expense is linked to a penalty assignment")
			return nil, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	assignmentID := uuid.New()
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Expense, Title: "Refund", Amount: 500, PenaltyAssignmentId: &assignmentID}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrPenaltyAssignmentRequiresIncome)
}

func TestService_CreateTransaction_RejectsPenaltyAssignmentFromAnotherTeam(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		penaltyAssignmentBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil },
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string, *uuid.UUID, *uuid.UUID, *string) (*finances.TransactionRow, error) {
			t.Fatal("CreateTransaction must not be called when the penalty assignment isn't in the team")
			return nil, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	assignmentID := uuid.New()
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Strafe", Amount: 500, PenaltyAssignmentId: &assignmentID}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrPenaltyAssignmentNotInTeam)
}

func TestService_CreateTransaction_LinksIncomeToPenaltyAssignment(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	assignmentID := uuid.New()
	var gotAssignmentID *uuid.UUID
	repo := &mockRepo{
		penaltyAssignmentBelongsToTeamFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		createTransactionFn: func(_ context.Context, gotTeamID uuid.UUID, txType, title string, amount int64, _ time.Time, _ *string, _, gotPAID *uuid.UUID, _ *string) (*finances.TransactionRow, error) {
			gotAssignmentID = gotPAID
			return &finances.TransactionRow{ID: uuid.New(), TeamID: gotTeamID, Type: txType, Title: title, Amount: amount, PenaltyAssignmentID: gotPAID}, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Strafe", Amount: 500, PenaltyAssignmentId: &assignmentID}
	result, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.NoError(t, err)
	require.NotNil(t, gotAssignmentID)
	assert.Equal(t, assignmentID, *gotAssignmentID)
	require.NotNil(t, result.PenaltyAssignmentId)
	assert.Equal(t, assignmentID, *result.PenaltyAssignmentId)
}

// Regression test: a transaction must link to at most one of
// contributionId/penaltyAssignmentId -- rejected before either repo
// membership check runs.
func TestService_CreateTransaction_RejectsBothContributionAndPenaltyAssignment(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		createTransactionFn: func(context.Context, uuid.UUID, string, string, int64, time.Time, *string, *uuid.UUID, *uuid.UUID, *string) (*finances.TransactionRow, error) {
			t.Fatal("CreateTransaction must not be called when both contributionId and penaltyAssignmentId are set")
			return nil, nil
		},
	}
	svc := finances.NewService(repo, pagination.New(nil), slog.Default())
	contribID, assignmentID := uuid.New(), uuid.New()
	body := &gen.CreateTransactionJSONRequestBody{
		Type: gen.Income, Title: "Beitrag", Amount: 500,
		ContributionId: &contribID, PenaltyAssignmentId: &assignmentID,
	}
	_, err := svc.CreateTransaction(context.Background(), teamID, body)
	require.ErrorIs(t, err, finances.ErrTransactionLinksMultipleTargets)
}
