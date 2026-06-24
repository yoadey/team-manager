package stats

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// statsService is the interface the Handler relies on.
type statsService interface {
	GetOverview(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error)
	GetMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error)
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

// GetMemberStats returns attendance statistics for a single member.
func (h *Handler) GetMemberStats(ctx context.Context, req gen.GetMemberStatsRequestObject) (gen.GetMemberStatsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	stats, err := h.svc.GetMemberStats(ctx, req.TeamId, req.UserId, nil, nil)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetMemberStats failed", "err", err)
		return nil, apierror.Internal("failed to get member stats")
	}
	return gen.GetMemberStats200JSONResponse(*stats), nil
}
