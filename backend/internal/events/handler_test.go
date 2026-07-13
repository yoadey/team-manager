package events_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
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

func (m *mockEventService) ListEvents(context.Context, string, string, gen.ListEventsParamsScope, string, int) ([]gen.TeamEvent, *string, error) {
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
		Date:     openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
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
		Date:      openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		StartTime: ptr("not-a-time"),
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
}

// Regression test: startTime/endTime were only checked individually for
// HH:MM format, never for endTime coming after startTime -- an event with
// endTime before (or equal to) startTime used to be accepted silently.
func TestEventHandler_CreateEvent_RejectsEndTimeNotAfterStartTime(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	for _, endTime := range []string{"09:00", "08:59"} {
		body := &gen.CreateEventJSONRequestBody{
			Type:      gen.Training,
			Title:     "Practice",
			Date:      openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
			StartTime: ptr("09:00"),
			EndTime:   ptr(endTime),
		}
		_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
		require.Error(t, err, "endTime=%s must be rejected", endTime)
	}
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
		Date:             openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
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

// The generated *ParamsScope enum types all have a .Valid() method, but
// unlike Type/Status/ResponseMode elsewhere in this handler, an unrecognized
// ?scope= value used to be silently absorbed instead of rejected: ListEvents
// fell through to its "all" default, and UpdateEvent/DeleteEvent/
// SetEventStatus fell through to "single" -- both cases are still exercised
// below to lock in the fix.
func TestEventHandler_ListEvents_RejectsInvalidScope(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	scope := gen.ListEventsParamsScope("not-a-scope")
	_, err := h.ListEvents(ctxWithUser(), gen.ListEventsRequestObject{
		TeamId: uuid.New(), Params: gen.ListEventsParams{Scope: &scope},
	})
	require.Error(t, err)
}

func TestEventHandler_UpdateEvent_RejectsInvalidScope(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	scope := gen.UpdateEventParamsScope("not-a-scope")
	body := &gen.UpdateEventJSONRequestBody{}
	_, err := h.UpdateEvent(ctxWithUser(), gen.UpdateEventRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Params: gen.UpdateEventParams{Scope: &scope}, Body: body,
	})
	require.Error(t, err)
}

func TestEventHandler_DeleteEvent_RejectsInvalidScope(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	scope := gen.DeleteEventParamsScope("not-a-scope")
	_, err := h.DeleteEvent(ctxWithUser(), gen.DeleteEventRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Params: gen.DeleteEventParams{Scope: &scope},
	})
	require.Error(t, err)
}

func TestEventHandler_SetEventStatus_RejectsInvalidScope(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	scope := gen.SetEventStatusParamsScope("not-a-scope")
	body := &gen.SetEventStatusJSONRequestBody{Status: gen.Active}
	_, err := h.SetEventStatus(ctxWithUser(), gen.SetEventStatusRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Params: gen.SetEventStatusParams{Scope: &scope}, Body: body,
	})
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
		Date:      openapi_types.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		Location:  ptr("Main Hall"),
		StartTime: ptr("18:30"),
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.NoError(t, err)
	assert.True(t, called)
}

// CreateEventRequest.Date is a non-pointer Date field (required per
// openapi.yaml), but nothing in this stack enforces "required" at decode
// time — an omitted field just leaves Go's zero time.Time{}. Unlike absences,
// there is no DB CHECK constraint backstop, so an unguarded handler would
// silently persist an event dated 0001-01-01 with a 201.
func TestEventHandler_CreateEvent_MissingDate_Returns400(t *testing.T) {
	t.Parallel()
	svc := &mockEventService{
		createEvent: func(context.Context, string, string, *gen.CreateEventJSONRequestBody) (*gen.TeamEvent, error) {
			t.Fatal("service must not be called when 'date' is missing")
			return nil, nil
		},
	}
	h := events.NewHandler(svc, slog.Default())

	body := &gen.CreateEventJSONRequestBody{
		Type:  gen.Training,
		Title: "Practice",
	}
	_, err := h.CreateEvent(ctxWithUser(), gen.CreateEventRequestObject{TeamId: uuid.New(), Body: body})
	require.Error(t, err)
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

// Regression test: reasonId had no length cap at all, unlike its sibling
// reason field -- an unbounded TEXT column with no validation, reachable by
// any self-service caller (attendance is self-service).
func TestEventHandler_SetAttendance_RejectsOversizedReasonId(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{}, slog.Default())

	body := &gen.SetAttendanceJSONRequestBody{
		UserId:   uuid.New(),
		Status:   gen.No,
		ReasonId: ptr(strings.Repeat("x", 501)),
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

// SetNominationRequest.UserId is a non-pointer required UUID (per
// openapi.yaml); an omitted field decodes to the zero UUID rather than
// failing at decode time. Unlike SetAttendance (which explicitly treats the
// zero UUID as "fall back to the caller" for self-service calls),
// SetNomination has no such self-service meaning -- a caller must always
// specify who they're nominating -- so this must 400, not silently pass the
// zero UUID through to a misleading 404 "event not found".
func TestEventHandler_SetNomination_MissingUserId_Returns400(t *testing.T) {
	t.Parallel()
	h := events.NewHandler(&mockEventService{
		setNomination: func(context.Context, string, string, string, gen.SetNominationRequest) error {
			t.Fatal("service must not be called when userId is missing")
			return nil
		},
	}, slog.Default())

	body := &gen.SetNominationJSONRequestBody{Nominated: true}
	_, err := h.SetNomination(ctxWithUser(), gen.SetNominationRequestObject{
		TeamId: uuid.New(), EventId: uuid.New(), Body: body,
	})
	require.Error(t, err)
}

func ptr[T any](v T) *T { return &v }
