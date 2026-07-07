package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// eventService is the interface the Handler relies on.
type eventService interface {
	ListEvents(ctx context.Context, teamID, userID, scope, cursor string, limit int) ([]gen.TeamEvent, *string, error)
	CreateEvent(ctx context.Context, teamID, userID string, body *gen.CreateEventJSONRequestBody) (*gen.TeamEvent, error)
	GetEvent(ctx context.Context, teamID, userID, eventID string) (*gen.TeamEvent, error)
	UpdateEvent(ctx context.Context, teamID, userID, eventID string, scope string, body *gen.UpdateEventJSONRequestBody) (*gen.TeamEvent, error)
	DeleteEvent(ctx context.Context, eventID, teamID, scope string) error
	SetStatus(ctx context.Context, userID, eventID, teamID, status, scope string) (*gen.TeamEvent, error)
	ListComments(ctx context.Context, eventID, teamID string, limit, offset int) ([]gen.EventComment, error)
	AddComment(ctx context.Context, eventID, userID, teamID, text string) (*gen.EventComment, error)
	DeleteComment(ctx context.Context, commentID, userID, teamID string) error
	ListAttendance(ctx context.Context, eventID, teamID, viewerID string) ([]gen.AttendanceRow, error)
	SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, req gen.SetAttendanceRequest) (*gen.AttendanceRecord, error)
	SetNomination(ctx context.Context, eventID, callerID, teamID string, req gen.SetNominationRequest) error
}

// Handler implements the event-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    eventService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc eventService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ─── ListEvents ─────────────────────────────────────────────────────────────

// ListEvents returns paginated events for a team.
func (h *Handler) ListEvents(ctx context.Context, request gen.ListEventsRequestObject) (gen.ListEventsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	scope := "upcoming"
	if request.Params.Scope != nil {
		if !request.Params.Scope.Valid() {
			return nil, apierror.BadRequest("scope: not a valid value")
		}
		scope = string(*request.Params.Scope)
	}
	limit := pagination.ParseLimit(request.Params.Limit)
	cursor := ""
	if request.Params.Cursor != nil {
		cursor = *request.Params.Cursor
	}

	evts, next, err := h.svc.ListEvents(ctx, request.TeamId.String(), user.Id.String(), scope, cursor, limit)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			return nil, apierror.BadRequest("invalid cursor")
		}
		h.logger.ErrorContext(ctx, "ListEvents failed", "err", err)
		return nil, apierror.Internal("failed to list events")
	}
	return gen.ListEvents200JSONResponse{Items: evts, NextCursor: next}, nil
}

// ─── CreateEvent ────────────────────────────────────────────────────────────

// CreateEvent creates a new event or recurring series.
func (h *Handler) CreateEvent(ctx context.Context, request gen.CreateEventRequestObject) (gen.CreateEventResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.Name(request.Body.Title); err != nil {
		return nil, apierror.BadRequest("title: " + err.Error())
	}
	if !request.Body.Type.Valid() {
		return nil, apierror.BadRequest("type: not a valid event type")
	}
	if request.Body.Date.IsZero() {
		return nil, apierror.BadRequest("date: is required")
	}
	if err := validateEventFields(request.Body.Location, request.Body.Note, request.Body.MeetTime, request.Body.StartTime, request.Body.EndTime, request.Body.ResponseMode); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if request.Body.NominatedRoleIds != nil {
		if err := validate.UUIDItems(len(*request.Body.NominatedRoleIds), "nominatedRoleIds"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}

	ev, err := h.svc.CreateEvent(ctx, request.TeamId.String(), user.Id.String(), request.Body)
	if err != nil {
		if errors.Is(err, ErrInvalidNominatedRoleIDs) {
			return nil, apierror.BadRequest("nominated_role_ids must refer to roles belonging to this team")
		}
		if errors.Is(err, ErrRepeatWeeksTooLarge) {
			return nil, apierror.BadRequest(err.Error())
		}
		h.logger.ErrorContext(ctx, "CreateEvent failed", "err", err)
		return nil, apierror.Internal("failed to create event")
	}
	metrics.TeamEvents.WithLabelValues("event", "create").Inc()
	return gen.CreateEvent201JSONResponse(*ev), nil
}

// ─── GetEvent ───────────────────────────────────────────────────────────────

// GetEvent returns a single event by ID.
func (h *Handler) GetEvent(ctx context.Context, request gen.GetEventRequestObject) (gen.GetEventResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	ev, err := h.svc.GetEvent(ctx, request.TeamId.String(), user.Id.String(), request.EventId.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.GetEvent404ApplicationProblemPlusJSONResponse{
				NotFoundApplicationProblemPlusJSONResponse: notFoundProblem("event not found"),
			}, nil
		}
		h.logger.ErrorContext(ctx, "GetEvent failed", "err", err)
		return nil, apierror.Internal("failed to get event")
	}
	return gen.GetEvent200JSONResponse(*ev), nil
}

