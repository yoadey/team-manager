package finances_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockFinanceService struct {
	getOverview         func(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error)
	listTransactions    func(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.Transaction, *string, error)
	createTransaction   func(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error)
	updateTransaction   func(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error)
	deleteTransaction   func(ctx context.Context, id, teamID uuid.UUID) error
	createPenalty       func(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error)
	updatePenalty       func(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error)
	deletePenalty       func(ctx context.Context, id, teamID uuid.UUID) error
	createAssignment    func(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error)
	deleteAssignment    func(ctx context.Context, id, teamID uuid.UUID) error
	setPenaltyPaid      func(ctx context.Context, teamID, id uuid.UUID, paid bool) (*gen.PenaltyAssignment, error)
	updateContribution  func(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error)
	setContributionPaid func(ctx context.Context, id, teamID uuid.UUID, paid bool) (*gen.Contribution, error)
}

func (m *mockFinanceService) GetOverview(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error) {
	return m.getOverview(ctx, teamID)
}

func (m *mockFinanceService) ListTransactions(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.Transaction, *string, error) {
	return m.listTransactions(ctx, teamID, limit, cursor)
}

func (m *mockFinanceService) CreateTransaction(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
	return m.createTransaction(ctx, teamID, body)
}

func (m *mockFinanceService) UpdateTransaction(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
	return m.updateTransaction(ctx, id, teamID, body)
}

func (m *mockFinanceService) DeleteTransaction(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteTransaction(ctx, id, teamID)
}

func (m *mockFinanceService) CreatePenalty(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	return m.createPenalty(ctx, teamID, body)
}

func (m *mockFinanceService) UpdatePenalty(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	return m.updatePenalty(ctx, id, teamID, body)
}

func (m *mockFinanceService) DeletePenalty(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deletePenalty(ctx, id, teamID)
}

func (m *mockFinanceService) CreateAssignment(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error) {
	return m.createAssignment(ctx, teamID, body)
}

func (m *mockFinanceService) DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteAssignment(ctx, id, teamID)
}

func (m *mockFinanceService) SetPenaltyPaid(ctx context.Context, teamID, id uuid.UUID, paid bool) (*gen.PenaltyAssignment, error) {
	return m.setPenaltyPaid(ctx, teamID, id, paid)
}

func (m *mockFinanceService) UpdateContribution(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
	return m.updateContribution(ctx, id, teamID, body)
}

