package finances_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// ─── mock repository ────────────────────────────────────────────────────────

// mockRepo satisfies the unexported financeRepo interface via structural typing.
type mockRepo struct {
	listTransactionsFn       func(ctx context.Context, teamID uuid.UUID) ([]finances.TransactionRow, error)
	sumTransactionsFn        func(ctx context.Context, teamID uuid.UUID) (int64, int64, error)
	createTransactionFn      func(ctx context.Context, teamID uuid.UUID, txType, title string, amount int64, date time.Time, category *string) (*finances.TransactionRow, error)
	updateTransactionFn      func(ctx context.Context, id, teamID uuid.UUID, patch finances.TransactionPatch) (*finances.TransactionRow, error)
	deleteTransactionFn      func(ctx context.Context, id, teamID uuid.UUID) error
	listPenaltiesFn          func(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyRow, error)
	createPenaltyFn          func(ctx context.Context, teamID uuid.UUID, label string, amount int64) (*finances.PenaltyRow, error)
	updatePenaltyFn          func(ctx context.Context, id, teamID uuid.UUID, patch finances.PenaltyPatch) (*finances.PenaltyRow, error)
	deletePenaltyFn          func(ctx context.Context, id, teamID uuid.UUID) error
	penaltyBelongsToTeamFn   func(ctx context.Context, penaltyID, teamID uuid.UUID) (bool, error)
	listAssignmentsFn        func(ctx context.Context, teamID uuid.UUID) ([]finances.PenaltyAssignmentRow, error)
	getAssignmentByIDFn      func(ctx context.Context, id, teamID uuid.UUID) (*finances.PenaltyAssignmentRow, error)
	createAssignmentFn       func(ctx context.Context, teamID, userID, penaltyID uuid.UUID) (*finances.PenaltyAssignmentRow, error)
	deleteAssignmentFn       func(ctx context.Context, id, teamID uuid.UUID) error
	toggleAssignmentPaidFn   func(ctx context.Context, id, teamID uuid.UUID) (*finances.PenaltyAssignmentRow, error)
	userIsMemberOfTeamFn     func(ctx context.Context, userID, teamID uuid.UUID) (bool, error)
	listContributionsFn      func(ctx context.Context, teamID uuid.UUID) ([]finances.ContributionRow, error)
	countOpenContributionsFn func(ctx context.Context, teamID uuid.UUID) (int, error)
	updateContributionFn     func(ctx context.Context, id, teamID uuid.UUID, patch finances.ContributionPatch) (*finances.ContributionRow, error)
	toggleContributionFn     func(ctx context.Context, id, teamID uuid.UUID) (*finances.ContributionRow, error)
	listOpenPenaltiesFn      func(ctx context.Context, teamID uuid.UUID) ([]finances.OpenPenaltyAggregate, error)
	withReadTxFn             func(ctx context.Context, fn func(finances.OverviewReader) error) error
}

func (m *mockRepo) ListTransactions(ctx context.Context, teamID uuid.UUID) ([]finances.TransactionRow, error) {
	return m.listTransactionsFn(ctx, teamID)
}

func (m *mockRepo) SumTransactions(ctx context.Context, teamID uuid.UUID) (income, expense int64, err error) {
	return m.sumTransactionsFn(ctx, teamID)
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

func (m *mockRepo) CreateAssignment(ctx context.Context, teamID, userID, penaltyID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
	return m.createAssignmentFn(ctx, teamID, userID, penaltyID)
}

func (m *mockRepo) DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteAssignmentFn(ctx, id, teamID)
}

func (m *mockRepo) ToggleAssignmentPaid(ctx context.Context, id, teamID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
	return m.toggleAssignmentPaidFn(ctx, id, teamID)
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

func (m *mockRepo) ToggleContributionStatus(ctx context.Context, id, teamID uuid.UUID) (*finances.ContributionRow, error) {
	return m.toggleContributionFn(ctx, id, teamID)
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

	svc := finances.NewService(repo)
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

	svc := finances.NewService(repo)
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

	svc := finances.NewService(repo)
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

	svc := finances.NewService(repo)
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

	svc := finances.NewService(repo)
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

	svc := finances.NewService(repo)
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

	svc := finances.NewService(repo)
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: uuid.New(), UserId: uuid.New()}
	_, err := svc.CreateAssignment(context.Background(), uuid.New(), body)
	require.ErrorIs(t, err, finances.ErrUserNotInTeam)
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
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: penaltyID}, nil
		},
		getAssignmentByIDFn: func(_ context.Context, gotID, gotTeamID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			assert.Equal(t, createdID, gotID)
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: penaltyID, PenaltyLabel: &label}, nil
		},
	}

	svc := finances.NewService(repo)
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
			return &finances.PenaltyAssignmentRow{ID: createdID, TeamID: teamID, UserID: userID, PenaltyID: penaltyID}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return nil, errors.New("reload failed")
		},
	}

	svc := finances.NewService(repo)
	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: userID}
	result, err := svc.CreateAssignment(context.Background(), teamID, body)
	require.NoError(t, err, "a reload failure after a successful create should not fail the request")
	assert.Equal(t, createdID, result.Id)
	assert.Nil(t, result.Label, "unenriched fallback row has no penalty label")
}

func TestService_ToggleAssignmentPaid_ReloadsEnrichedRow(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	label := "Yellow card"
	repo := &mockRepo{
		toggleAssignmentPaidFn: func(_ context.Context, gotID, gotTeamID uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			return &finances.PenaltyAssignmentRow{ID: id, TeamID: teamID, Paid: true}, nil
		},
		getAssignmentByIDFn: func(context.Context, uuid.UUID, uuid.UUID) (*finances.PenaltyAssignmentRow, error) {
			return &finances.PenaltyAssignmentRow{ID: id, TeamID: teamID, Paid: true, PenaltyLabel: &label}, nil
		},
	}

	svc := finances.NewService(repo)
	result, err := svc.ToggleAssignmentPaid(context.Background(), teamID, id)
	require.NoError(t, err)
	assert.True(t, result.Paid)
	require.NotNil(t, result.Label)
	assert.Equal(t, label, *result.Label)
}

// ─── Contributions ───────────────────────────────────────────────────────────

func TestService_ToggleContribution(t *testing.T) {
	t.Parallel()

	teamID, id := uuid.New(), uuid.New()
	repo := &mockRepo{
		toggleContributionFn: func(_ context.Context, gotID, gotTeamID uuid.UUID) (*finances.ContributionRow, error) {
			assert.Equal(t, id, gotID)
			assert.Equal(t, teamID, gotTeamID)
			return &finances.ContributionRow{ID: id, TeamID: teamID, Status: "paid"}, nil
		},
	}

	svc := finances.NewService(repo)
	result, err := svc.ToggleContribution(context.Background(), id, teamID)
	require.NoError(t, err)
	assert.Equal(t, gen.ContributionStatus("paid"), result.Status)
}
