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
	upsertFn            func(ctx context.Context, userID uuid.UUID, sub push.Subscription) error
	deleteFn            func(ctx context.Context, userID uuid.UUID, endpoint string) error
	getPreferencesFn    func(ctx context.Context, teamID, userID uuid.UUID) (push.CategoryPreferences, error)
	upsertPreferencesFn func(ctx context.Context, teamID, userID uuid.UUID, prefs push.CategoryPreferences) error
}

func (m *mockRepo) Upsert(ctx context.Context, userID uuid.UUID, sub push.Subscription) error {
	return m.upsertFn(ctx, userID, sub)
}

func (m *mockRepo) Delete(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return m.deleteFn(ctx, userID, endpoint)
}

func (m *mockRepo) GetPreferences(ctx context.Context, teamID, userID uuid.UUID) (push.CategoryPreferences, error) {
	return m.getPreferencesFn(ctx, teamID, userID)
}

func (m *mockRepo) UpsertPreferences(ctx context.Context, teamID, userID uuid.UUID, prefs push.CategoryPreferences) error {
	return m.upsertPreferencesFn(ctx, teamID, userID, prefs)
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

func TestService_GetPreferences_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	want := push.CategoryPreferences{Attendance: true, Events: false, News: true, Polls: false, Absence: true}
	repo := &mockRepo{
		getPreferencesFn: func(_ context.Context, tID, uID uuid.UUID) (push.CategoryPreferences, error) {
			assert.Equal(t, teamID, tID)
			assert.Equal(t, userID, uID)
			return want, nil
		},
	}

	svc := push.NewService(repo)
	got, err := svc.GetPreferences(context.Background(), teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_GetPreferences_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		getPreferencesFn: func(context.Context, uuid.UUID, uuid.UUID) (push.CategoryPreferences, error) {
			return push.CategoryPreferences{}, wantErr
		},
	}

	svc := push.NewService(repo)
	_, err := svc.GetPreferences(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_SetPreferences_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	prefs := push.CategoryPreferences{Attendance: false, Events: true, News: false, Polls: true, Absence: false}
	var gotTeamID, gotUserID uuid.UUID
	var gotPrefs push.CategoryPreferences
	repo := &mockRepo{
		upsertPreferencesFn: func(_ context.Context, tID, uID uuid.UUID, p push.CategoryPreferences) error {
			gotTeamID, gotUserID, gotPrefs = tID, uID, p
			return nil
		},
	}

	svc := push.NewService(repo)
	require.NoError(t, svc.SetPreferences(context.Background(), teamID, userID, prefs))
	assert.Equal(t, teamID, gotTeamID)
	assert.Equal(t, userID, gotUserID)
	assert.Equal(t, prefs, gotPrefs)
}

func TestService_SetPreferences_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		upsertPreferencesFn: func(context.Context, uuid.UUID, uuid.UUID, push.CategoryPreferences) error { return wantErr },
	}

	svc := push.NewService(repo)
	err := svc.SetPreferences(context.Background(), uuid.New(), uuid.New(), push.CategoryPreferences{})
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}
