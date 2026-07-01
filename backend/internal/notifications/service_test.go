package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/notifications"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listByTeamAndUserFn func(ctx context.Context, teamID, userID uuid.UUID) ([]*notifications.NotificationRow, error)
	markSeenFn          func(ctx context.Context, teamID, userID uuid.UUID) error
}

func (m *mockRepo) ListByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) ([]*notifications.NotificationRow, error) {
	return m.listByTeamAndUserFn(ctx, teamID, userID)
}

func (m *mockRepo) MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error {
	return m.markSeenFn(ctx, teamID, userID)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_List_CountsOnlyUnread(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	title := "Training added"
	rows := []*notifications.NotificationRow{
		{Id: uuid.New(), TeamId: teamID, Type: "event_created", Title: &title, CreatedAt: time.Now(), Unread: true},
		{Id: uuid.New(), TeamId: teamID, Type: "event_created", Title: &title, CreatedAt: time.Now(), Unread: true},
		{Id: uuid.New(), TeamId: teamID, Type: "event_created", Title: &title, CreatedAt: time.Now(), Unread: false},
	}
	repo := &mockRepo{
		listByTeamAndUserFn: func(_ context.Context, gotTeamID, gotUserID uuid.UUID) ([]*notifications.NotificationRow, error) {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, userID, gotUserID)
			return rows, nil
		},
	}

	svc := notifications.NewService(repo)
	result, err := svc.List(context.Background(), teamID, userID)
	require.NoError(t, err)
	assert.Len(t, result.Items, 3)
	assert.Equal(t, 2, result.UnreadCount, "unread count must only count rows with Unread=true")
}

func TestService_List_EmptyResult(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		listByTeamAndUserFn: func(context.Context, uuid.UUID, uuid.UUID) ([]*notifications.NotificationRow, error) {
			return nil, nil
		},
	}

	svc := notifications.NewService(repo)
	result, err := svc.List(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.UnreadCount)
}

func TestService_List_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		listByTeamAndUserFn: func(context.Context, uuid.UUID, uuid.UUID) ([]*notifications.NotificationRow, error) {
			return nil, wantErr
		},
	}

	svc := notifications.NewService(repo)
	_, err := svc.List(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_MarkSeen(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	called := false
	repo := &mockRepo{
		markSeenFn: func(_ context.Context, gotTeamID, gotUserID uuid.UUID) error {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, userID, gotUserID)
			called = true
			return nil
		},
	}

	svc := notifications.NewService(repo)
	err := svc.MarkSeen(context.Background(), teamID, userID)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_MarkSeen_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		markSeenFn: func(context.Context, uuid.UUID, uuid.UUID) error { return wantErr },
	}

	svc := notifications.NewService(repo)
	err := svc.MarkSeen(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
