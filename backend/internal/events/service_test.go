package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ────────────────────────────────────────────────────────

// mockSvcRepo satisfies the unexported eventRepo interface via structural typing.
type mockSvcRepo struct {
	listEventsFn             func(ctx context.Context, teamID, scope string, limit int, cur *events.ListCursor) ([]events.EventRow, error)
	getEventFn               func(ctx context.Context, eventID, teamID string) (*events.EventRow, error)
	createEventFn            func(ctx context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error)
	createSeriesFn           func(ctx context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error)
	updateEventFn            func(ctx context.Context, eventID, teamID string, params *events.UpdateEventParams, scope string) (*events.EventRow, error)
	setStatusFn              func(ctx context.Context, eventID, teamID, status, scope string) (*events.EventRow, error)
	deleteEventFn            func(ctx context.Context, eventID, teamID, scope string) error
	getAttendanceSummaryFn   func(ctx context.Context, eventID, teamID string) (events.EventSummaryData, error)
	getMyAttendanceFn        func(ctx context.Context, eventID, userID, teamID string) (*events.AttendanceDBRow, error)
	getAttendanceSummariesFn func(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]events.EventSummaryData, error)
	getMyAttendancesFn       func(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]events.AttendanceDBRow, error)
	listAttendanceFn         func(ctx context.Context, eventID, teamID string) ([]events.AttendanceEnriched, error)
	getReasonVisibilityCtxFn func(ctx context.Context, teamID, viewerID string) ([]string, []string, error)
	setAttendanceFn          func(ctx context.Context, eventID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*events.AttendanceDBRow, error)
	setNominationFn          func(ctx context.Context, eventID, userID, teamID string, nominated bool) error
	listCommentsFn           func(ctx context.Context, eventID, teamID string, limit, offset int) ([]events.CommentRow, error)
	addCommentFn             func(ctx context.Context, eventID, userID, teamID, text string) (*events.CommentRow, error)
	deleteCommentFn          func(ctx context.Context, commentID, userID, teamID string) error
}

func (m *mockSvcRepo) ListEvents(ctx context.Context, teamID, scope string, limit int, cur *events.ListCursor) ([]events.EventRow, error) {
	return m.listEventsFn(ctx, teamID, scope, limit, cur)
}

func (m *mockSvcRepo) GetEvent(ctx context.Context, eventID, teamID string) (*events.EventRow, error) {
	return m.getEventFn(ctx, eventID, teamID)
}

func (m *mockSvcRepo) CreateEvent(ctx context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error) {
	return m.createEventFn(ctx, teamID, params)
}

func (m *mockSvcRepo) CreateSeries(ctx context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error) {
	return m.createSeriesFn(ctx, teamID, params)
}

func (m *mockSvcRepo) UpdateEvent(ctx context.Context, eventID, teamID string, params *events.UpdateEventParams, scope string) (*events.EventRow, error) {
	return m.updateEventFn(ctx, eventID, teamID, params, scope)
}

func (m *mockSvcRepo) SetStatus(ctx context.Context, eventID, teamID, status, scope string) (*events.EventRow, error) {
	return m.setStatusFn(ctx, eventID, teamID, status, scope)
}

func (m *mockSvcRepo) DeleteEvent(ctx context.Context, eventID, teamID, scope string) error {
	return m.deleteEventFn(ctx, eventID, teamID, scope)
}

func (m *mockSvcRepo) GetAttendanceSummary(ctx context.Context, eventID, teamID string) (events.EventSummaryData, error) {
	return m.getAttendanceSummaryFn(ctx, eventID, teamID)
}

func (m *mockSvcRepo) GetMyAttendance(ctx context.Context, eventID, userID, teamID string) (*events.AttendanceDBRow, error) {
	return m.getMyAttendanceFn(ctx, eventID, userID, teamID)
}

func (m *mockSvcRepo) GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]events.EventSummaryData, error) {
	if m.getAttendanceSummariesFn != nil {
		return m.getAttendanceSummariesFn(ctx, eventIDs)
	}
	return map[uuid.UUID]events.EventSummaryData{}, nil
}

