package push_test

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

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/push"
)

type mockPushService struct {
	registerFn   func(ctx context.Context, userID uuid.UUID, sub push.Subscription) error
	unregisterFn func(ctx context.Context, userID uuid.UUID, endpoint string) error
}

func (m *mockPushService) Register(ctx context.Context, userID uuid.UUID, sub push.Subscription) error {
	return m.registerFn(ctx, userID, sub)
}

func (m *mockPushService) Unregister(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return m.unregisterFn(ctx, userID, endpoint)
}

var pushUserID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

func pushAuthedCtx() context.Context {
	user := &auth.UserRow{Id: pushUserID, Name: "Test User", Email: "test@example.com", AvatarColor: "#6366f1", CreatedAt: time.Now()}
	return auth.ContextWithUser(context.Background(), user)
}

func TestHandler_RegisterPushSubscription_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := push.NewHandler(&mockPushService{}, slog.Default())
	_, err := h.RegisterPushSubscription(context.Background(), gen.RegisterPushSubscriptionRequestObject{})
	require.Error(t, err)
}

func TestHandler_RegisterPushSubscription_MissingBody(t *testing.T) {
	t.Parallel()
	h := push.NewHandler(&mockPushService{}, slog.Default())
	_, err := h.RegisterPushSubscription(pushAuthedCtx(), gen.RegisterPushSubscriptionRequestObject{Body: nil})
	require.Error(t, err)
}

func TestHandler_RegisterPushSubscription_Success(t *testing.T) {
	t.Parallel()
	var gotUserID uuid.UUID
	var gotSub push.Subscription
	svc := &mockPushService{
		registerFn: func(_ context.Context, userID uuid.UUID, sub push.Subscription) error {
			gotUserID, gotSub = userID, sub
			return nil
		},
	}
	h := push.NewHandler(svc, slog.Default())

	body := gen.PushSubscriptionRequest{Endpoint: "https://push.example/abc"}
	body.Keys.P256dh = "p256dh"
	body.Keys.Auth = "auth"

	resp, err := h.RegisterPushSubscription(pushAuthedCtx(), gen.RegisterPushSubscriptionRequestObject{Body: &body})
	require.NoError(t, err)
	assert.Equal(t, pushUserID, gotUserID)
	assert.Equal(t, push.Subscription{Endpoint: "https://push.example/abc", P256dh: "p256dh", AuthKey: "auth"}, gotSub)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitRegisterPushSubscriptionResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_RegisterPushSubscription_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockPushService{
		registerFn: func(context.Context, uuid.UUID, push.Subscription) error { return errors.New("db error") },
	}
	h := push.NewHandler(svc, slog.Default())

	body := gen.PushSubscriptionRequest{Endpoint: "https://push.example/abc"}
	body.Keys.P256dh = "p256dh"
	body.Keys.Auth = "auth"

	_, err := h.RegisterPushSubscription(pushAuthedCtx(), gen.RegisterPushSubscriptionRequestObject{Body: &body})
	require.Error(t, err)
}

func TestHandler_DeletePushSubscription_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := push.NewHandler(&mockPushService{}, slog.Default())
	_, err := h.DeletePushSubscription(context.Background(), gen.DeletePushSubscriptionRequestObject{})
	require.Error(t, err)
}

func TestHandler_DeletePushSubscription_Success(t *testing.T) {
	t.Parallel()
	var gotUserID uuid.UUID
	var gotEndpoint string
	svc := &mockPushService{
		unregisterFn: func(_ context.Context, userID uuid.UUID, endpoint string) error {
			gotUserID, gotEndpoint = userID, endpoint
			return nil
		},
	}
	h := push.NewHandler(svc, slog.Default())

	resp, err := h.DeletePushSubscription(pushAuthedCtx(), gen.DeletePushSubscriptionRequestObject{
		Params: gen.DeletePushSubscriptionParams{Endpoint: "https://push.example/abc"},
	})
	require.NoError(t, err)
	assert.Equal(t, pushUserID, gotUserID)
	assert.Equal(t, "https://push.example/abc", gotEndpoint)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitDeletePushSubscriptionResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
}
