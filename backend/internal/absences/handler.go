package absences

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// absenceService is the interface the Handler relies on.
type absenceService interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]gen.Absence, error)
	ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit, offset int) ([]gen.Absence, error)
	Create(ctx context.Context, teamID uuid.UUID, body *gen.CreateAbsenceRequest) (gen.Absence, error)
	Update(ctx context.Context, id uuid.UUID, body *gen.UpdateAbsenceRequest) (gen.Absence, error)
	Delete(ctx context.Context, id uuid.UUID) error
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
	limit, offset := pagination.Parse(req.Params.Limit, req.Params.Offset)
	absences, err := h.svc.ListByTeam(ctx, req.TeamId, limit, offset)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListAbsences failed", "err", err)
		return nil, apierror.Internal("failed to list absences")
	}
	return gen.ListAbsences200JSONResponse(absences), nil
}

// CreateAbsence creates a new absence entry.
func (h *Handler) CreateAbsence(ctx context.Context, req gen.CreateAbsenceRequestObject) (gen.CreateAbsenceResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
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
	return gen.CreateAbsence201JSONResponse(absence), nil
}

// ListMyAbsences returns paginated absences for the authenticated user.
func (h *Handler) ListMyAbsences(ctx context.Context, req gen.ListMyAbsencesRequestObject) (gen.ListMyAbsencesResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	limit, offset := pagination.Parse(req.Params.Limit, req.Params.Offset)
	absences, err := h.svc.ListByUser(ctx, req.TeamId, user.Id, limit, offset)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListMyAbsences failed", "err", err)
		return nil, apierror.Internal("failed to list absences")
	}
	return gen.ListMyAbsences200JSONResponse(absences), nil
}

// DeleteAbsence removes an absence.
func (h *Handler) DeleteAbsence(ctx context.Context, req gen.DeleteAbsenceRequestObject) (gen.DeleteAbsenceResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.Delete(ctx, req.AbsenceId); err != nil {
		h.logger.ErrorContext(ctx, "DeleteAbsence failed", "err", err)
		return nil, apierror.Internal("failed to delete absence")
	}
	return gen.DeleteAbsence204Response{}, nil
}

// UpdateAbsence modifies an existing absence.
func (h *Handler) UpdateAbsence(ctx context.Context, req gen.UpdateAbsenceRequestObject) (gen.UpdateAbsenceResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
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
	absence, err := h.svc.Update(ctx, req.AbsenceId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "UpdateAbsence failed", "err", err)
		return nil, apierror.Internal("failed to update absence")
	}
	return gen.UpdateAbsence200JSONResponse(absence), nil
}
