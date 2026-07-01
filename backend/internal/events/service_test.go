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
)

// ─── mock repository ────────────────────────────────────────────────────────

// mockSvcRepo satisfies the unexported eventRepo interface via structural typing.
type mockSvcRepo struct {
	listEventsFn           func(ctx context.Context, teamID, scope string, limit int, cur *events.ListCursor) ([]events.EventRow, error)
	getEventFn             func(ctx context.Context, eventID string) (*events.EventRow, error)
	createEventFn          func(ctx context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error)
	createSeriesFn         func(ctx context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error)
	updateEventFn          func(ctx context.Context, eventID, teamID string, params *events.UpdateEventParams, scope string) (*events.EventRow, error)
	setStatusFn            func(ctx context.Context, eventID, status, scope string) (*events.EventRow, error)
	deleteEventFn          func(ctx context.Context, eventID, teamID, scope string) error
	getAttendanceSummaryFn func(ctx context.Context, eventID string) (events.EventSummaryData, error)
	getMyAttendanceFn      func(ctx context.Context, eventID, userID string) (*events.AttendanceDBRow, error)
	listAttendanceFn       func(ctx context.Context, eventID string) ([]events.AttendanceEnriched, error)
	setAttendanceFn        func(ctx context.Context, eventID, userID string, status, reason, reasonID, reasonVisibility *string) (*events.AttendanceDBRow, error)
	setNominationFn        func(ctx context.Context, eventID, userID string, nominated bool) error
	listCommentsFn         func(ctx context.Context, eventID string, limit, offset int) ([]events.CommentRow, error)
	addCommentFn           func(ctx context.Context, eventID, userID, text string) (*events.CommentRow, error)
	deleteCommentFn        func(ctx context.Context, commentID, userID string) error
}

func (m *mockSvcRepo) ListEvents(ctx context.Context, teamID, scope string, limit int, cur *events.ListCursor) ([]events.EventRow, error) {
	return m.listEventsFn(ctx, teamID, scope, limit, cur)
}

func (m *mockSvcRepo) GetEvent(ctx context.Context, eventID string) (*events.EventRow, error) {
	return m.getEventFn(ctx, eventID)
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

func (m *mockSvcRepo) SetStatus(ctx context.Context, eventID, status, scope string) (*events.EventRow, error) {
	return m.setStatusFn(ctx, eventID, status, scope)
}

func (m *mockSvcRepo) DeleteEvent(ctx context.Context, eventID, teamID, scope string) error {
	return m.deleteEventFn(ctx, eventID, teamID, scope)
}

func (m *mockSvcRepo) GetAttendanceSummary(ctx context.Context, eventID string) (events.EventSummaryData, error) {
	return m.getAttendanceSummaryFn(ctx, eventID)
}

func (m *mockSvcRepo) GetMyAttendance(ctx context.Context, eventID, userID string) (*events.AttendanceDBRow, error) {
	return m.getMyAttendanceFn(ctx, eventID, userID)
}

func (m *mockSvcRepo) ListAttendance(ctx context.Context, eventID string) ([]events.AttendanceEnriched, error) {
	return m.listAttendanceFn(ctx, eventID)
}

func (m *mockSvcRepo) SetAttendance(ctx context.Context, eventID, userID string, status, reason, reasonID, reasonVisibility *string) (*events.AttendanceDBRow, error) {
	return m.setAttendanceFn(ctx, eventID, userID, status, reason, reasonID, reasonVisibility)
}

func (m *mockSvcRepo) SetNomination(ctx context.Context, eventID, userID string, nominated bool) error {
	return m.setNominationFn(ctx, eventID, userID, nominated)
}

func (m *mockSvcRepo) ListComments(ctx context.Context, eventID string, limit, offset int) ([]events.CommentRow, error) {
	return m.listCommentsFn(ctx, eventID, limit, offset)
}

func (m *mockSvcRepo) AddComment(ctx context.Context, eventID, userID, text string) (*events.CommentRow, error) {
	return m.addCommentFn(ctx, eventID, userID, text)
}

func (m *mockSvcRepo) DeleteComment(ctx context.Context, commentID, userID string) error {
	return m.deleteCommentFn(ctx, commentID, userID)
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

func zeroSummaryFn(_ context.Context, _ string) (events.EventSummaryData, error) {
	return events.EventSummaryData{}, nil
}

func nilMyAttendanceFn(_ context.Context, _, _ string) (*events.AttendanceDBRow, error) {
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

	svc := events.NewService(repo, nil, nil, nil)
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

	svc := events.NewService(repo, nil, nil, nil)
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

func TestEventService_SetAttendance(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	userID := uuid.New()

	capturedStatus := ""
	rec := &events.AttendanceDBRow{
		Id:      uuid.New(),
		EventId: eventID,
		UserId:  userID,
		Status:  "yes",
	}

	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, evID, uID string, status, _, _, _ *string) (*events.AttendanceDBRow, error) {
			assert.Equal(t, eventID.String(), evID)
			assert.Equal(t, userID.String(), uID)
			if status != nil {
				capturedStatus = *status
			}
			return rec, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil)
	req := gen.SetAttendanceRequest{
		UserId: userID,
		Status: gen.Yes,
	}

	result, err := svc.SetAttendance(context.Background(), eventID.String(), userID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "yes", capturedStatus)
	assert.Equal(t, gen.Yes, result.Status)
}
