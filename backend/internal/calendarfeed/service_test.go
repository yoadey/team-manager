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
	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mocks ──────────────────────────────────────────────────────────────────

type mockTokens struct {
	issueTokenFn        func(ctx context.Context, userID, teamID uuid.UUID) (string, error)
	revokeFn            func(ctx context.Context, userID, teamID uuid.UUID) error
	findActiveByTokenFn func(ctx context.Context, token string) (*calendarfeed.TokenRow, error)
	getSettingsFn       func(ctx context.Context, userID, teamID uuid.UUID) ([]string, bool, error)
	updateSettingsFn    func(ctx context.Context, userID, teamID uuid.UUID, types []string, includeBirthdays bool) error
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

func (m *mockTokens) GetSettings(ctx context.Context, userID, teamID uuid.UUID) (types []string, includeBirthdays bool, err error) {
	return m.getSettingsFn(ctx, userID, teamID)
}

func (m *mockTokens) UpdateSettings(ctx context.Context, userID, teamID uuid.UUID, types []string, includeBirthdays bool) error {
	return m.updateSettingsFn(ctx, userID, teamID, types, includeBirthdays)
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

type mockMemberLister struct {
	members []members.MemberRow
	err     error
}

func (m *mockMemberLister) ListMembers(context.Context, string, int, *members.ListCursor) ([]members.MemberRow, error) {
	return m.members, m.err
}

func readAccessPerms() teams.PermissionsJSON {
	return teams.PermissionsJSON{Events: "read"}
}

func readAccessEventsAndMembersPerms() teams.PermissionsJSON {
	return teams.PermissionsJSON{Events: "read", Members: "read"}
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

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
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

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
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

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
	require.NoError(t, svc.RevokeToken(context.Background(), userID, teamID))
	assert.True(t, called)
}

// ─── ServeFeed ──────────────────────────────────────────────────────────────

func activeTokenRow(userID, teamID uuid.UUID) *calendarfeed.TokenRow {
	return &calendarfeed.TokenRow{
		Id: uuid.New(), UserId: userID, TeamId: teamID, Token: "tok", CreatedAt: time.Now(),
		Types: []string{"training", "auftritt", "event"}, IncludeBirthdays: true,
	}
}

func TestService_ServeFeed_UnknownOrRevokedToken(t *testing.T) {
	t.Parallel()

	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) { return nil, pgx.ErrNoRows },
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
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

	svc := calendarfeed.NewService(tokens, membership, &mockPerms{perms: readAccessPerms()}, nil, nil, nil, "https://app.example.com")
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

	svc := calendarfeed.NewService(tokens, membership, perms, nil, nil, nil, "https://app.example.com")
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

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, evLister, nil, "https://app.example.com")
	ics, err := svc.ServeFeed(context.Background(), "tok")
	require.NoError(t, err)
	assert.Contains(t, string(ics), "X-WR-CALNAME:Meine Mannschaft")
	assert.Contains(t, string(ics), "Training")
}

func TestService_ServeFeed_FiltersEventsByConfiguredTypes(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	row := activeTokenRow(userID, teamID)
	row.Types = []string{"auftritt"}
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) { return row, nil },
	}
	membership := &mockMembership{isMember: true}
	perms := &mockPerms{perms: readAccessPerms()}
	teamRepo := &mockTeamRepo{team: &teams.TeamRow{Id: teamID, Name: "Team"}}
	evLister := &mockEventLister{events: []events.EventRow{
		{Id: uuid.New(), Type: "training", Title: "Training", Date: time.Now(), Status: "active"},
		{Id: uuid.New(), Type: "auftritt", Title: "Turnier", Date: time.Now(), Status: "active"},
	}}

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, evLister, nil, "https://app.example.com")
	ics, err := svc.ServeFeed(context.Background(), "tok")
	require.NoError(t, err)
	assert.Contains(t, string(ics), "Turnier")
	assert.NotContains(t, string(ics), "Training")
}

func TestService_ServeFeed_IncludesBirthdaysWhenPermittedAndEnabled(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	row := activeTokenRow(userID, teamID)
	row.Types = nil
	row.IncludeBirthdays = true
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) { return row, nil },
	}
	membership := &mockMembership{isMember: true}
	perms := &mockPerms{perms: readAccessEventsAndMembersPerms()}
	teamRepo := &mockTeamRepo{team: &teams.TeamRow{Id: teamID, Name: "Team"}}
	evLister := &mockEventLister{}
	birthday := time.Date(2000, 5, 17, 0, 0, 0, 0, time.UTC)
	memberLister := &mockMemberLister{members: []members.MemberRow{
		{UserID: uuid.New(), Name: "Ada Lovelace", Birthday: &birthday},
		{UserID: uuid.New(), Name: "No Birthday"},
	}}

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, evLister, memberLister, "https://app.example.com")
	ics, err := svc.ServeFeed(context.Background(), "tok")
	require.NoError(t, err)
	assert.Contains(t, string(ics), "Geburtstag: Ada Lovelace")
	assert.Contains(t, string(ics), "RRULE:FREQ=YEARLY")
	assert.NotContains(t, string(ics), "No Birthday")
}

