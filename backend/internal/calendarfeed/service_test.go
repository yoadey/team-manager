package calendarfeed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/calendarfeed"
	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mocks ──────────────────────────────────────────────────────────────────

type mockTokens struct {
	issueTokenFn        func(ctx context.Context, userID, teamID uuid.UUID) (string, error)
	revokeFn            func(ctx context.Context, userID, teamID uuid.UUID) error
	findActiveByTokenFn func(ctx context.Context, token string) (*calendarfeed.TokenRow, error)
}

func (m *mockTokens) IssueToken(ctx context.Context, userID, teamID uuid.UUID) (string, error) {
	return m.issueTokenFn(ctx, userID, teamID)
}

func (m *mockTokens) Revoke(ctx context.Context, userID, teamID uuid.UUID) error {
	return m.revokeFn(ctx, userID, teamID)
}

func (m *mockTokens) FindActiveByToken(ctx context.Context, token string) (*calendarfeed.TokenRow, error) {
	return m.findActiveByTokenFn(ctx, token)
}

type mockMembership struct {
	isMember bool
	err      error
}

func (m *mockMembership) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return m.isMember, m.err
}

type mockPerms struct {
	perms teams.PermissionsJSON
	err   error
}

func (m *mockPerms) GetPermissions(context.Context, uuid.UUID, uuid.UUID) (teams.PermissionsJSON, error) {
	return m.perms, m.err
}

type mockTeamRepo struct {
	team *teams.TeamRow
	err  error
}

func (m *mockTeamRepo) GetTeam(context.Context, string) (*teams.TeamRow, error) {
	return m.team, m.err
}

type mockEventLister struct {
	events []events.EventRow
	err    error
}

func (m *mockEventLister) ListEvents(context.Context, string, gen.ListEventsParamsScope, int, *events.ListCursor) ([]events.EventRow, error) {
	return m.events, m.err
}

func readAccessPerms() teams.PermissionsJSON {
	return teams.PermissionsJSON{Events: "read"}
}

// ─── IssueToken / RevokeToken ───────────────────────────────────────────────

func TestService_IssueToken_BuildsURLFromPublicBaseURL(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	tokens := &mockTokens{
		issueTokenFn: func(_ context.Context, gotUser, gotTeam uuid.UUID) (string, error) {
			assert.Equal(t, userID, gotUser)
			assert.Equal(t, teamID, gotTeam)
			return "abc123", nil
		},
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, "https://app.example.com")
	url, err := svc.IssueToken(context.Background(), userID, teamID)
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com/api/v1/calendar-feed/abc123.ics", url)
}

func TestService_IssueToken_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	tokens := &mockTokens{
		issueTokenFn: func(context.Context, uuid.UUID, uuid.UUID) (string, error) { return "", wantErr },
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, "https://app.example.com")
	_, err := svc.IssueToken(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_RevokeToken_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	called := false
	tokens := &mockTokens{
		revokeFn: func(_ context.Context, gotUser, gotTeam uuid.UUID) error {
			assert.Equal(t, userID, gotUser)
			assert.Equal(t, teamID, gotTeam)
			called = true
			return nil
		},
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, "https://app.example.com")
	require.NoError(t, svc.RevokeToken(context.Background(), userID, teamID))
	assert.True(t, called)
}

// ─── ServeFeed ──────────────────────────────────────────────────────────────

func activeTokenRow(userID, teamID uuid.UUID) *calendarfeed.TokenRow {
	return &calendarfeed.TokenRow{Id: uuid.New(), UserId: userID, TeamId: teamID, Token: "tok", CreatedAt: time.Now()}
}

func TestService_ServeFeed_UnknownOrRevokedToken(t *testing.T) {
	t.Parallel()

	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) { return nil, pgx.ErrNoRows },
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, "https://app.example.com")
	_, err := svc.ServeFeed(context.Background(), "unknown")
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrFeedUnavailable)
}

func TestService_ServeFeed_TokenHolderNoLongerMember(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) {
			return activeTokenRow(userID, teamID), nil
		},
	}
	membership := &mockMembership{isMember: false}

	svc := calendarfeed.NewService(tokens, membership, &mockPerms{perms: readAccessPerms()}, nil, nil, "https://app.example.com")
	_, err := svc.ServeFeed(context.Background(), "tok")
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrFeedUnavailable, "a token holder who has left the team must not be able to fetch the feed")
}

func TestService_ServeFeed_TokenHolderLostEventsPermission(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) {
			return activeTokenRow(userID, teamID), nil
		},
	}
	membership := &mockMembership{isMember: true}
	perms := &mockPerms{perms: teams.PermissionsJSON{Events: "none"}}

	svc := calendarfeed.NewService(tokens, membership, perms, nil, nil, "https://app.example.com")
	_, err := svc.ServeFeed(context.Background(), "tok")
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrFeedUnavailable, "events:none must hide the feed just like it hides the in-app event list")
}

func TestService_ServeFeed_RendersVisibleEvents(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) {
			return activeTokenRow(userID, teamID), nil
		},
	}
	membership := &mockMembership{isMember: true}
	perms := &mockPerms{perms: readAccessPerms()}
	teamRepo := &mockTeamRepo{team: &teams.TeamRow{Id: teamID, Name: "Meine Mannschaft"}}
	evLister := &mockEventLister{events: []events.EventRow{
		{Id: uuid.New(), Type: "training", Title: "Training", Date: time.Now(), Status: "active"},
	}}

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, evLister, "https://app.example.com")
	ics, err := svc.ServeFeed(context.Background(), "tok")
	require.NoError(t, err)
	assert.Contains(t, string(ics), "X-WR-CALNAME:Meine Mannschaft")
	assert.Contains(t, string(ics), "Training")
}

func TestService_ServeFeed_TeamGone(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) {
			return activeTokenRow(userID, teamID), nil
		},
	}
	membership := &mockMembership{isMember: true}
	perms := &mockPerms{perms: readAccessPerms()}
	teamRepo := &mockTeamRepo{err: pgx.ErrNoRows}

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, nil, "https://app.example.com")
	_, err := svc.ServeFeed(context.Background(), "tok")
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrFeedUnavailable)
}
