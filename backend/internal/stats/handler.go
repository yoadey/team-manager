package stats

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// statsService is the interface the Handler relies on.
type statsService interface {
	GetOverview(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error)
	GetMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error)
	GetAttendanceMatrix(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.AttendanceMatrix, error)
	ListGroupThresholds(ctx context.Context, teamID uuid.UUID) ([]GroupThresholdRow, error)
	SetGroupThresholds(ctx context.Context, teamID uuid.UUID, items []GroupThresholdRow) ([]GroupThresholdRow, error)
}

// Handler implements the stats-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    statsService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc statsService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// GetStatsOverview returns aggregated attendance statistics for the team.
func (h *Handler) GetStatsOverview(ctx context.Context, req gen.GetStatsOverviewRequestObject) (gen.GetStatsOverviewResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	overview, err := h.svc.GetOverview(ctx, req.TeamId, req.Params.From, req.Params.To)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetStatsOverview failed", "err", err)
		return nil, apierror.Internal("failed to get stats overview")
	}
	return gen.GetStatsOverview200JSONResponse(*overview), nil
}

// GetAttendanceMatrix returns the per-member-per-event attendance matrix.
func (h *Handler) GetAttendanceMatrix(ctx context.Context, req gen.GetAttendanceMatrixRequestObject) (gen.GetAttendanceMatrixResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	matrix, err := h.svc.GetAttendanceMatrix(ctx, req.TeamId, req.Params.From, req.Params.To)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetAttendanceMatrix failed", "err", err)
		return nil, apierror.Internal("failed to get attendance matrix")
	}
	return gen.GetAttendanceMatrix200JSONResponse(*matrix), nil
}

// GetMemberStats returns attendance statistics for a single member.
func (h *Handler) GetMemberStats(ctx context.Context, req gen.GetMemberStatsRequestObject) (gen.GetMemberStatsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	stats, err := h.svc.GetMemberStats(ctx, req.TeamId, req.UserId, req.Params.From, req.Params.To)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		h.logger.ErrorContext(ctx, "GetMemberStats failed", "err", err)
		return nil, apierror.Internal("failed to get member stats")
	}
	return gen.GetMemberStats200JSONResponse(*stats), nil
}

// ListStatsGroupThresholds returns the per-group (role) attendance
// thresholds of the team. Carries x-rbac-module: settings, so it requires
// settings:read -- it backs the team settings sheet, not the statistics page
// (which gets each group's resolved threshold inline on EventStat.groups).
func (h *Handler) ListStatsGroupThresholds(ctx context.Context, req gen.ListStatsGroupThresholdsRequestObject) (gen.ListStatsGroupThresholdsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	rows, err := h.svc.ListGroupThresholds(ctx, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListStatsGroupThresholds failed", "err", err)
		return nil, apierror.Internal("failed to list attendance thresholds")
	}
	return gen.ListStatsGroupThresholds200JSONResponse(toGenThresholdList(rows)), nil
}

// SetStatsGroupThresholds stores per-group (role) attendance thresholds.
// Requires settings:write (x-rbac-module: settings, mutating method).
func (h *Handler) SetStatsGroupThresholds(ctx context.Context, req gen.SetStatsGroupThresholdsRequestObject) (gen.SetStatsGroupThresholdsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	// Bounds the number of upserts one request can trigger -- the schema's
	// maxItems is not enforced by any middleware (see validateThreshold).
	if err := validate.UUIDItems(len(req.Body.Items), "items"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	items := make([]GroupThresholdRow, len(req.Body.Items))
	for i, it := range req.Body.Items {
		items[i] = GroupThresholdRow{RoleID: it.RoleId, Mode: string(it.Mode), Value: it.Value}
	}
	rows, err := h.svc.SetGroupThresholds(ctx, req.TeamId, items)
	if err != nil {
		if errors.Is(err, ErrInvalidThreshold) {
			return nil, apierror.BadRequest(err.Error())
		}
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, apierror.UnprocessableEntity("one or more roles do not belong to this team")
		}
		h.logger.ErrorContext(ctx, "SetStatsGroupThresholds failed", "err", err)
		return nil, apierror.Internal("failed to save attendance thresholds")
	}
	return gen.SetStatsGroupThresholds200JSONResponse(toGenThresholdList(rows)), nil
}

func toGenThresholdList(rows []GroupThresholdRow) gen.StatsGroupThresholdList {
	items := make([]gen.StatsGroupThreshold, len(rows))
	for i, r := range rows {
		items[i] = gen.StatsGroupThreshold{RoleId: r.RoleID, Mode: gen.StatsThresholdMode(r.Mode), Value: r.Value}
	}
	return gen.StatsGroupThresholdList{Items: items}
}
