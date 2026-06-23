package notifications_test

import (
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
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/notifications"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockNotifService struct {
	list     func(ctx context.Context, teamID, userID uuid.UUID) (gen.NotificationsResult, error)
	markSeen func(ctx context.Context, teamID, userID uuid.UUID) error
}

func (m *mockNotifService) List(ctx context.Context, teamID, userID uuid.UUID) (gen.NotificationsResult, error) {
	return m.list(ctx, teamID, userID)
}
func (m *mockNotifService) MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error {
	return m.markSeen(ctx, teamID, userID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

var (
	notifTeamID = openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	notifUserID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)

func notifAuthedCtx() context.Context {
	user := &auth.UserRow{
		Id:          notifUserID,
		Name:        "Test User",
		Email:       "test@example.com",
		AvatarColor: "#6366f1",
		CreatedAt:   time.Now(),
	}
	return auth.ContextWithUser(context.Background(), user)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestHandler_ListNotifications_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := notifications.NewHandler(&mockNotifService{}, slog.Default())
	_, err := h.ListNotifications(context.Background(), gen.ListNotificationsRequestObject{TeamId: notifTeamID})
	require.Error(t, err)
}

func TestHandler_ListNotifications_Success(t *testing.T) {
	t.Parallel()
	result := gen.NotificationsResult{
		Items:       []gen.AppNotification{},
		UnreadCount: 0,
	}
	svc := &mockNotifService{
		list: func(_ context.Context, teamID, userID uuid.UUID) (gen.NotificationsResult, error) {
			assert.Equal(t, uuid.UUID(notifTeamID), teamID)
			assert.Equal(t, notifUserID, userID)
			return result, nil
		},
	}
	h := notifications.NewHandler(svc, slog.Default())

	resp, err := h.ListNotifications(notifAuthedCtx(), gen.ListNotificationsRequestObject{TeamId: notifTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListNotificationsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var out gen.NotificationsResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&out))
	assert.Equal(t, 0, out.UnreadCount)
}

func TestHandler_ListNotifications_WithUnread(t *testing.T) {
	t.Parallel()
	unread := true
	notifID := openapi_types.UUID(uuid.New())
	result := gen.NotificationsResult{
		Items: []gen.AppNotification{
			{
				Id:        notifID,
				TeamId:    notifTeamID,
				Type:      "news",
				CreatedAt: time.Now(),
				Unread:    &unread,
			},
		},
		UnreadCount: 1,
	}
	svc := &mockNotifService{
		list: func(_ context.Context, _, _ uuid.UUID) (gen.NotificationsResult, error) {
			return result, nil
		},
	}
	h := notifications.NewHandler(svc, slog.Default())

	resp, err := h.ListNotifications(notifAuthedCtx(), gen.ListNotificationsRequestObject{TeamId: notifTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListNotificationsResponse(w))

	var out gen.NotificationsResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&out))
	assert.Equal(t, 1, out.UnreadCount)
	assert.Len(t, out.Items, 1)
}

func TestHandler_ListNotifications_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockNotifService{
		list: func(_ context.Context, _, _ uuid.UUID) (gen.NotificationsResult, error) {
			return gen.NotificationsResult{}, errors.New("db error")
		},
	}
	h := notifications.NewHandler(svc, slog.Default())
	_, err := h.ListNotifications(notifAuthedCtx(), gen.ListNotificationsRequestObject{TeamId: notifTeamID})
	require.Error(t, err)
}

func TestHandler_MarkNotificationsSeen_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := notifications.NewHandler(&mockNotifService{}, slog.Default())
	_, err := h.MarkNotificationsSeen(context.Background(), gen.MarkNotificationsSeenRequestObject{TeamId: notifTeamID})
	require.Error(t, err)
}

func TestHandler_MarkNotificationsSeen_Success(t *testing.T) {
	t.Parallel()
	called := false
	svc := &mockNotifService{
		markSeen: func(_ context.Context, _, _ uuid.UUID) error {
			called = true
			return nil
		},
	}
	h := notifications.NewHandler(svc, slog.Default())

	resp, err := h.MarkNotificationsSeen(notifAuthedCtx(), gen.MarkNotificationsSeenRequestObject{TeamId: notifTeamID})
	require.NoError(t, err)
	assert.True(t, called)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitMarkNotificationsSeenResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}
