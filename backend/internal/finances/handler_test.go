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
	createContributions func(ctx context.Context, teamID uuid.UUID, body *gen.CreateContributionsJSONRequestBody) ([]gen.Contribution, error)
	updateContribution  func(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error)
	deleteContribution  func(ctx context.Context, id, teamID uuid.UUID) error
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

func (m *mockFinanceService) CreateContributions(ctx context.Context, teamID uuid.UUID, body *gen.CreateContributionsJSONRequestBody) ([]gen.Contribution, error) {
	return m.createContributions(ctx, teamID, body)
}

func (m *mockFinanceService) UpdateContribution(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
	return m.updateContribution(ctx, id, teamID, body)
}

func (m *mockFinanceService) DeleteContribution(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deleteContribution(ctx, id, teamID)
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

func TestHandler_CreateTransaction_RejectsOversizedNote(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createTransaction: func(_ context.Context, _ uuid.UUID, _ *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
			t.Fatal("service should not be called when note validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	note := strings.Repeat("x", 10001)
	body := &gen.CreateTransactionJSONRequestBody{Type: gen.Income, Title: "Membership fee", Amount: 100, Note: &note}
	_, err := h.CreateTransaction(authedCtx(), gen.CreateTransactionRequestObject{TeamId: testTeamID, Body: body})
	require.Error(t, err)
}

func TestHandler_UpdateTransaction_RejectsOversizedNote(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		updateTransaction: func(_ context.Context, _, _ uuid.UUID, _ *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
			t.Fatal("service should not be called when note validation fails")
			return nil, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)

	note := strings.Repeat("x", 10001)
	body := &gen.UpdateTransactionJSONRequestBody{Note: &note}
	_, err := h.UpdateTransaction(authedCtx(), gen.UpdateTransactionRequestObject{TransactionId: testTxID, TeamId: testTeamID, Body: body})
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
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, testTxID.String(), rec["transactionId"])
	assert.Equal(t, float64(5000), rec["amount"])
	assert.Equal(t, "income", rec["type"])
}

// Regression test: UpdateTransaction's audit record used to omit both teamId
// (unlike its own create counterpart) and the amount, so an operator could
// not tell what a transaction's amount changed to -- or even which team it
// belonged to -- from the audit log alone.
func TestHandler_UpdateTransaction_EmitsAuditEventWithAmountAndTeam(t *testing.T) {
	t.Parallel()
	tx := &gen.Transaction{
		Id: testTxID, TeamId: testTeamID, Type: gen.Expense, Title: "Fee", Amount: 9999,
		Date: openapi_types.Date{Time: time.Now()},
	}
	svc := &mockFinanceService{
		updateTransaction: func(_ context.Context, _, _ uuid.UUID, _ *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
			return tx, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	var newAmount int64 = 9999
	body := &gen.UpdateTransactionJSONRequestBody{Amount: &newAmount}
	_, err := h.UpdateTransaction(authedCtx(), gen.UpdateTransactionRequestObject{TransactionId: testTxID, TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "transaction.update", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, testTxID.String(), rec["transactionId"])
	assert.Equal(t, float64(9999), rec["amount"])
	assert.Equal(t, "expense", rec["type"])
}

// Regression test: DeleteTransaction's audit record used to omit teamId,
// unlike CreateTransaction's.
func TestHandler_DeleteTransaction_EmitsAuditEventWithTeam(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		deleteTransaction: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	_, err := h.DeleteTransaction(authedCtx(), gen.DeleteTransactionRequestObject{TeamId: testTeamID, TransactionId: testTxID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "transaction.delete", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, testTxID.String(), rec["transactionId"])
}

// Regression test: CreatePenalty's audit record used to omit the amount.
func TestHandler_CreatePenalty_EmitsAuditEventWithAmount(t *testing.T) {
	t.Parallel()
	penaltyID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	p := &gen.Penalty{Id: penaltyID, TeamId: testTeamID, Label: "Late", Amount: 750}
	svc := &mockFinanceService{
		createPenalty: func(_ context.Context, _ uuid.UUID, _ *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
			return p, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreatePenaltyJSONRequestBody{Label: "Late", Amount: 750}
	_, err := h.CreatePenalty(authedCtx(), gen.CreatePenaltyRequestObject{TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "penalty.create", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, penaltyID.String(), rec["penaltyId"])
	assert.Equal(t, float64(750), rec["amount"])
}

// Regression test: UpdatePenalty's audit record used to omit both teamId and
// the amount.
func TestHandler_UpdatePenalty_EmitsAuditEventWithAmountAndTeam(t *testing.T) {
	t.Parallel()
	penaltyID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	p := &gen.Penalty{Id: penaltyID, TeamId: testTeamID, Label: "Late", Amount: 1200}
	svc := &mockFinanceService{
		updatePenalty: func(_ context.Context, _, _ uuid.UUID, _ *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error) {
			return p, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	var newAmount int64 = 1200
	body := &gen.UpdatePenaltyJSONRequestBody{Amount: &newAmount}
	_, err := h.UpdatePenalty(authedCtx(), gen.UpdatePenaltyRequestObject{PenaltyId: penaltyID, TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "penalty.update", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, penaltyID.String(), rec["penaltyId"])
	assert.Equal(t, float64(1200), rec["amount"])
}

// Regression test: DeletePenalty's audit record used to omit teamId.
func TestHandler_DeletePenalty_EmitsAuditEventWithTeam(t *testing.T) {
	t.Parallel()
	penaltyID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	svc := &mockFinanceService{
		deletePenalty: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	_, err := h.DeletePenalty(authedCtx(), gen.DeletePenaltyRequestObject{TeamId: testTeamID, PenaltyId: penaltyID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "penalty.delete", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, penaltyID.String(), rec["penaltyId"])
}

// Regression test: CreatePenaltyAssignment's audit record used to omit the
// (snapshotted) penalty amount.
func TestHandler_CreatePenaltyAssignment_EmitsAuditEventWithAmount(t *testing.T) {
	t.Parallel()
	assignmentID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	penaltyID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	amount := int64(500)
	a := &gen.PenaltyAssignment{
		Id: assignmentID, TeamId: testTeamID, UserId: testUserID, PenaltyId: &penaltyID,
		Date: openapi_types.Date{Time: time.Now()}, Amount: &amount,
	}
	svc := &mockFinanceService{
		createAssignment: func(_ context.Context, _ uuid.UUID, _ *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error) {
			return a, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreatePenaltyAssignmentJSONRequestBody{PenaltyId: penaltyID, UserId: testUserID}
	_, err := h.CreatePenaltyAssignment(authedCtx(), gen.CreatePenaltyAssignmentRequestObject{TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "assignment.create", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, assignmentID.String(), rec["assignmentId"])
	assert.Equal(t, float64(500), rec["amount"])
}

// Regression test: DeletePenaltyAssignment's audit record used to omit teamId.
func TestHandler_DeletePenaltyAssignment_EmitsAuditEventWithTeam(t *testing.T) {
	t.Parallel()
	assignmentID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	svc := &mockFinanceService{
		deleteAssignment: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	_, err := h.DeletePenaltyAssignment(authedCtx(), gen.DeletePenaltyAssignmentRequestObject{TeamId: testTeamID, AssignmentId: assignmentID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "assignment.delete", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, assignmentID.String(), rec["assignmentId"])
}

// Regression test: CreateContributions's audit record used to omit the
// per-contribution amount.
func TestHandler_CreateContributions_EmitsAuditEventWithAmount(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createContributions: func(_ context.Context, _ uuid.UUID, body *gen.CreateContributionsJSONRequestBody) ([]gen.Contribution, error) {
			out := make([]gen.Contribution, 0, len(body.UserIds))
			for _, uid := range body.UserIds {
				out = append(out, gen.Contribution{Id: testTxID, TeamId: testTeamID, UserId: uid, Name: body.Name, Amount: body.Amount})
			}
			return out, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	body := &gen.CreateContributionsJSONRequestBody{Name: "Beitrag", Amount: 2500, UserIds: []uuid.UUID{testUserID}}
	_, err := h.CreateContributions(authedCtx(), gen.CreateContributionsRequestObject{TeamId: testTeamID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "contribution.create", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, float64(1), rec["count"])
	assert.Equal(t, float64(2500), rec["amount"])
}

// Regression test: UpdateContribution's audit record used to omit both
// teamId and the amount.
func TestHandler_UpdateContribution_EmitsAuditEventWithAmountAndTeam(t *testing.T) {
	t.Parallel()
	c := &gen.Contribution{Id: testTxID, TeamId: testTeamID, UserId: testUserID, Name: "Beitrag", Amount: 3000}
	svc := &mockFinanceService{
		updateContribution: func(_ context.Context, _, _ uuid.UUID, _ *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
			return c, nil
		},
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	var newAmount int64 = 3000
	body := &gen.UpdateContributionJSONRequestBody{Amount: &newAmount}
	_, err := h.UpdateContribution(authedCtx(), gen.UpdateContributionRequestObject{TeamId: testTeamID, ContributionId: testTxID, Body: body})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "contribution.update", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, testTxID.String(), rec["contributionId"])
	assert.Equal(t, float64(3000), rec["amount"])
}

// Regression test: DeleteContribution's audit record used to omit teamId.
func TestHandler_DeleteContribution_EmitsAuditEventWithTeam(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		deleteContribution: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	var buf bytes.Buffer
	h := finances.NewHandler(svc, slog.New(slog.NewJSONHandler(&buf, nil)), nil)

	_, err := h.DeleteContribution(authedCtx(), gen.DeleteContributionRequestObject{TeamId: testTeamID, ContributionId: testTxID})
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "contribution.delete", rec["operation"])
	assert.Equal(t, testTeamID.String(), rec["teamId"])
	assert.Equal(t, testTxID.String(), rec["contributionId"])
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

func TestHandler_UpdateContribution_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default(), nil)
	_, err := h.UpdateContribution(context.Background(), gen.UpdateContributionRequestObject{
		TeamId:         testTeamID,
		ContributionId: testTxID,
		Body:           &gen.UpdateContributionJSONRequestBody{},
	})
	require.Error(t, err)
}

func TestHandler_CreateContributions_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default(), nil)
	_, err := h.CreateContributions(context.Background(), gen.CreateContributionsRequestObject{
		TeamId: testTeamID,
		Body:   &gen.CreateContributionsJSONRequestBody{Name: "Beitrag", Amount: 2500, UserIds: []uuid.UUID{testUserID}},
	})
	require.Error(t, err)
}

func TestHandler_CreateContributions_RejectsEmptyUserIds(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{}, slog.Default(), nil)
	_, err := h.CreateContributions(authedCtx(), gen.CreateContributionsRequestObject{
		TeamId: testTeamID,
		Body:   &gen.CreateContributionsJSONRequestBody{Name: "Beitrag", Amount: 2500, UserIds: []uuid.UUID{}},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestHandler_CreateContributions_Success(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		createContributions: func(_ context.Context, _ uuid.UUID, body *gen.CreateContributionsJSONRequestBody) ([]gen.Contribution, error) {
			out := make([]gen.Contribution, 0, len(body.UserIds))
			for _, uid := range body.UserIds {
				out = append(out, gen.Contribution{
					Id: testTxID, TeamId: testTeamID, UserId: uid,
					Name: body.Name, Amount: body.Amount, PaidAmount: 0, Status: gen.Open,
				})
			}
			return out, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)
	resp, err := h.CreateContributions(authedCtx(), gen.CreateContributionsRequestObject{
		TeamId: testTeamID,
		Body:   &gen.CreateContributionsJSONRequestBody{Name: "Beitrag", Amount: 2500, UserIds: []uuid.UUID{testUserID}},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreateContributionsResponse(w))
	assert.Equal(t, http.StatusCreated, w.Code)
	var got []gen.Contribution
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "Beitrag", got[0].Name)
}

func TestHandler_CreateContributions_RejectsOversizedDescription(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{
		createContributions: func(_ context.Context, _ uuid.UUID, _ *gen.CreateContributionsJSONRequestBody) ([]gen.Contribution, error) {
			t.Fatal("service should not be called when description validation fails")
			return nil, nil
		},
	}, slog.Default(), nil)

	description := strings.Repeat("x", 2001)
	_, err := h.CreateContributions(authedCtx(), gen.CreateContributionsRequestObject{
		TeamId: testTeamID,
		Body: &gen.CreateContributionsJSONRequestBody{
			Name: "Beitrag", Description: &description, Amount: 2500, UserIds: []uuid.UUID{testUserID},
		},
	})
	require.Error(t, err)
}

func TestHandler_UpdateContribution_RejectsOversizedDescription(t *testing.T) {
	t.Parallel()
	h := finances.NewHandler(&mockFinanceService{
		updateContribution: func(_ context.Context, _, _ uuid.UUID, _ *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
			t.Fatal("service should not be called when description validation fails")
			return nil, nil
		},
	}, slog.Default(), nil)

	description := strings.Repeat("x", 2001)
	_, err := h.UpdateContribution(authedCtx(), gen.UpdateContributionRequestObject{
		TeamId: testTeamID, ContributionId: testTxID,
		Body: &gen.UpdateContributionJSONRequestBody{Description: &description},
	})
	require.Error(t, err)
}

// TestHandler_UpdateContribution_ArchivedRoundTrips verifies the archived
// flag flows from the request body through to the response unchanged.
func TestHandler_UpdateContribution_ArchivedRoundTrips(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		updateContribution: func(_ context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
			require.NotNil(t, body.Archived)
			return &gen.Contribution{
				Id: id, TeamId: teamID, UserId: testUserID,
				Name: "Beitrag", Amount: 2500, PaidAmount: 0, Status: gen.Open, Archived: *body.Archived,
			}, nil
		},
	}
	h := finances.NewHandler(svc, slog.Default(), nil)
	archived := true
	resp, err := h.UpdateContribution(authedCtx(), gen.UpdateContributionRequestObject{
		TeamId: testTeamID, ContributionId: testTxID,
		Body: &gen.UpdateContributionJSONRequestBody{Archived: &archived},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitUpdateContributionResponse(w))
	var got gen.Contribution
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.True(t, got.Archived)
}

func TestHandler_DeleteContribution_NotFoundReturns404(t *testing.T) {
	t.Parallel()
	svc := &mockFinanceService{
		deleteContribution: func(_ context.Context, _, _ uuid.UUID) error { return pgx.ErrNoRows },
	}
	h := finances.NewHandler(svc, slog.Default(), nil)
	_, err := h.DeleteContribution(authedCtx(), gen.DeleteContributionRequestObject{
		TeamId: testTeamID, ContributionId: testTxID,
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