func (m *mockSvcRepo) GetMyAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]events.AttendanceDBRow, error) {
	if m.getMyAttendancesFn != nil {
		return m.getMyAttendancesFn(ctx, eventIDs, userID)
	}
	return map[uuid.UUID]events.AttendanceDBRow{}, nil
}

func (m *mockSvcRepo) ListAttendance(ctx context.Context, eventID, teamID string) ([]events.AttendanceEnriched, error) {
	return m.listAttendanceFn(ctx, eventID, teamID)
}

func (m *mockSvcRepo) GetReasonVisibilityContext(ctx context.Context, teamID, viewerID string) (teamRoleIDs, viewerRoleIDs []string, err error) {
	if m.getReasonVisibilityCtxFn != nil {
		return m.getReasonVisibilityCtxFn(ctx, teamID, viewerID)
	}
	return nil, nil, nil
}

func (m *mockSvcRepo) SetAttendance(ctx context.Context, eventID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*events.AttendanceDBRow, error) {
	return m.setAttendanceFn(ctx, eventID, userID, teamID, status, reason, reasonID, reasonVisibility)
}

func (m *mockSvcRepo) SetNomination(ctx context.Context, eventID, userID, teamID string, nominated bool) error {
	return m.setNominationFn(ctx, eventID, userID, teamID, nominated)
}

func (m *mockSvcRepo) ListComments(ctx context.Context, eventID, teamID string, limit, offset int) ([]events.CommentRow, error) {
	return m.listCommentsFn(ctx, eventID, teamID, limit, offset)
}

func (m *mockSvcRepo) AddComment(ctx context.Context, eventID, userID, teamID, text string) (*events.CommentRow, error) {
	return m.addCommentFn(ctx, eventID, userID, teamID, text)
}

