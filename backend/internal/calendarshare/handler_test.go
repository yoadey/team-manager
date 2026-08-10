package calendarshare_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/calendarshare"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

type mockShareService struct {
	grantFn       func(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*calendarshare.ShareRow, error)
	revokeFn      func(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error
	listGrantsFn  func(ctx context.Context, ownerTeamID uuid.UUID) ([]calendarshare.ShareRow, error)
	listSourcesFn func(ctx context.Context, viewerTeamID uuid.UUID) ([]calendarshare.ShareRow, error)
	listEventsFn  func(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID, from, to *time.Time) ([]calendarshare.RedactedEventRow, error)
}

func (m *mockShareService) Grant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*calendarshare.ShareRow, error) {
	return m.grantFn(ctx, ownerTeamID, viewerTeamID)
}

func (m *mockShareService) Revoke(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error {
	return m.revokeFn(ctx, ownerTeamID, viewerTeamID)
}

func (m *mockShareService) ListGrants(ctx context.Context, ownerTeamID uuid.UUID) ([]calendarshare.ShareRow, error) {
	return m.listGrantsFn(ctx, ownerTeamID)
}

func (m *mockShareService) ListSources(ctx context.Context, viewerTeamID uuid.UUID) ([]calendarshare.ShareRow, error) {
	return m.listSourcesFn(ctx, viewerTeamID)
}

func (m *mockShareService) ListEvents(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID, from, to *time.Time) ([]calendarshare.RedactedEventRow, error) {
	return m.listEventsFn(ctx, ownerTeamID, viewerTeamID, from, to)
}

var (
	ownerTeamID  = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	viewerTeamID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)

// ─── ListCalendarShares ─────────────────────────────────────────────────────

func TestHandler_ListCalendarShares_Success(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		listGrantsFn: func(_ context.Context, gotOwner uuid.UUID) ([]calendarshare.ShareRow, error) {
			assert.Equal(t, ownerTeamID, gotOwner)
			return []calendarshare.ShareRow{{TeamId: viewerTeamID, TeamName: "Viewers", CreatedAt: time.Now()}}, nil
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	resp, err := h.ListCalendarShares(context.Background(), gen.ListCalendarSharesRequestObject{TeamId: ownerTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListCalendarSharesResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Viewers")
}

func TestHandler_ListCalendarShares_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		listGrantsFn: func(context.Context, uuid.UUID) ([]calendarshare.ShareRow, error) { return nil, errors.New("db error") },
	}
	h := calendarshare.NewHandler(svc, slog.Default())
	_, err := h.ListCalendarShares(context.Background(), gen.ListCalendarSharesRequestObject{TeamId: ownerTeamID})
	require.Error(t, err)
}

// ─── CreateCalendarShare ────────────────────────────────────────────────────

func TestHandler_CreateCalendarShare_Success(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		grantFn: func(_ context.Context, gotOwner, gotViewer uuid.UUID) (*calendarshare.ShareRow, error) {
			assert.Equal(t, ownerTeamID, gotOwner)
			assert.Equal(t, viewerTeamID, gotViewer)
			return &calendarshare.ShareRow{TeamId: viewerTeamID, TeamName: "Viewers", CreatedAt: time.Now()}, nil
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.CreateCalendarShareRequestObject{
		TeamId: ownerTeamID,
		Body:   &gen.CreateCalendarShareJSONRequestBody{ViewerTeamId: viewerTeamID},
	}
	resp, err := h.CreateCalendarShare(context.Background(), req)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreateCalendarShareResponse(w))
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Viewers")
}

func TestHandler_CreateCalendarShare_MissingBody(t *testing.T) {
	t.Parallel()
	h := calendarshare.NewHandler(&mockShareService{}, slog.Default())
	_, err := h.CreateCalendarShare(context.Background(), gen.CreateCalendarShareRequestObject{TeamId: ownerTeamID})
	require.Error(t, err)
}

func TestHandler_CreateCalendarShare_TeamNotFoundMapsTo404(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		grantFn: func(context.Context, uuid.UUID, uuid.UUID) (*calendarshare.ShareRow, error) {
			return nil, calendarshare.ErrTeamNotFound
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.CreateCalendarShareRequestObject{TeamId: ownerTeamID, Body: &gen.CreateCalendarShareJSONRequestBody{ViewerTeamId: viewerTeamID}}
	resp, err := h.CreateCalendarShare(context.Background(), req)
	require.NoError(t, err, "an unknown viewer team is a typed 404 response, not a Go error")

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitCreateCalendarShareResponse(w))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CreateCalendarShare_SelfShareRejected(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		grantFn: func(context.Context, uuid.UUID, uuid.UUID) (*calendarshare.ShareRow, error) {
			return nil, calendarshare.ErrCannotShareWithSelf
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.CreateCalendarShareRequestObject{TeamId: ownerTeamID, Body: &gen.CreateCalendarShareJSONRequestBody{ViewerTeamId: ownerTeamID}}
	_, err := h.CreateCalendarShare(context.Background(), req)
	require.Error(t, err)
}

// ─── DeleteCalendarShare ────────────────────────────────────────────────────

func TestHandler_DeleteCalendarShare_Success(t *testing.T) {
	t.Parallel()
	called := false
	svc := &mockShareService{
		revokeFn: func(_ context.Context, gotOwner, gotViewer uuid.UUID) error {
			assert.Equal(t, ownerTeamID, gotOwner)
			assert.Equal(t, viewerTeamID, gotViewer)
			called = true
			return nil
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	resp, err := h.DeleteCalendarShare(context.Background(), gen.DeleteCalendarShareRequestObject{TeamId: ownerTeamID, ViewerTeamId: viewerTeamID})
	require.NoError(t, err)
	assert.True(t, called)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitDeleteCalendarShareResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ─── ListSharedCalendarSources ──────────────────────────────────────────────

func TestHandler_ListSharedCalendarSources_Success(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		listSourcesFn: func(_ context.Context, gotViewer uuid.UUID) ([]calendarshare.ShareRow, error) {
			assert.Equal(t, viewerTeamID, gotViewer)
			return []calendarshare.ShareRow{{TeamId: ownerTeamID, TeamName: "Owners"}}, nil
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	resp, err := h.ListSharedCalendarSources(context.Background(), gen.ListSharedCalendarSourcesRequestObject{TeamId: viewerTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListSharedCalendarSourcesResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Owners")
}

// ─── ListSharedCalendarEvents ───────────────────────────────────────────────

func TestHandler_ListSharedCalendarEvents_Success(t *testing.T) {
	t.Parallel()
	eventID := uuid.New()
	svc := &mockShareService{
		listEventsFn: func(_ context.Context, gotOwner, gotViewer uuid.UUID, _, _ *time.Time) ([]calendarshare.RedactedEventRow, error) {
			assert.Equal(t, ownerTeamID, gotOwner)
			assert.Equal(t, viewerTeamID, gotViewer)
			return []calendarshare.RedactedEventRow{{Id: eventID, Type: "training", Title: "Training", Date: time.Now()}}, nil
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.ListSharedCalendarEventsRequestObject{TeamId: viewerTeamID, OwnerTeamId: ownerTeamID}
	resp, err := h.ListSharedCalendarEvents(context.Background(), req)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListSharedCalendarEventsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Training")
	// The redacted projection must never leak attendance/participant/comment/
	// note fields -- this is a schedule-only view by construction (see
	// RedactedEventRow), but assert on the wire body too as a regression
	// guard against a future field accidentally being added to the mapping.
	assert.NotContains(t, w.Body.String(), "attendance")
	assert.NotContains(t, w.Body.String(), "note")
}

func TestHandler_ListSharedCalendarEvents_MapsMultiDayEndDate(t *testing.T) {
	t.Parallel()
	eventID := uuid.New()
	end := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	svc := &mockShareService{
		listEventsFn: func(context.Context, uuid.UUID, uuid.UUID, *time.Time, *time.Time) ([]calendarshare.RedactedEventRow, error) {
			return []calendarshare.RedactedEventRow{{
				Id: eventID, Type: "training", Title: "Trainingslager",
				Date: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), EndDate: &end,
			}}, nil
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.ListSharedCalendarEventsRequestObject{TeamId: viewerTeamID, OwnerTeamId: ownerTeamID}
	resp, err := h.ListSharedCalendarEvents(context.Background(), req)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListSharedCalendarEventsResponse(w))
	assert.Contains(t, w.Body.String(), `"multiDayEndDate":"2026-06-12"`)
}

func TestHandler_ListSharedCalendarEvents_NoGrantMapsTo404(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		listEventsFn: func(context.Context, uuid.UUID, uuid.UUID, *time.Time, *time.Time) ([]calendarshare.RedactedEventRow, error) {
			return nil, calendarshare.ErrNoGrant
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.ListSharedCalendarEventsRequestObject{TeamId: viewerTeamID, OwnerTeamId: ownerTeamID}
	resp, err := h.ListSharedCalendarEvents(context.Background(), req)
	require.NoError(t, err, "no active grant is a typed 404 response, not a Go error")

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListSharedCalendarEventsResponse(w))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ListSharedCalendarEvents_UnexpectedError(t *testing.T) {
	t.Parallel()
	svc := &mockShareService{
		listEventsFn: func(context.Context, uuid.UUID, uuid.UUID, *time.Time, *time.Time) ([]calendarshare.RedactedEventRow, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := calendarshare.NewHandler(svc, slog.Default())

	req := gen.ListSharedCalendarEventsRequestObject{TeamId: viewerTeamID, OwnerTeamId: ownerTeamID}
	_, err := h.ListSharedCalendarEvents(context.Background(), req)
	require.Error(t, err)
}
