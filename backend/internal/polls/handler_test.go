package polls_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/polls"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockPollService struct {
	listByTeam func(ctx context.Context, teamID, currentUserID uuid.UUID, limit int, cursor string) ([]gen.Poll, *string, error)
	create     func(ctx context.Context, teamID, creatorID uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error)
	vote       func(ctx context.Context, pollID, teamID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error)
	deletePoll func(ctx context.Context, id, teamID uuid.UUID) error
}

func (m *mockPollService) ListByTeam(ctx context.Context, teamID, currentUserID uuid.UUID, limit int, cursor string) ([]gen.Poll, *string, error) {
	return m.listByTeam(ctx, teamID, currentUserID, limit, cursor)
}

func (m *mockPollService) Create(ctx context.Context, teamID, creatorID uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error) {
	return m.create(ctx, teamID, creatorID, body)
}

func (m *mockPollService) Vote(ctx context.Context, pollID, teamID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error) {
	return m.vote(ctx, pollID, teamID, userID, optionIDs)
}

func (m *mockPollService) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	return m.deletePoll(ctx, id, teamID)
}

func authedCtx() context.Context {
	return auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Test User", Email: "t@example.com"})
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestPollHandler_CreatePoll_RejectsTooLongOption(t *testing.T) {
	t.Parallel()
	svc := &mockPollService{
		create: func(_ context.Context, _, _ uuid.UUID, _ *gen.CreatePollRequest) (gen.Poll, error) {
			t.Fatal("service should not be called when option validation fails")
			return gen.Poll{}, nil
		},
	}
	h := polls.NewHandler(svc, slog.Default())

	body := &gen.CreatePollRequest{
		Question: "Which date?",
		Options:  []string{"Monday", strings.Repeat("x", 501)},
	}
	_, err := h.CreatePoll(authedCtx(), gen.CreatePollRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestPollHandler_CreatePoll_RejectsTooFewOptions(t *testing.T) {
	t.Parallel()
	svc := &mockPollService{
		create: func(_ context.Context, _, _ uuid.UUID, _ *gen.CreatePollRequest) (gen.Poll, error) {
			t.Fatal("service should not be called when option count validation fails")
			return gen.Poll{}, nil
		},
	}
	h := polls.NewHandler(svc, slog.Default())

	body := &gen.CreatePollRequest{Question: "Which date?", Options: []string{"Monday"}}
	_, err := h.CreatePoll(authedCtx(), gen.CreatePollRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

// The OpenAPI contract declares maxItems: 4 on CreatePollRequest.options, but
// oapi-codegen doesn't generate runtime array-length checks and no request
// validator is wired into the router — the handler is the only enforcement
// point, matching how minItems is already enforced.
func TestPollHandler_CreatePoll_RejectsTooManyOptions(t *testing.T) {
	t.Parallel()
	svc := &mockPollService{
		create: func(_ context.Context, _, _ uuid.UUID, _ *gen.CreatePollRequest) (gen.Poll, error) {
			t.Fatal("service should not be called when option count validation fails")
			return gen.Poll{}, nil
		},
	}
	h := polls.NewHandler(svc, slog.Default())

	body := &gen.CreatePollRequest{Question: "Which date?", Options: []string{"A", "B", "C", "D", "E"}}
	_, err := h.CreatePoll(authedCtx(), gen.CreatePollRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestPollHandler_CreatePoll_AcceptsFourOptions(t *testing.T) {
	t.Parallel()
	svc := &mockPollService{
		create: func(_ context.Context, _, _ uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error) {
			return gen.Poll{Options: make([]gen.PollOption, len(body.Options))}, nil
		},
	}
	h := polls.NewHandler(svc, slog.Default())

	body := &gen.CreatePollRequest{Question: "Which date?", Options: []string{"A", "B", "C", "D"}}
	_, err := h.CreatePoll(authedCtx(), gen.CreatePollRequestObject{TeamId: uuid.New(), Body: body})
	require.NoError(t, err)
}

func TestPollHandler_CreatePoll_Success(t *testing.T) {
	t.Parallel()
	svc := &mockPollService{
		create: func(_ context.Context, _, _ uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error) {
			assert.Equal(t, "Which date?", body.Question)
			return gen.Poll{Id: uuid.New(), Question: body.Question}, nil
		},
	}
	h := polls.NewHandler(svc, slog.Default())

	body := &gen.CreatePollRequest{Question: "Which date?", Options: []string{"Monday", "Tuesday"}}
	resp, err := h.CreatePoll(authedCtx(), gen.CreatePollRequestObject{TeamId: uuid.New(), Body: body})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreatePollResponse(w))
	assert.Equal(t, http.StatusCreated, w.Code)
}