func (m *mockSvcRepo) DeleteComment(ctx context.Context, commentID, userID, teamID string) error {
	return m.deleteCommentFn(ctx, commentID, userID, teamID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func svcMakeEventRow(title string) events.EventRow {
	return events.EventRow{
		Id:     uuid.New(),
		TeamId: uuid.MustParse(testTeamID),
		Type:   "training",
		Title:  title,
		Date:   time.Now().UTC(),
		Status: "active",
	}
}

func zeroSummaryFn(_ context.Context, _, _ string) (events.EventSummaryData, error) {
	return events.EventSummaryData{}, nil
}

func nilMyAttendanceFn(_ context.Context, _, _, _ string) (*events.AttendanceDBRow, error) {
	return nil, nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestEventService_ListEvents_Upcoming(t *testing.T) {
	t.Parallel()

	rows := []events.EventRow{
		svcMakeEventRow("Event 1"),
		svcMakeEventRow("Event 2"),
		svcMakeEventRow("Event 3"),
	}

	capturedScope := ""
	repo := &mockSvcRepo{
		listEventsFn: func(_ context.Context, _, scope string, _ int, _ *events.ListCursor) ([]events.EventRow, error) {
			capturedScope = scope
			return rows, nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	result, next, err := svc.ListEvents(context.Background(), testTeamID, testUserID, "upcoming", "", 50)
	require.NoError(t, err)
	assert.Nil(t, next)
	assert.Len(t, result, 3)
	assert.Equal(t, "upcoming", capturedScope, "scope should be passed through to repository")
}

func TestEventService_CreateEvent_Recurring(t *testing.T) {
	t.Parallel()

	sid := uuid.New()
	eventRows := []events.EventRow{
		svcMakeEventRow("Weekly Training"),
		svcMakeEventRow("Weekly Training"),
		svcMakeEventRow("Weekly Training"),
	}
	for i := range eventRows {
		eventRows[i].SeriesId = &sid
	}

	seriesCalled := false
	createCalled := false

	repo := &mockSvcRepo{
		createSeriesFn: func(_ context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error) {
			seriesCalled = true
			assert.True(t, params.Recurring)
			assert.Equal(t, 3, params.RepeatWeeks)
			return eventRows, nil
		},
		createEventFn: func(_ context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error) {
			createCalled = true
			return &eventRows[0], nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	repeatWeeks := 3
	recurring := true
	body := &gen.CreateEventRequest{
		Type:        gen.Training,
		Title:       "Weekly Training",
		Date:        openapi_types.Date{Time: time.Now().UTC()},
		Recurring:   &recurring,
		RepeatWeeks: &repeatWeeks,
	}

	result, err := svc.CreateEvent(context.Background(), testTeamID, testUserID, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, seriesCalled, "CreateSeries should be called for recurring events")
	assert.False(t, createCalled, "CreateEvent should NOT be called for recurring events")
	assert.True(t, result.Recurring, "resulting event should have Recurring=true")
	assert.NotNil(t, result.SeriesId)
}

func TestEventService_CreateEvent_Recurring_RejectsExcessiveRepeatWeeks(t *testing.T) {
	t.Parallel()

	repo := &mockSvcRepo{
		createSeriesFn: func(_ context.Context, _ string, _ *events.CreateEventParams) ([]events.EventRow, error) {
			t.Fatal("CreateSeries must not be called when repeatWeeks exceeds the cap")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	repeatWeeks := 100000
	recurring := true
	body := &gen.CreateEventRequest{
		Type:        gen.Training,
		Title:       "Runaway Series",
		Date:        openapi_types.Date{Time: time.Now().UTC()},
		Recurring:   &recurring,
		RepeatWeeks: &repeatWeeks,
	}

	_, err := svc.CreateEvent(context.Background(), testTeamID, testUserID, body)
	require.ErrorIs(t, err, events.ErrRepeatWeeksTooLarge)
}

func TestEventService_SetAttendance(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	capturedStatus := ""
	rec := &events.AttendanceDBRow{
		Id:      uuid.New(),
		EventId: eventID,
		UserId:  userID,
		Status:  "yes",
	}

	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, evID, uID, tID string, status, _, _, _ *string) (*events.AttendanceDBRow, error) {
			assert.Equal(t, eventID.String(), evID)
			assert.Equal(t, userID.String(), uID)
			assert.Equal(t, teamID.String(), tID)
			if status != nil {
				capturedStatus = *status
			}
			return rec, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	req := gen.SetAttendanceRequest{
		UserId: userID,
		Status: gen.Yes,
	}

	result, err := svc.SetAttendance(context.Background(), eventID.String(), userID.String(), userID.String(), teamID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "yes", capturedStatus)
	assert.Equal(t, gen.Yes, result.Status)
}

// mockPermChecker satisfies the unexported permissionChecker interface via
// structural typing.
type mockPermChecker struct {
	perms teams.PermissionsJSON
	err   error
}

func (m *mockPermChecker) GetPermissions(_ context.Context, _, _ uuid.UUID) (teams.PermissionsJSON, error) {
	return m.perms, m.err
}

func TestEventService_SetAttendance_ForOtherMember_RequiresEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, _, _, _ string, _, _, _, _ *string) (*events.AttendanceDBRow, error) {
			t.Fatal("repository must not be called when caller lacks events:write")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "read"}})
	req := gen.SetAttendanceRequest{UserId: targetUserID, Status: gen.Yes}

	_, err := svc.SetAttendance(context.Background(), eventID.String(), callerID.String(), targetUserID.String(), teamID.String(), req)
	require.ErrorIs(t, err, events.ErrSetAttendanceForbidden)
}

func TestEventService_SetAttendance_ForOtherMember_AllowedWithEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	rec := &events.AttendanceDBRow{Id: uuid.New(), EventId: eventID, UserId: targetUserID, Status: "yes"}
	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, _, uID, _ string, _, _, _, _ *string) (*events.AttendanceDBRow, error) {
			assert.Equal(t, targetUserID.String(), uID)
			return rec, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "write"}})
	req := gen.SetAttendanceRequest{UserId: targetUserID, Status: gen.Yes}

	result, err := svc.SetAttendance(context.Background(), eventID.String(), callerID.String(), targetUserID.String(), teamID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestEventService_SetNomination_RequiresEventsWrite guards against a
// regression where a middleware path-parsing bug made this endpoint
// self-service (any member, regardless of permissions) instead of requiring
// events:write — nominating another member is an organizer-only action,
// never self-service, even for nominating oneself. This test exercises the
// service-layer check directly so the endpoint stays safe even if the
// middleware's route classification regresses again.
func TestEventService_SetNomination_RequiresEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	repo := &mockSvcRepo{
		setNominationFn: func(context.Context, string, string, string, bool) error {
			t.Fatal("repository must not be called when caller lacks events:write")
			return nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "read"}})
	req := gen.SetNominationRequest{UserId: targetUserID, Nominated: true}

	err := svc.SetNomination(context.Background(), eventID.String(), callerID.String(), teamID.String(), req)
	require.ErrorIs(t, err, events.ErrSetNominationForbidden)
}

func TestEventService_SetNomination_NilPermChecker_Forbidden(t *testing.T) {
	t.Parallel()

	repo := &mockSvcRepo{
		setNominationFn: func(context.Context, string, string, string, bool) error {
			t.Fatal("repository must not be called when there is no permission checker")
			return nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	req := gen.SetNominationRequest{UserId: uuid.New(), Nominated: true}

	err := svc.SetNomination(context.Background(), uuid.New().String(), uuid.New().String(), uuid.New().String(), req)
	require.ErrorIs(t, err, events.ErrSetNominationForbidden)
}

func TestEventService_SetNomination_AllowedWithEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	called := false
	repo := &mockSvcRepo{
		setNominationFn: func(_ context.Context, evID, uID, tID string, nominated bool) error {
			called = true
			assert.Equal(t, eventID.String(), evID)
			assert.Equal(t, targetUserID.String(), uID)
			assert.Equal(t, teamID.String(), tID)
			assert.True(t, nominated)
			return nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "write"}})
	req := gen.SetNominationRequest{UserId: targetUserID, Nominated: true}

	err := svc.SetNomination(context.Background(), eventID.String(), callerID.String(), teamID.String(), req)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestEventService_SetStatus_PassesTeamIDThrough(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	otherTeamID := uuid.New()
	row := svcMakeEventRow("Cancel Me")
	row.Id = eventID

	var capturedTeamID string
	repo := &mockSvcRepo{
		setStatusFn: func(_ context.Context, evID, tID, status, scope string) (*events.EventRow, error) {
			assert.Equal(t, eventID.String(), evID)
			capturedTeamID = tID
			assert.Equal(t, "cancelled", status)
			assert.Equal(t, "single", scope)
			return &row, nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	_, err := svc.SetStatus(context.Background(), testUserID, eventID.String(), otherTeamID.String(), "cancelled", "single")
	require.NoError(t, err)
	assert.Equal(t, otherTeamID.String(), capturedTeamID, "teamID must be threaded through to the repository so cross-team status changes are rejected at the DB layer")
}

func TestEventService_ListAttendance_RedactsDeclineReasonWithoutRole(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "private medical reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "no", Reason: &reason, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			// Team requires role "trainer-role"; viewer has no roles at all.
			return []string{"trainer-role"}, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].Reason, "a viewer without a reason-visibility role must not see another member's decline reason")
}

func TestEventService_ListAttendance_ShowsOwnDeclineReason(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	reason := "my own reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: viewerID, Status: "no", Reason: &reason, Name: "Self"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			t.Fatal("reason-visibility context must not be fetched for the viewer's own row")
			return nil, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].Reason)
	assert.Equal(t, reason, *rows[0].Reason)
}

func TestEventService_ListAttendance_ShowsDeclineReasonWithMatchingRole(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "private medical reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "no", Reason: &reason, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			return []string{"trainer-role"}, []string{"trainer-role"}, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil)
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].Reason)
	assert.Equal(t, reason, *rows[0].Reason)
}
