package push_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/push"
)

type mockRepo struct {
	upsertFn func(ctx context.Context, userID uuid.UUID, sub push.Subscription) error
	deleteFn func(ctx context.Context, userID uuid.UUID, endpoint string) error
}

func (m *mockRepo) Upsert(ctx context.Context, userID uuid.UUID, sub push.Subscription) error {
	return m.upsertFn(ctx, userID, sub)
}

func (m *mockRepo) Delete(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return m.deleteFn(ctx, userID, endpoint)
}

func TestService_Register_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sub := push.Subscription{Endpoint: "https://push.example/abc", P256dh: "p256dh", AuthKey: "auth"}
	var gotUserID uuid.UUID
	var gotSub push.Subscription
	repo := &mockRepo{
		upsertFn: func(_ context.Context, u uuid.UUID, s push.Subscription) error {
			gotUserID, gotSub = u, s
			return nil
		},
	}

	svc := push.NewService(repo)
	require.NoError(t, svc.Register(context.Background(), userID, sub))
	assert.Equal(t, userID, gotUserID)
	assert.Equal(t, sub, gotSub)
}

func TestService_Register_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		upsertFn: func(context.Context, uuid.UUID, push.Subscription) error { return wantErr },
	}

	svc := push.NewService(repo)
	err := svc.Register(context.Background(), uuid.New(), push.Subscription{})
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_Unregister_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	const endpoint = "https://push.example/abc"
	var gotUserID uuid.UUID
	var gotEndpoint string
	repo := &mockRepo{
		deleteFn: func(_ context.Context, u uuid.UUID, e string) error {
			gotUserID, gotEndpoint = u, e
			return nil
		},
	}

	svc := push.NewService(repo)
	require.NoError(t, svc.Unregister(context.Background(), userID, endpoint))
	assert.Equal(t, userID, gotUserID)
	assert.Equal(t, endpoint, gotEndpoint)
}

func TestService_Unregister_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		deleteFn: func(context.Context, uuid.UUID, string) error { return wantErr },
	}

	svc := push.NewService(repo)
	err := svc.Unregister(context.Background(), uuid.New(), "endpoint")
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