// ─── UpdateEvent ────────────────────────────────────────────────────────────

// UpdateEvent updates an event or series.
func (h *Handler) UpdateEvent(ctx context.Context, request gen.UpdateEventRequestObject) (gen.UpdateEventResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if request.Body.Title != nil {
		if err := validate.Name(*request.Body.Title); err != nil {
			return nil, apierror.BadRequest("title: " + err.Error())
		}
	}
	if request.Body.Type != nil && !request.Body.Type.Valid() {
		return nil, apierror.BadRequest("type: not a valid event type")
	}
	if err := validateEventFields(request.Body.Location, request.Body.Note, request.Body.MeetTime, request.Body.StartTime, request.Body.EndTime, request.Body.ResponseMode); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if request.Body.NominatedRoleIds != nil {
		if err := validate.UUIDItems(len(*request.Body.NominatedRoleIds), "nominatedRoleIds"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}

	scope := "single"
	if request.Params.Scope != nil {
		if !request.Params.Scope.Valid() {
			return nil, apierror.BadRequest("scope: not a valid value")
		}
		scope = string(*request.Params.Scope)
	}

	ev, err := h.svc.UpdateEvent(ctx, request.TeamId.String(), user.Id.String(), request.EventId.String(), scope, request.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("event not found")
		}
		if errors.Is(err, ErrInvalidNominatedRoleIDs) {
			return nil, apierror.BadRequest("nominated_role_ids must refer to roles belonging to this team")
		}
		h.logger.ErrorContext(ctx, "UpdateEvent failed", "err", err)
		return nil, apierror.Internal("failed to update event")
	}
	metrics.TeamEvents.WithLabelValues("event", "update").Inc()
	return gen.UpdateEvent200JSONResponse(*ev), nil
}

// ─── DeleteEvent ────────────────────────────────────────────────────────────

// DeleteEvent deletes an event or series.
func (h *Handler) DeleteEvent(ctx context.Context, request gen.DeleteEventRequestObject) (gen.DeleteEventResponseObject, error) {
	_, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	scope := "single"
	if request.Params.Scope != nil {
		if !request.Params.Scope.Valid() {
			return nil, apierror.BadRequest("scope: not a valid value")
		}
		scope = string(*request.Params.Scope)
	}

	if err := h.svc.DeleteEvent(ctx, request.EventId.String(), request.TeamId.String(), scope); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("event not found")
		}
		h.logger.ErrorContext(ctx, "DeleteEvent failed", "err", err)
		return nil, apierror.Internal("failed to delete event")
	}
	metrics.TeamEvents.WithLabelValues("event", "delete").Inc()
	return gen.DeleteEvent204Response{}, nil
}

// ─── SetEventStatus ─────────────────────────────────────────────────────────

