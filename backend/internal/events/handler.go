package events

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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
	DeleteEvent(ctx context.Context, eventID, scope string) error
	SetStatus(ctx context.Context, userID, eventID, status, scope string) (*gen.TeamEvent, error)
	ListComments(ctx context.Context, eventID string, limit, offset int) ([]gen.EventComment, error)
	AddComment(ctx context.Context, eventID, userID, text string) (*gen.EventComment, error)
	DeleteComment(ctx context.Context, commentID, userID string) error
	ListAttendance(ctx context.Context, eventID string) ([]gen.AttendanceRow, error)
	SetAttendance(ctx context.Context, eventID, userID string, req gen.SetAttendanceRequest) (*gen.AttendanceRecord, error)
	SetNomination(ctx context.Context, eventID, userID string, req gen.SetNominationRequest) error
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

	ev, err := h.svc.CreateEvent(ctx, request.TeamId.String(), user.Id.String(), request.Body)
	if err != nil {
		if errors.Is(err, ErrInvalidNominatedRoleIDs) {
			return nil, apierror.BadRequest("nominated_role_ids must refer to roles belonging to this team")
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

	scope := "single"
	if request.Params.Scope != nil {
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
		scope = string(*request.Params.Scope)
	}

	if err := h.svc.DeleteEvent(ctx, request.EventId.String(), scope); err != nil {
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

	scope := "single"
	if request.Params.Scope != nil {
		scope = string(*request.Params.Scope)
	}

	ev, err := h.svc.SetStatus(ctx, user.Id.String(), request.EventId.String(), string(request.Body.Status), scope)
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
	comments, err := h.svc.ListComments(ctx, request.EventId.String(), limit, offset)
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

	comment, err := h.svc.AddComment(ctx, request.EventId.String(), user.Id.String(), request.Body.Text)
	if err != nil {
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

	if err := h.svc.DeleteComment(ctx, request.CommentId.String(), user.Id.String()); err != nil {
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
	_, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	rows, err := h.svc.ListAttendance(ctx, request.EventId.String())
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

	// Use user from body (manager setting attendance for others) or fall back to authed user.
	userID := request.Body.UserId.String()
	if userID == "" || userID == "00000000-0000-0000-0000-000000000000" {
		userID = user.Id.String()
	}

	rec, err := h.svc.SetAttendance(ctx, request.EventId.String(), userID, *request.Body)
	if err != nil {
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

	if err := h.svc.SetNomination(ctx, request.EventId.String(), user.Id.String(), *request.Body); err != nil {
		h.logger.ErrorContext(ctx, "SetNomination failed", "err", err)
		return nil, apierror.Internal("failed to set nomination")
	}
	metrics.TeamEvents.WithLabelValues("event", "update").Inc()
	return gen.SetNomination204Response{}, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// notFoundProblem builds a gen.NotFoundApplicationProblemPlusJSONResponse value.
func notFoundProblem(detail string) gen.NotFoundApplicationProblemPlusJSONResponse {
	title := "Not Found"
	status := http.StatusNotFound
	typeStr := "https://teammanager.example/errors/not-found"
	return gen.NotFoundApplicationProblemPlusJSONResponse{
		Title:  &title,
		Status: &status,
		Detail: &detail,
		Type:   &typeStr,
	}
}