func (m *mockFinanceService) SetContributionPaid(ctx context.Context, id, teamID uuid.UUID, paid bool) (*gen.Contribution, error) {
	return m.setContributionPaid(ctx, id, teamID, paid)
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

// findAuditLogLine parses a multi-line JSON log buffer (a handler that logs
// both an error and an audit record writes two separate JSON objects to the
// same buffer, which json.Unmarshal can't parse as one value) and returns the
// first line that is an audit record.
func findAuditLogLine(t *testing.T, buf []byte) map[string]any {
	t.Helper()
	for _, line := range strings.Split(string(buf), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		if rec["audit"] == true {
			return rec
		}
	}
	t.Fatal("no audit log line found")
	return nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestHandler_GetFinanceOverview_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default(), nil)
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
		Balance:       10000,
		Income:        20000,
		Expense:       10000,
	}
	svc := &mockFinanceService{
		getOverview: func(_ context.Context, _ uuid.UUID) (*gen.FinanceOverview, error) {
			return overview, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	resp, err := h.GetFinanceOverview(authedCtx(), gen.GetFinanceOverviewRequestObject{TeamId: testTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetFinanceOverviewResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var result gen.FinanceOverview
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, int64(10000), result.Balance)
}

func TestHandler_GetFinanceOverview_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		getOverview: func(_ context.Context, _ uuid.UUID) (*gen.FinanceOverview, error) {
			return nil, errors.New("db error")
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)
	_, err := h.GetFinanceOverview(authedCtx(), gen.GetFinanceOverviewRequestObject{TeamId: testTeamID})
	require.Error(t, err)
}

func TestHandler_ListTransactions_Success(t *testing.T) {
	t.Parallel()
	next := "next-cursor"
	svc := &mockFinanceService{
		listTransactions: func(_ context.Context, _ uuid.UUID, limit int, cursor string) ([]gen.Transaction, *string, error) {
			assert.Equal(t, 50, limit, "an omitted limit param must default to 50")
			assert.Equal(t, "", cursor)
			return []gen.Transaction{{Id: testTxID, TeamId: testTeamID, Type: gen.Income, Title: "Dues", Amount: 100, Date: openapi_types.Date{Time: time.Now()}}}, &next, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	resp, err := h.ListTransactions(authedCtx(), gen.ListTransactionsRequestObject{TeamId: testTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListTransactionsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var result gen.ListTransactions200JSONResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result.Items, 1)
	require.NotNil(t, result.NextCursor)
	assert.Equal(t, "next-cursor", *result.NextCursor)
}

func TestHandler_ListTransactions_InvalidCursorIsBadRequest(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		listTransactions: func(_ context.Context, _ uuid.UUID, _ int, _ string) ([]gen.Transaction, *string, error) {
			return nil, nil, pagination.ErrInvalidCursor
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	bad := "not-a-cursor"
	_, err := h.ListTransactions(authedCtx(), gen.ListTransactionsRequestObject{TeamId: testTeamID, Params: gen.ListTransactionsParams{Cursor: &bad}})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestHandler_CreateTransaction_MissingBody(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default(), nil)
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
		Amount: 5000,
		Date:   openapi_types.Date{Time: time.Now()},
	}
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			assert.Equal(t, "Membership fee", body.Title)
			return tx, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	body := &gen.CreateTransactionJSONRequestBody{
		Type:   gen.Income,
		Title:  "Membership fee",
		Amount: 5000,
	}
	resp, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreateTransactionResponse(w))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateTransaction_RejectsNonPositiveAmount(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			t.Fatal("service should not be called when amount validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	for _, amount := range []int64{0, -10, 100_000_001, math.MaxInt64} {
		body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Membership fee", Amount: amount}
		_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
		require.Error(t, err)
	}
}

func TestHandler_CreateTransaction_RejectsOversizedCategory(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			t.Fatal("service should not be called when category validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	category := strings.Repeat("x", 256)
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Membership fee", Amount: 100, Category: &category}
	_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.Error(t, err)
}

func TestHandler_CreateTransaction_RejectsInvalidType(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			t.Fatal("service should not be called when type validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	body := &gen.CreateTransactionJSONRequestBody{Type: gen.TransactionType("bogus"), Title: "Membership fee", Amount: 100}
	_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.Error(t, err)
}

func TestHandler_UpdateTransaction_RejectsNonPositiveAmount(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		updateTransaction: func(_ context.Context, _, _ uuid.UUID, _ *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
			t.Fatal("service should not be called when amount validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	var badAmount int64 = -1
	body := &gen.UpdateTransactionJSONRequestBody{Amount: &badAmount}
	_, err := h.UpdateTransaction(authedCtx(), gen.UpdateTransactionRequestObject{TransactionId: testTxID, TeamId: testTeamID, Body: body})
	require.Error(t, err)
}

func TestHandler_CreatePenalty_RejectsNonPositiveAmount(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createPenalty: func(_ context.Context, _ uuid.UUID, _ *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
			t.Fatal("service should not be called when amount validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	body := &gen.CreatePenaltyJSONRequestBody{Label: "Late arrival", Amount: 0}
	_, err := h.CreatePenalty(authedCtx(), gen.CreatePenaltyRequestObject{TeamId: testTeamID, Body: body})
	require.Error(t, err)
}

func TestHandler_CreateTransaction_EmitsAuditEvent(t *testing.T) {
	t.Parallel()
	tx := &gen.Transaction{
		Id: testTxID, TeamId: testTeamID, Type: gen.Income, Title: "Fee", Amount: 5000,
		Date: openapi_types.Date{Time: time.Now()},
	}
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			return tx, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Fee", Amount: 5000}
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

// Regression test: unlike UpdateTransaction/DeleteTransaction/UpdatePenalty/etc.,
// CreateTransaction's service-error branch used to only log via h.logger and
// never call h.recordFinanceFailure, leaving no audit_log trace of a failed
// (or repeatedly probed) transaction creation.
func TestHandler_CreateTransaction_ServiceError_RecordsAuditFailure(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			return nil, errors.New("db error")
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Fee", Amount: 5000}
	_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.Error(t, err)

	rec := findAuditLogLine(t, buf.Bytes())
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "finance.mutation", rec["event"])
	assert.Equal(t, "failure", rec["outcome"])
	assert.Equal(t, "transaction.create", rec["operation"])
}

// Regression test: same gap as CreateTransaction above, for CreatePenalty.
func TestHandler_CreatePenalty_ServiceError_RecordsAuditFailure(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createPenalty: func(_ context.Context, _ uuid.UUID, _ *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
			return nil, errors.New("db error")
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreatePenaltyJSONRequestBody{Label: "Late", Amount: 500}
	_, err := h.CreatePenalty(authedCtx(), gen.CreatePenaltyRequestObject{TeamId: testTeamID, Body: body})
	require.Error(t, err)

	rec := findAuditLogLine(t, buf.Bytes())
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "finance.mutation", rec["event"])
	assert.Equal(t, "failure", rec["outcome"])
	assert.Equal(t, "penalty.create", rec["operation"])
}

func TestHandler_DeleteTransaction_Success(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		deleteTransaction: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	h := finances.NewHandler(svc, slog.Default(), nil)
	resp, err := h.DeleteTransaction(authedCtx(), gen.DeleteTransactionRequestObject{
		TeamId:        testTeamID,
		TransactionId: testTxID,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitDeleteTransactionResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_SetContributionPaid_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default(), nil)
	_, err := h.SetContributionPaid(context.Background(), gen.SetContributionPaidRequestObject{
		TeamId:         testTeamID,
		ContributionId: testTxID,
		Body:           &gen.SetPaidRequest{Paid: true},
	})
	require.Error(t, err)
}

func TestHandler_SetPenaltyPaid_NotFoundReturns404(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		setPenaltyPaid: func(_ context.Context, _, _ uuid.UUID, _ bool) (*gen.PenaltyAssignment, error) {
			return nil, pgx.ErrNoRows
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	_, err := h.SetPenaltyPaid(authedCtx(), gen.SetPenaltyPaidRequestObject{
		TeamId:       testTeamID,
		AssignmentId: testTxID,
		Body:         &gen.SetPaidRequest{Paid: true},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Status)
}

// Regression test: Service.CreateAssignment returns bare pgx.ErrNoRows when
// the just-created row is gone by the time it's reloaded (a concurrent
// DeletePenalty cascaded it away) -- unlike every other write handler in
// this file, CreatePenaltyAssignment had no pgx.ErrNoRows branch at all, so
// this benign race fell through to a generic 500 instead of the intended 404.
func TestHandler_CreatePenaltyAssignment_ReloadRaceReturns404(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createAssignment: func(_ context.Context, _ uuid.UUID, _ *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error) {
			return nil, pgx.ErrNoRows
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: testTxID, UserId: testTxID}
	_, err := h.CreatePenaltyAssignment(authedCtx(), gen.CreatePenaltyAssignmentRequestObject{
		TeamId: testTeamID,
		Body:   body,
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Status)
}

func TestHandler_SetPenaltyPaid_ServiceErrorReturns500(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		setPenaltyPaid: func(_ context.Context, _, _ uuid.UUID, _ bool) (*gen.PenaltyAssignment, error) {
			return nil, errors.New("db error")
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	_, err := h.SetPenaltyPaid(authedCtx(), gen.SetPenaltyPaidRequestObject{
		TeamId:       testTeamID,
		AssignmentId: testTxID,
		Body:         &gen.SetPaidRequest{Paid: true},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Status)
}