// SetEventStatus updates the status of an event or series.
func (h *Handler) SetEventStatus(ctx context.Context, request gen.SetEventStatusRequestObject) (gen.SetEventStatusResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if !request.Body.Status.Valid() {
		return nil, apierror.BadRequest("status: not a valid event status")
	}

	scope := "single"
	if request.Params.Scope != nil {
		if !request.Params.Scope.Valid() {
			return nil, apierror.BadRequest("scope: not a valid value")
		}
		scope = string(*request.Params.Scope)
	}

	ev, err := h.svc.SetStatus(ctx, user.Id.String(), request.EventId.String(), request.TeamId.String(), string(request.Body.Status), scope)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("event not found")
		}
		h.logger.ErrorContext(ctx, "SetEventStatus failed", "err", err)
		return nil, apierror.Internal("failed to set event status")
	}
	metrics.TeamEvents.WithLabelValues("event", "update").Inc()
	return gen.SetEventStatus200JSONResponse(*ev), nil
}

// ─── ListEventComments ───────────────────────────────────────────────────────

// ListEventComments returns paginated comments for an event.
func (h *Handler) ListEventComments(ctx context.Context, request gen.ListEventCommentsRequestObject) (gen.ListEventCommentsResponseObject, error) {
	_, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	limit, offset := pagination.Parse(request.Params.Limit, request.Params.Offset)
	comments, err := h.svc.ListComments(ctx, request.EventId.String(), request.TeamId.String(), limit, offset)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListEventComments failed", "err", err)
		return nil, apierror.Internal("failed to list comments")
	}
	return gen.ListEventComments200JSONResponse(comments), nil
}

// ─── AddEventComment ─────────────────────────────────────────────────────────

// AddEventComment adds a comment to an event.
func (h *Handler) AddEventComment(ctx context.Context, request gen.AddEventCommentRequestObject) (gen.AddEventCommentResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.Text(request.Body.Text, "text"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	comment, err := h.svc.AddComment(ctx, request.EventId.String(), user.Id.String(), request.TeamId.String(), request.Body.Text)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("event not found")
		}
		h.logger.ErrorContext(ctx, "AddEventComment failed", "err", err)
		return nil, apierror.Internal("failed to add comment")
	}
	metrics.TeamEvents.WithLabelValues("event", "create").Inc()
	return gen.AddEventComment201JSONResponse(*comment), nil
}

// ─── DeleteEventComment ──────────────────────────────────────────────────────

// DeleteEventComment deletes a comment if the user owns it.
func (h *Handler) DeleteEventComment(ctx context.Context, request gen.DeleteEventCommentRequestObject) (gen.DeleteEventCommentResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if err := h.svc.DeleteComment(ctx, request.CommentId.String(), user.Id.String(), request.TeamId.String()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("comment not found or not owned by user")
		}
		h.logger.ErrorContext(ctx, "DeleteEventComment failed", "err", err)
		return nil, apierror.Internal("failed to delete comment")
	}
	metrics.TeamEvents.WithLabelValues("event", "delete").Inc()
	return gen.DeleteEventComment204Response{}, nil
}

// ─── ListAttendance ──────────────────────────────────────────────────────────

// ListAttendance returns all attendance rows for an event.
func (h *Handler) ListAttendance(ctx context.Context, request gen.ListAttendanceRequestObject) (gen.ListAttendanceResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	rows, err := h.svc.ListAttendance(ctx, request.EventId.String(), request.TeamId.String(), user.Id.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "ListAttendance failed", "err", err)
		return nil, apierror.Internal("failed to list attendance")
	}
	return gen.ListAttendance200JSONResponse(rows), nil
}

// ─── SetAttendance ───────────────────────────────────────────────────────────

// SetAttendance upserts an attendance record.
func (h *Handler) SetAttendance(ctx context.Context, request gen.SetAttendanceRequestObject) (gen.SetAttendanceResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if !request.Body.Status.Valid() {
		return nil, apierror.BadRequest("status: not a valid attendance status")
	}
	if request.Body.ReasonVisibility != nil && !request.Body.ReasonVisibility.Valid() {
		return nil, apierror.BadRequest("reasonVisibility: not a valid value")
	}
	if request.Body.Reason != nil {
		if err := validate.MaxLen(*request.Body.Reason, 500, "reason"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}

	// Use user from body (manager setting attendance for others) or fall back to authed user.
	userID := request.Body.UserId.String()
	if userID == "" || userID == "00000000-0000-0000-0000-000000000000" {
		userID = user.Id.String()
	}

	rec, err := h.svc.SetAttendance(ctx, request.EventId.String(), user.Id.String(), userID, request.TeamId.String(), *request.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("event not found")
		}
		if errors.Is(err, ErrSetAttendanceForbidden) {
			return nil, apierror.Forbidden("not allowed to set attendance for another member")
		}
		h.logger.ErrorContext(ctx, "SetAttendance failed", "err", err)
		return nil, apierror.Internal("failed to set attendance")
	}
	metrics.TeamEvents.WithLabelValues("event", "update").Inc()
	return gen.SetAttendance200JSONResponse(*rec), nil
}

