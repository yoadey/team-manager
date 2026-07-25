package calendarfeed_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/calendarfeed"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

type mockFeedService struct {
	issueTokenFn  func(ctx context.Context, userID, teamID uuid.UUID) (string, error)
	revokeTokenFn func(ctx context.Context, userID, teamID uuid.UUID) error
	serveFeedFn   func(ctx context.Context, token string) ([]byte, error)
}

func (m *mockFeedService) IssueToken(ctx context.Context, userID, teamID uuid.UUID) (string, error) {
	return m.issueTokenFn(ctx, userID, teamID)
}

func (m *mockFeedService) RevokeToken(ctx context.Context, userID, teamID uuid.UUID) error {
	return m.revokeTokenFn(ctx, userID, teamID)
}

func (m *mockFeedService) ServeFeed(ctx context.Context, token string) ([]byte, error) {
	return m.serveFeedFn(ctx, token)
}

var (
	feedUserID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	feedTeamID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
)

func feedAuthedCtx() context.Context {
	user := &auth.UserRow{Id: feedUserID, Name: "Test User", Email: "test@example.com", AvatarColor: "#6366f1", CreatedAt: time.Now()}
	return auth.ContextWithUser(context.Background(), user)
}

func TestHandler_IssueCalendarFeedToken_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := calendarfeed.NewHandler(&mockFeedService{}, slog.Default())
	_, err := h.IssueCalendarFeedToken(context.Background(), gen.IssueCalendarFeedTokenRequestObject{TeamId: feedTeamID})
	require.Error(t, err)
}

func TestHandler_IssueCalendarFeedToken_Success(t *testing.T) {
	t.Parallel()
	svc := &mockFeedService{
		issueTokenFn: func(_ context.Context, userID, teamID uuid.UUID) (string, error) {
			assert.Equal(t, feedUserID, userID)
			assert.Equal(t, feedTeamID, teamID)
			return "https://app.example.com/api/v1/calendar-feed/abc.ics", nil
		},
	}
	h := calendarfeed.NewHandler(svc, slog.Default())

	resp, err := h.IssueCalendarFeedToken(feedAuthedCtx(), gen.IssueCalendarFeedTokenRequestObject{TeamId: feedTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitIssueCalendarFeedTokenResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "abc.ics")
}

func TestHandler_IssueCalendarFeedToken_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockFeedService{
		issueTokenFn: func(context.Context, uuid.UUID, uuid.UUID) (string, error) { return "", errors.New("db error") },
	}
	h := calendarfeed.NewHandler(svc, slog.Default())
	_, err := h.IssueCalendarFeedToken(feedAuthedCtx(), gen.IssueCalendarFeedTokenRequestObject{TeamId: feedTeamID})
	require.Error(t, err)
}

func TestHandler_RevokeCalendarFeedToken_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := calendarfeed.NewHandler(&mockFeedService{}, slog.Default())
	_, err := h.RevokeCalendarFeedToken(context.Background(), gen.RevokeCalendarFeedTokenRequestObject{TeamId: feedTeamID})
	require.Error(t, err)
}

func TestHandler_RevokeCalendarFeedToken_Success(t *testing.T) {
	t.Parallel()
	called := false
	svc := &mockFeedService{
		revokeTokenFn: func(_ context.Context, userID, teamID uuid.UUID) error {
			assert.Equal(t, feedUserID, userID)
			assert.Equal(t, feedTeamID, teamID)
			called = true
			return nil
		},
	}
	h := calendarfeed.NewHandler(svc, slog.Default())

	resp, err := h.RevokeCalendarFeedToken(feedAuthedCtx(), gen.RevokeCalendarFeedTokenRequestObject{TeamId: feedTeamID})
	require.NoError(t, err)
	assert.True(t, called)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitRevokeCalendarFeedTokenResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestHandler_GetCalendarFeed_NoAuthRequired is a regression test: unlike
// every other handler method in this package, GetCalendarFeed must NOT
// require auth.UserFromContext -- calendar apps polling this URL have no
// session cookie at all.
func TestHandler_GetCalendarFeed_NoAuthRequired(t *testing.T) {
	t.Parallel()
	ics := []byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR")
	svc := &mockFeedService{
		serveFeedFn: func(_ context.Context, token string) ([]byte, error) {
			assert.Equal(t, "sometoken", token)
			return ics, nil
		},
	}
	h := calendarfeed.NewHandler(svc, slog.Default())

	resp, err := h.GetCalendarFeed(context.Background(), gen.GetCalendarFeedRequestObject{Token: "sometoken"})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetCalendarFeedResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/calendar", w.Header().Get("Content-Type"))
	gotBody, err := io.ReadAll(w.Body)
	require.NoError(t, err)
	assert.Equal(t, ics, gotBody)
}

func TestHandler_GetCalendarFeed_UnavailableTokenMapsTo404(t *testing.T) {
	t.Parallel()
	svc := &mockFeedService{
		serveFeedFn: func(context.Context, string) ([]byte, error) { return nil, calendarfeed.ErrFeedUnavailable },
	}
	h := calendarfeed.NewHandler(svc, slog.Default())

	resp, err := h.GetCalendarFeed(context.Background(), gen.GetCalendarFeedRequestObject{Token: "bad"})
	require.NoError(t, err, "an unavailable feed is a typed 404 response, not a Go error")

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetCalendarFeedResponse(w))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetCalendarFeed_UnexpectedError(t *testing.T) {
	t.Parallel()
	svc := &mockFeedService{
		serveFeedFn: func(context.Context, string) ([]byte, error) { return nil, errors.New("db unavailable") },
	}
	h := calendarfeed.NewHandler(svc, slog.Default())
	_, err := h.GetCalendarFeed(context.Background(), gen.GetCalendarFeedRequestObject{Token: "bad"})
	require.Error(t, err)
}
