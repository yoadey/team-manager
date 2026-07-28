package calendarshare_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/calendarshare"
)

// ─── mocks ──────────────────────────────────────────────────────────────────

type mockRepo struct {
	grantFn               func(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*calendarshare.ShareRow, error)
	revokeFn              func(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error
	listGrantedByOwnerFn  func(ctx context.Context, ownerTeamID uuid.UUID) ([]calendarshare.ShareRow, error)
	listGrantedToViewerFn func(ctx context.Context, viewerTeamID uuid.UUID) ([]calendarshare.ShareRow, error)
	hasGrantFn            func(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (bool, error)
	listRedactedEventsFn  func(ctx context.Context, ownerTeamID uuid.UUID, from, to *time.Time) ([]calendarshare.RedactedEventRow, error)
}

func (m *mockRepo) Grant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*calendarshare.ShareRow, error) {
	return m.grantFn(ctx, ownerTeamID, viewerTeamID)
}

func (m *mockRepo) Revoke(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error {
	return m.revokeFn(ctx, ownerTeamID, viewerTeamID)
}

func (m *mockRepo) ListGrantedByOwner(ctx context.Context, ownerTeamID uuid.UUID) ([]calendarshare.ShareRow, error) {
	return m.listGrantedByOwnerFn(ctx, ownerTeamID)
}

func (m *mockRepo) ListGrantedToViewer(ctx context.Context, viewerTeamID uuid.UUID) ([]calendarshare.ShareRow, error) {
	return m.listGrantedToViewerFn(ctx, viewerTeamID)
}

func (m *mockRepo) HasGrant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (bool, error) {
	return m.hasGrantFn(ctx, ownerTeamID, viewerTeamID)
}

func (m *mockRepo) ListRedactedEvents(ctx context.Context, ownerTeamID uuid.UUID, from, to *time.Time) ([]calendarshare.RedactedEventRow, error) {
	return m.listRedactedEventsFn(ctx, ownerTeamID, from, to)
}

// ─── Grant ──────────────────────────────────────────────────────────────────

func TestService_Grant_RejectsSharingWithSelf(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	svc := calendarshare.NewService(&mockRepo{})
	_, err := svc.Grant(context.Background(), teamID, teamID)
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarshare.ErrCannotShareWithSelf)
}

func TestService_Grant_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	owner, viewer := uuid.New(), uuid.New()
	want := &calendarshare.ShareRow{TeamId: viewer, TeamName: "Viewer Team", CreatedAt: time.Now()}
	repo := &mockRepo{
		grantFn: func(_ context.Context, gotOwner, gotViewer uuid.UUID) (*calendarshare.ShareRow, error) {
			assert.Equal(t, owner, gotOwner)
			assert.Equal(t, viewer, gotViewer)
			return want, nil
		},
	}

	svc := calendarshare.NewService(repo)
	got, err := svc.Grant(context.Background(), owner, viewer)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_Grant_PropagatesTeamNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		grantFn: func(context.Context, uuid.UUID, uuid.UUID) (*calendarshare.ShareRow, error) {
			return nil, calendarshare.ErrTeamNotFound
		},
	}

	svc := calendarshare.NewService(repo)
	_, err := svc.Grant(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarshare.ErrTeamNotFound)
}

// ─── Revoke ─────────────────────────────────────────────────────────────────

func TestService_Revoke_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	owner, viewer := uuid.New(), uuid.New()
	called := false
	repo := &mockRepo{
		revokeFn: func(_ context.Context, gotOwner, gotViewer uuid.UUID) error {
			assert.Equal(t, owner, gotOwner)
			assert.Equal(t, viewer, gotViewer)
			called = true
			return nil
		},
	}

	svc := calendarshare.NewService(repo)
	require.NoError(t, svc.Revoke(context.Background(), owner, viewer))
	assert.True(t, called)
}

// ─── ListEvents ─────────────────────────────────────────────────────────────

func TestService_ListEvents_NoGrantReturnsErrNoGrant(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		hasGrantFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil },
	}

	svc := calendarshare.NewService(repo)
	_, err := svc.ListEvents(context.Background(), uuid.New(), uuid.New(), nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarshare.ErrNoGrant)
}

func TestService_ListEvents_RevokedGrantTakesEffectImmediately(t *testing.T) {
	t.Parallel()

	// HasGrant is re-checked on every call rather than cached, so a grant
	// revoked between two reads must deny the second one.
	calls := 0
	repo := &mockRepo{
		hasGrantFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
			calls++
			return calls == 1, nil // granted on the first call, revoked by the second
		},
		listRedactedEventsFn: func(context.Context, uuid.UUID, *time.Time, *time.Time) ([]calendarshare.RedactedEventRow, error) {
			return nil, nil
		},
	}

	svc := calendarshare.NewService(repo)
	_, err := svc.ListEvents(context.Background(), uuid.New(), uuid.New(), nil, nil)
	require.NoError(t, err)

	_, err = svc.ListEvents(context.Background(), uuid.New(), uuid.New(), nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarshare.ErrNoGrant)
}

func TestService_ListEvents_ReturnsRedactedProjection(t *testing.T) {
	t.Parallel()

	owner := uuid.New()
	want := []calendarshare.RedactedEventRow{
		{Id: uuid.New(), Type: "training", Title: "Training", Date: time.Now()},
	}
	repo := &mockRepo{
		hasGrantFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil },
		listRedactedEventsFn: func(_ context.Context, gotOwner uuid.UUID, _, _ *time.Time) ([]calendarshare.RedactedEventRow, error) {
			assert.Equal(t, owner, gotOwner)
			return want, nil
		},
	}

	svc := calendarshare.NewService(repo)
	got, err := svc.ListEvents(context.Background(), owner, uuid.New(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_ListEvents_PropagatesHasGrantError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		hasGrantFn: func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, wantErr },
	}

	svc := calendarshare.NewService(repo)
	_, err := svc.ListEvents(context.Background(), uuid.New(), uuid.New(), nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

// ─── ListGrants / ListSources ───────────────────────────────────────────────

func TestService_ListGrants_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	owner := uuid.New()
	want := []calendarshare.ShareRow{{TeamId: uuid.New(), TeamName: "B"}}
	repo := &mockRepo{
		listGrantedByOwnerFn: func(_ context.Context, gotOwner uuid.UUID) ([]calendarshare.ShareRow, error) {
			assert.Equal(t, owner, gotOwner)
			return want, nil
		},
	}

	svc := calendarshare.NewService(repo)
	got, err := svc.ListGrants(context.Background(), owner)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_ListSources_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	viewer := uuid.New()
	want := []calendarshare.ShareRow{{TeamId: uuid.New(), TeamName: "A"}}
	repo := &mockRepo{
		listGrantedToViewerFn: func(_ context.Context, gotViewer uuid.UUID) ([]calendarshare.ShareRow, error) {
			assert.Equal(t, viewer, gotViewer)
			return want, nil
		},
	}

	svc := calendarshare.NewService(repo)
	got, err := svc.ListSources(context.Background(), viewer)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