// ─── SetNomination ───────────────────────────────────────────────────────────

// SetNomination sets or removes nomination for a user on an event.
func (h *Handler) SetNomination(ctx context.Context, request gen.SetNominationRequestObject) (gen.SetNominationResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if request.Body.UserId == uuid.Nil {
		return nil, apierror.BadRequest("userId: is required")
	}

	if err := h.svc.SetNomination(ctx, request.EventId.String(), user.Id.String(), request.TeamId.String(), *request.Body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("event not found")
		}
		if errors.Is(err, ErrSetNominationForbidden) {
			return nil, apierror.Forbidden("events:write required to nominate a member")
		}
		h.logger.ErrorContext(ctx, "SetNomination failed", "err", err)
		return nil, apierror.Internal("failed to set nomination")
	}
	metrics.TeamEvents.WithLabelValues("event", "update").Inc()
	return gen.SetNomination204Response{}, nil
}

// errInvalidResponseMode is returned by validateEventFields when responseMode
// is set to a value outside the ResponseMode enum.
var errInvalidResponseMode = errors.New("responseMode: not a valid response mode")

// errEndTimeBeforeStartTime is returned by validateEventFields when both
// startTime and endTime are set and endTime does not come after startTime.
var errEndTimeBeforeStartTime = errors.New("endTime: must be after startTime")

// validateEventFields validates the optional free-text/time/enum fields shared
// by CreateEvent and UpdateEvent. No request-schema validator is wired into
// the router (see events/service.go), so these are the only enforcement point
// for the openapi.yaml-declared constraints on these fields.
func validateEventFields(location, note, meetTime, startTime, endTime *string, responseMode *gen.ResponseMode) error {
	if err := validateEventTextFields(location, note); err != nil {
		return err
	}
	if err := validateEventTimeFields(meetTime, startTime, endTime); err != nil {
		return err
	}
	if responseMode != nil && !responseMode.Valid() {
		return errInvalidResponseMode
	}
	return nil
}

// validateEventTextFields validates the optional location/note free-text fields.
func validateEventTextFields(location, note *string) error {
	if location != nil {
		if err := validate.MaxLen(*location, 255, "location"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if note != nil {
		if err := validate.MaxLen(*note, 10000, "note"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}

// validateEventTimeFields validates the optional HH:MM time-of-day fields.
func validateEventTimeFields(meetTime, startTime, endTime *string) error {
	if meetTime != nil && *meetTime != "" {
		if err := validate.TimeOfDay(*meetTime, "meetTime"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if startTime != nil && *startTime != "" {
		if err := validate.TimeOfDay(*startTime, "startTime"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if endTime != nil && *endTime != "" {
		if err := validate.TimeOfDay(*endTime, "endTime"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if startTime != nil && *startTime != "" && endTime != nil && *endTime != "" && *endTime <= *startTime {
		return errEndTimeBeforeStartTime
	}
	return nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// notFoundProblem builds a gen.NotFoundApplicationProblemPlusJSONResponse
// value, with a Type URI computed via apierror so it honors
// ERROR_TYPE_BASE_URI like every other error response.
func notFoundProblem(detail string) gen.NotFoundApplicationProblemPlusJSONResponse {
	e := apierror.New(http.StatusNotFound, "Not Found", detail)
	return gen.NotFoundApplicationProblemPlusJSONResponse{
		Title:  &e.Title,
		Status: &e.Status,
		Detail: &detail,
		Type:   &e.Type,
	}
}
