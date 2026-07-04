package events_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// mockEventService implements the eventService interface used by events.Handler.
// Fields left nil will panic if invoked, which is intentional: tests that
// expect a validation error to short-circuit before reaching the service must
// not touch these fields.
type mockEventService struct {
	createEvent   func(ctx context.Context, teamID, userID string, body *gen.CreateEventJSONRequestBody) (*gen.TeamEvent, error)
	updateEvent   func(ctx context.Context, teamID, userID, eventID, scope string, body *gen.UpdateEventJSONRequestBody) (*gen.TeamEvent, error)
	setStatus     func(ctx context.Context, userID, eventID, teamID, status, scope string) (*gen.TeamEvent, error)
	setAttendance func(ctx context.Context, eventID, callerID, userID, teamID string, req gen.SetAttendanceRequest) (*gen.AttendanceRecord, error)
	setNomination func(ctx context.Context, eventID, callerID, teamID string, req gen.SetNominationRequest) error
}

func (m *mockEventService) ListEvents(context.Context, string, string, string, string, int) ([]gen.TeamEvent, *string, error) {
	panic("not implemented")
}

func (m *mockEventService) CreateEvent(ctx context.Context, teamID, userID string, body *gen.CreateEventJSONRequestBody) (*gen.TeamEvent, error) {
	return m.createEvent(ctx, teamID, userID, body)
}

func (m *mockEventService) GetEvent(context.Context, string, string, string) (*gen.TeamEvent, error) {
	panic("not implemented")
}

func (m *mockEventService) UpdateEvent(ctx context.Context, teamID, userID, eventID, scope string, body *gen.UpdateEventJSONRequestBody) (*gen.TeamEvent, error) {
	return m.updateEvent(ctx, teamID, userID, eventID, scope, body)
}

func (m *mockEventService) DeleteEvent(context.Context, string, string, string) error {
	panic("not implemented")
}

func (m *mockEventService) SetStatus(ctx context.Context, userID, eventID, teamID, status, scope string) (*gen.TeamEvent, error) {
	return m.setStatus(ctx, userID, eventID, teamID, status, scope)
}

func (m *mockEventService) ListComments(context.Context, string, string, int, int) ([]gen.EventComment, error) {
	panic("not implemented")
}

func (m *mockEventService) AddComment(context.Context, string, string, string, string) (*gen.EventComment, error) {
	panic("not implemented")
}

func (m *mockEventService) DeleteComment(context.Context, string, string, string) error {
	panic("not implemented")
}

func (m *mockEventService) ListAttendance(context.Context, string, string, string) ([]gen.AttendanceRow, error) {
	panic("not implemented")
}

func (m *mockEventService) SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, req gen.SetAttendanceRequest) (*gen.AttendanceRecord, error) {
	return m.setAttendance(ctx, eventID, callerID, userID, teamID, req)
}

func (m *mockEventService) SetNomination(ctx context.Context, eventID, callerID, teamID string, req gen.SetNominationRequest) error {
	if m.setNomination != nil {
		return m.setNomination(ctx, eventID, callerID, teamID, req)
	}
	panic("not implemented")
}

func ctxWithUser() context.Context {
	return auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Alice", Email: "a@x.c"})
}

func TestEventHandler_CreateEvent_RejectsOversizedLocation(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.CreateEventJSONRequestBody{
		Type:     gen.Training,
		Title:    "Practice",
		Location: ptr(strings.Repeat("x", 256)),
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestEventHandler_CreateEvent_RejectsMalformedTime(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.CreateEventJSONRequestBody{
		Type:      gen.Training,
		Title:     "Practice",
		StartTime: ptr("not-a-time"),
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestEventHandler_CreateEvent_RejectsTooManyNominatedRoleIds(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	roleIDs := make([]uuid.UUID, 201)
	for i := range roleIDs {
		roleIDs[i] = uuid.New()
	}
	body := &gen.CreateEventJSONRequestBody{
		Type:             gen.Training,
		Title:            "Practice",
		NominatedRoleIds: &roleIDs,
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestEventHandler_UpdateEvent_RejectsTooManyNominatedRoleIds(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	roleIDs := make([]uuid.UUID, 201)
	for i := range roleIDs {
		roleIDs[i] = uuid.New()
	}
	body := &gen.UpdateEventJSONRequestBody{NominatedRoleIds: &roleIDs}
	_, err := h.UpdateEvent(ctxWithUser(), gen.UpdateEventRequestObject{TeamId: uuid.New(), EventId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestEventHandler_CreateEvent_RejectsInvalidType(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.CreateEventJSONRequestBody{
		Type:  gen.EventType("not-a-type"),
		Title: "Practice",
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

func TestEventHandler_CreateEvent_AcceptsValidFields(t *testing.T) {
	t.Parallel()
	called := false
	svc := &mockEventService{
		createEvent: func(context.Context, string, string, *gen.CreateEventJSONRequestBody) (*gen.TeamEvent, error) {
			called = true
			return &gen.TeamEvent{}, nil
		},
	}
	h := events.NewHandler(svc, slog.Default())

	body := &gen.CreateEventJSONRequestBody{
		Type:      gen.Training,
		Title:     "Practice",
		Location:  ptr("Main Hall"),
		StartTime: ptr("18:30"),
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestEventHandler_SetEventStatus_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.SetEventStatusJSONRequestBody{Status: gen.EventStatus("bogus")}
	_, err := h.SetEventStatus(ctxWithUser(), gen.SetEventStatusRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Body: body,
	})
	require.Error(t, err)
}

func TestEventHandler_SetAttendance_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.SetAttendanceJSONRequestBody{UserId: uuid.New(), Status: gen.AttendanceStatus("bogus")}
	_, err := h.SetAttendance(ctxWithUser(), gen.SetAttendanceRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Body: body,
	})
	require.Error(t, err)
}

func TestEventHandler_SetAttendance_RejectsOversizedReason(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.SetAttendanceJSONRequestBody{
		UserId: uuid.New(),
		Status: gen.No,
		Reason: ptr(strings.Repeat("x", 501)),
	}
	_, err := h.SetAttendance(ctxWithUser(), gen.SetAttendanceRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Body: body,
	})
	require.Error(t, err)
}

func TestEventHandler_SetNomination_ForbiddenMapsTo403(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{
		setNomination: func(context.Context, string, string, string, gen.SetNominationRequest) error {
			return events.ErrSetNominationForbidden
		},
	}, slog.Default())

	body := &gen.SetNominationJSONRequestBody{UserId: uuid.New(), Nominated: true}
	_, err := h.SetNomination(ctxWithUser(), gen.SetNominationRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Body: body,
	})
	require.Error(t, err)
	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok, "expected *apierror.APIError, got %T", err)
	assert.Equal(t, 403, apiErr.Status)
}

func ptr[T any](v T) *T { return &v }
