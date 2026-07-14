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
	"github.com/yoadey/team-manager/backend/internal/teams"
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

// mockPermChecker returns allWritePerms by default (every module visible),
// unless a test overrides perms directly.
type mockPermChecker struct {
	perms teams.PermissionsJSON
	err   error
}

func (m *mockPermChecker) GetPermissions(context.Context, uuid.UUID, uuid.UUID) (teams.PermissionsJSON, error) {
	return m.perms, m.err
}

func allWritePerms() teams.PermissionsJSON {
	return teams.PermissionsJSON{Events: "write", Members: "write", Finances: "write", News: "write", Polls: "write", Settings: "write"}
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

	svc := notifications.NewService(repo, &mockPermChecker{perms: allWritePerms()})
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

	svc := notifications.NewService(repo, &mockPermChecker{perms: allWritePerms()})
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

	svc := notifications.NewService(repo, &mockPermChecker{perms: allWritePerms()})
	_, err := svc.List(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

// Regression test: /notifications has no module-level RBAC gate of its own
// (it aggregates across events, news, and polls), so without filtering by
// the caller's live permissions here, a member with e.g. events:none would
// still see event/attendance notices in their feed -- the same "none must
// hide the module" property enforced everywhere else in the RBAC system.
func TestService_List_FiltersByModulePermission(t *testing.T) {
	t.Parallel()

	title := "x"
	rows := []*notifications.NotificationRow{
		{Id: uuid.New(), Type: "event_created", Title: &title, CreatedAt: time.Now(), Unread: true},
		{Id: uuid.New(), Type: "attendance", Title: &title, CreatedAt: time.Now(), Unread: true},
		{Id: uuid.New(), Type: "news", Title: &title, CreatedAt: time.Now(), Unread: true},
		{Id: uuid.New(), Type: "poll", Title: &title, CreatedAt: time.Now(), Unread: false},
		{Id: uuid.New(), Type: "absence", Title: &title, CreatedAt: time.Now(), Unread: true},
	}
	repo := &mockRepo{
		listByTeamAndUserFn: func(context.Context, uuid.UUID, uuid.UUID) ([]*notifications.NotificationRow, error) {
			return rows, nil
		},
	}
	perms := &mockPermChecker{perms: teams.PermissionsJSON{Events: "none", News: "read", Polls: "write"}}

	svc := notifications.NewService(repo, perms)
	result, err := svc.List(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)

	// event_created and attendance (events:none) are filtered out; news,
	// poll, and the module-less absence notice remain.
	gotTypes := make([]string, len(result.Items))
	for i, item := range result.Items {
		gotTypes[i] = string(item.Type)
	}
	assert.ElementsMatch(t, []string{"news", "poll", "absence"}, gotTypes)
	assert.Equal(t, 2, result.UnreadCount, "unread count must only count visible rows (news + absence)")
}

func TestService_List_PropagatesPermissionCheckError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("permission lookup failed")
	repo := &mockRepo{
		listByTeamAndUserFn: func(context.Context, uuid.UUID, uuid.UUID) ([]*notifications.NotificationRow, error) {
			return nil, nil
		},
	}
	perms := &mockPermChecker{err: wantErr}

	svc := notifications.NewService(repo, perms)
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

	svc := notifications.NewService(repo, nil)
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

	svc := notifications.NewService(repo, nil)
	err := svc.MarkSeen(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
