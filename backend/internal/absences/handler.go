package absences

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// absenceService is the interface the Handler relies on.
type absenceService interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.Absence, *string, error)
	ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit int, cursor string) ([]gen.Absence, *string, error)
	Create(ctx context.Context, teamID uuid.UUID, body *gen.CreateAbsenceRequest) (gen.Absence, error)
	Update(ctx context.Context, id, teamID, userID uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error)
	Delete(ctx context.Context, id, teamID, userID uuid.UUID) error
}

// Handler implements the absence-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    absenceService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc absenceService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ListAbsences returns paginated absences for a team.
func (h *Handler) ListAbsences(ctx context.Context, req gen.ListAbsencesRequestObject) (gen.ListAbsencesResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	limit := pagination.ParseLimit(req.Params.Limit)
	cursor := ""
	if req.Params.Cursor != nil {
		cursor = *req.Params.Cursor
	}
	absences, next, err := h.svc.ListByTeam(ctx, req.TeamId, limit, cursor)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			return nil, apierror.BadRequest("invalid cursor")
		}
		h.logger.ErrorContext(ctx, "ListAbsences failed", "err", err)
		return nil, apierror.Internal("failed to list absences")
	}
	return gen.ListAbsences200JSONResponse{Items: absences, NextCursor: next}, nil
}

// CreateAbsence creates a new absence entry. Absences are self-service (any
// team member may report their own absence, regardless of RBAC module
// permissions), so the target user must be the authenticated caller — this is
// the only guard against one member creating absences on another's behalf.
func (h *Handler) CreateAbsence(ctx context.Context, req gen.CreateAbsenceRequestObject) (gen.CreateAbsenceResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.UserId != user.Id {
		return nil, apierror.Forbidden("cannot create an absence for another user")
	}
	if !req.Body.To.IsZero() && req.Body.From.After(req.Body.To.Time) {
		return nil, apierror.BadRequest("'from' must not be after 'to'")
	}
	if req.Body.Reason != nil {
		if err := validate.MaxLen(*req.Body.Reason, 500, "reason"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	absence, err := h.svc.Create(ctx, req.TeamId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateAbsence failed", "err", err)
		return nil, apierror.Internal("failed to create absence")
	}
	metrics.TeamEvents.WithLabelValues("absence", "create").Inc()
	return gen.CreateAbsence201JSONResponse(absence), nil
}

// ListMyAbsences returns paginated absences for the authenticated user.
func (h *Handler) ListMyAbsences(ctx context.Context, req gen.ListMyAbsencesRequestObject) (gen.ListMyAbsencesResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	limit := pagination.ParseLimit(req.Params.Limit)
	cursor := ""
	if req.Params.Cursor != nil {
		cursor = *req.Params.Cursor
	}
	absences, next, err := h.svc.ListByUser(ctx, req.TeamId, user.Id, limit, cursor)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			return nil, apierror.BadRequest("invalid cursor")
		}
		h.logger.ErrorContext(ctx, "ListMyAbsences failed", "err", err)
		return nil, apierror.Internal("failed to list absences")
	}
	return gen.ListMyAbsences200JSONResponse{Items: absences, NextCursor: next}, nil
}

// DeleteAbsence removes an absence. Self-service: a member may only delete
// their own absence entries.
func (h *Handler) DeleteAbsence(ctx context.Context, req gen.DeleteAbsenceRequestObject) (gen.DeleteAbsenceResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.Delete(ctx, req.AbsenceId, req.TeamId, user.Id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("absence not found")
		}
		h.logger.ErrorContext(ctx, "DeleteAbsence failed", "err", err)
		return nil, apierror.Internal("failed to delete absence")
	}
	metrics.TeamEvents.WithLabelValues("absence", "delete").Inc()
	return gen.DeleteAbsence204Response{}, nil
}

// UpdateAbsence modifies an existing absence. Self-service: a member may only
// update their own absence entries.
func (h *Handler) UpdateAbsence(ctx context.Context, req gen.UpdateAbsenceRequestObject) (gen.UpdateAbsenceResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.From != nil && req.Body.To != nil && req.Body.From.After(req.Body.To.Time) {
		return nil, apierror.BadRequest("'from' must not be after 'to'")
	}
	if req.Body.Reason != nil {
		if err := validate.MaxLen(*req.Body.Reason, 500, "reason"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	absence, err := h.svc.Update(ctx, req.AbsenceId, req.TeamId, user.Id, req.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("absence not found")
		}
		h.logger.ErrorContext(ctx, "UpdateAbsence failed", "err", err)
		return nil, apierror.Internal("failed to update absence")
	}
	metrics.TeamEvents.WithLabelValues("absence", "update").Inc()
	return gen.UpdateAbsence200JSONResponse(absence), nil
}
