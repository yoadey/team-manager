package finances_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockFinanceService struct {
	getOverview          func(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error)
	createTransaction    func(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error)
	updateTransaction    func(ctx context.Context, id uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error)
	deleteTransaction    func(ctx context.Context, id uuid.UUID) error
	createPenalty        func(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error)
	updatePenalty        func(ctx context.Context, id uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error)
	deletePenalty        func(ctx context.Context, id uuid.UUID) error
	createAssignment     func(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error)
	deleteAssignment     func(ctx context.Context, id uuid.UUID) error
	toggleAssignmentPaid func(ctx context.Context, teamID, id uuid.UUID) (*gen.PenaltyAssignment, error)
	updateContribution   func(ctx context.Context, id uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error)
	toggleContribution   func(ctx context.Context, id uuid.UUID) (*gen.Contribution, error)
}

func (m *mockFinanceService) GetOverview(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error) {
	return m.getOverview(ctx, teamID)
}

func (m *mockFinanceService) CreateTransaction(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
	return m.createTransaction(ctx, teamID, body)
}

func (m *mockFinanceService) UpdateTransaction(ctx context.Context, id uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
	return m.updateTransaction(ctx, id, body)
}

func (m *mockFinanceService) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	return m.deleteTransaction(ctx, id)
}

func (m *mockFinanceService) CreatePenalty(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	return m.createPenalty(ctx, teamID, body)
}

func (m *mockFinanceService) UpdatePenalty(ctx context.Context, id uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	return m.updatePenalty(ctx, id, body)
}

func (m *mockFinanceService) DeletePenalty(ctx context.Context, id uuid.UUID) error {
	return m.deletePenalty(ctx, id)
}

func (m *mockFinanceService) CreateAssignment(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error) {
	return m.createAssignment(ctx, teamID, body)
}

func (m *mockFinanceService) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	return m.deleteAssignment(ctx, id)
}

func (m *mockFinanceService) ToggleAssignmentPaid(ctx context.Context, teamID, id uuid.UUID) (*gen.PenaltyAssignment, error) {
	return m.toggleAssignmentPaid(ctx, teamID, id)
}

func (m *mockFinanceService) UpdateContribution(ctx context.Context, id uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
	return m.updateContribution(ctx, id, body)
}

func (m *mockFinanceService) ToggleContribution(ctx context.Context, id uuid.UUID) (*gen.Contribution, error) {
	return m.toggleContribution(ctx, id)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

var (
	testTeamID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	testUserID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	testTxID   = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
)

func authedCtx() context.Context {
	user := &auth.UserRow{
		Id:          testUserID,
		Name:        "Test User",
		Email:       "test@example.com",
		AvatarColor: "#6366f1",
		CreatedAt:   time.Now(),
	}
	ctx := context.Background()
	return auth.ContextWithUser(ctx, user)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestHandler_GetFinanceOverview_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default())
	resp, err := h.GetFinanceOverview(context.Background(), gen.GetFinanceOverviewRequestObject{TeamId: testTeamID})
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestHandler_GetFinanceOverview_Success(t *testing.T) {
	t.Parallel()
	overview := &gen.FinanceOverview{
		Transactions:  []gen.Transaction{},
		Penalties:     []gen.Penalty{},
		Assignments:   []gen.PenaltyAssignment{},
		Contributions: []gen.Contribution{},
		OpenPenalties: []gen.OpenPenalty{},
		Balance:       100.0,
		Income:        200.0,
		Expense:       100.0,
	}
	svc := &mockFinanceService{
		getOverview: func(_ context.Context, _ uuid.UUID) (*gen.FinanceOverview, error) {
			return overview, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default())

	resp, err := h.GetFinanceOverview(authedCtx(), gen.GetFinanceOverviewRequestObject{TeamId: testTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetFinanceOverviewResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var result gen.FinanceOverview
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.InEpsilon(t, 100.0, result.Balance, 0.001)
}

func TestHandler_GetFinanceOverview_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		getOverview: func(_ context.Context, _ uuid.UUID) (*gen.FinanceOverview, error) {
			return nil, errors.New("db error")
		},
	}
	h := finances.NewHandler(svc, slog.Default())
	_, err := h.GetFinanceOverview(authedCtx(), gen.GetFinanceOverviewRequestObject{TeamId: testTeamID})
	require.Error(t, err)
}

func TestHandler_CreateTransaction_MissingBody(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default())
	_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: nil})
	require.Error(t, err)
}

func TestHandler_CreateTransaction_Success(t *testing.T) {
	t.Parallel()
	tx := &gen.Transaction{
		Id:     testTxID,
		TeamId: testTeamID,
		Type:   gen.Income,
		Title:  "Membership fee",
		Amount: 50.0,
		Date:   openapi_types.Date{Time: time.Now()},
	}
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			assert.Equal(t, "Membership fee", body.Title)
			return tx, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default())

	body := &gen.CreateTransactionJSONRequestBody{
		Type:   gen.Income,
		Title:  "Membership fee",
		Amount: 50.0,
	}
	resp, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreateTransactionResponse(w))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateTransaction_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx := &gen.Transaction{
		Id: testTxID, TeamId: testTeamID, Type: gen.Income, Title: "Fee", Amount: 50.0,
		Date: openapi_types.Date{Time: time.Now()},
	}
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			return tx, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)))

	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Fee", Amount: 50.0}
	_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "finance.mutation", rec["event"])
	assert.Equal(t, "transaction.create", rec["operation"])
	assert.Equal(t, testUserID.String(), rec["actor"])
	assert.Equal(t, testTxID.String(), rec["transactionId"])
}

func TestHandler_DeleteTransaction_Success(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		deleteTransaction: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	h := finances.NewHandler(svc, slog.Default())
	resp, err := h.DeleteTransaction(authedCtx(), gen.DeleteTransactionRequestObject{
		TeamId:        testTeamID,
		TransactionId: testTxID,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitDeleteTransactionResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ToggleContribution_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default())
	_, err := h.ToggleContribution(context.Background(), gen.ToggleContributionRequestObject{
		TeamId:         testTeamID,
		ContributionId: testTxID,
	})
	require.Error(t, err)
}