func TestService_ServeFeed_ExcludesBirthdaysWithoutMembersReadAccess(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	row := activeTokenRow(userID, teamID)
	tokens := &mockTokens{
		findActiveByTokenFn: func(context.Context, string) (*calendarfeed.TokenRow, error) { return row, nil },
	}
	membership := &mockMembership{isMember: true}
	// events:read but no members access -- birthdays live behind the
	// members module, so they must not leak into a feed the caller can
	// otherwise see.
	perms := &mockPerms{perms: readAccessPerms()}
	teamRepo := &mockTeamRepo{team: &teams.TeamRow{Id: teamID, Name: "Team"}}
	evLister := &mockEventLister{}
	memberLister := &mockMemberLister{err: assert.AnError}

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, evLister, memberLister, "https://app.example.com")
	ics, err := svc.ServeFeed(context.Background(), "tok")
	require.NoError(t, err, "memberLister must not even be called without members read access")
	assert.NotContains(t, string(ics), "Geburtstag")
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

	svc := calendarfeed.NewService(tokens, membership, perms, teamRepo, nil, nil, "https://app.example.com")
	_, err := svc.ServeFeed(context.Background(), "tok")
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrFeedUnavailable)
}

// ─── GetSettings / UpdateSettings ───────────────────────────────────────────

func TestService_GetSettings_DefaultsWhenNoActiveToken(t *testing.T) {
	t.Parallel()

	tokens := &mockTokens{
		getSettingsFn: func(context.Context, uuid.UUID, uuid.UUID) ([]string, bool, error) {
			return nil, false, pgx.ErrNoRows
		},
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
	types, includeBirthdays, err := svc.GetSettings(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"training", "auftritt", "event"}, types)
	assert.True(t, includeBirthdays)
}

func TestService_GetSettings_ReturnsStoredSelection(t *testing.T) {
	t.Parallel()

	tokens := &mockTokens{
		getSettingsFn: func(context.Context, uuid.UUID, uuid.UUID) ([]string, bool, error) {
			return []string{"auftritt"}, false, nil
		},
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
	types, includeBirthdays, err := svc.GetSettings(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, []string{"auftritt"}, types)
	assert.False(t, includeBirthdays)
}

func TestService_UpdateSettings_RejectsInvalidType(t *testing.T) {
	t.Parallel()

	svc := calendarfeed.NewService(&mockTokens{}, nil, nil, nil, nil, nil, "https://app.example.com")
	err := svc.UpdateSettings(context.Background(), uuid.New(), uuid.New(), []string{"not-a-real-type"}, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrInvalidEventType)
}

func TestService_UpdateSettings_NoActiveToken(t *testing.T) {
	t.Parallel()

	tokens := &mockTokens{
		updateSettingsFn: func(context.Context, uuid.UUID, uuid.UUID, []string, bool) error { return pgx.ErrNoRows },
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
	err := svc.UpdateSettings(context.Background(), uuid.New(), uuid.New(), []string{"training"}, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarfeed.ErrNoActiveToken)
}

func TestService_UpdateSettings_Success(t *testing.T) {
	t.Parallel()

	userID, teamID := uuid.New(), uuid.New()
	var gotTypes []string
	var gotIncludeBirthdays bool
	tokens := &mockTokens{
		updateSettingsFn: func(_ context.Context, gotUser, gotTeam uuid.UUID, types []string, includeBirthdays bool) error {
			assert.Equal(t, userID, gotUser)
			assert.Equal(t, teamID, gotTeam)
			gotTypes = types
			gotIncludeBirthdays = includeBirthdays
			return nil
		},
	}

	svc := calendarfeed.NewService(tokens, nil, nil, nil, nil, nil, "https://app.example.com")
	require.NoError(t, svc.UpdateSettings(context.Background(), userID, teamID, []string{"training", "event"}, false))
	assert.Equal(t, []string{"training", "event"}, gotTypes)
	assert.False(t, gotIncludeBirthdays)
}
