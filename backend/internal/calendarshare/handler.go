package calendarshare

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// shareService is the interface the Handler relies on.
type shareService interface {
	Grant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*ShareRow, error)
	Revoke(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error
	ListGrants(ctx context.Context, ownerTeamID uuid.UUID) ([]ShareRow, error)
	ListSources(ctx context.Context, viewerTeamID uuid.UUID) ([]ShareRow, error)
	ListEvents(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID, from, to *time.Time) ([]RedactedEventRow, error)
}

// Handler implements the calendar-share methods of gen.StrictServerInterface.
type Handler struct {
	svc    shareService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc shareService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func toGenCalendarShare(r ShareRow) gen.CalendarShare {
	return gen.CalendarShare{
		ViewerTeamId:   r.TeamId,
		ViewerTeamName: r.TeamName,
		CreatedAt:      r.CreatedAt,
	}
}

func toGenSharedCalendarSource(r ShareRow) gen.SharedCalendarSource {
	return gen.SharedCalendarSource{
		OwnerTeamId:   r.TeamId,
		OwnerTeamName: r.TeamName,
	}
}

func notFoundResponse(detail string) gen.NotFoundApplicationProblemPlusJSONResponse {
	e := apierror.NotFound(detail)
	return gen.NotFoundApplicationProblemPlusJSONResponse{
		Title:  &e.Title,
		Detail: &detail,
		Status: &e.Status,
		Type:   &e.Type,
	}
}

// ListCalendarShares lists the teams this team has granted calendar
// visibility to. Settings-gated (owner-team perspective).
func (h *Handler) ListCalendarShares(ctx context.Context, req gen.ListCalendarSharesRequestObject) (gen.ListCalendarSharesResponseObject, error) {
	rows, err := h.svc.ListGrants(ctx, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListCalendarShares failed", "err", err)
		return nil, apierror.Internal("failed to list calendar shares")
	}
	out := make(gen.ListCalendarShares200JSONResponse, len(rows))
	for i, r := range rows {
		out[i] = toGenCalendarShare(r)
	}
	return out, nil
}

// CreateCalendarShare grants another team read-only calendar visibility.
// Settings-gated (owner-team perspective).
func (h *Handler) CreateCalendarShare(ctx context.Context, req gen.CreateCalendarShareRequestObject) (gen.CreateCalendarShareResponseObject, error) {
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	row, err := h.svc.Grant(ctx, req.TeamId, req.Body.ViewerTeamId)
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return gen.CreateCalendarShare404ApplicationProblemPlusJSONResponse{
				NotFoundApplicationProblemPlusJSONResponse: notFoundResponse("viewer team not found"),
			}, nil
		}
		if errors.Is(err, ErrCannotShareWithSelf) {
			return nil, apierror.BadRequest(err.Error())
		}
		h.logger.ErrorContext(ctx, "CreateCalendarShare failed", "err", err)
		return nil, apierror.Internal("failed to create calendar share")
	}
	return gen.CreateCalendarShare201JSONResponse(toGenCalendarShare(*row)), nil
}

// DeleteCalendarShare revokes a previously granted calendar share.
// Settings-gated (owner-team perspective).
func (h *Handler) DeleteCalendarShare(ctx context.Context, req gen.DeleteCalendarShareRequestObject) (gen.DeleteCalendarShareResponseObject, error) {
	if err := h.svc.Revoke(ctx, req.TeamId, req.ViewerTeamId); err != nil {
		h.logger.ErrorContext(ctx, "DeleteCalendarShare failed", "err", err)
		return nil, apierror.Internal("failed to revoke calendar share")
	}
	return gen.DeleteCalendarShare204Response{}, nil
}

// ListSharedCalendarSources lists the teams that have granted this team
// calendar visibility. Membership-gated only (viewer-team perspective).
func (h *Handler) ListSharedCalendarSources(ctx context.Context, req gen.ListSharedCalendarSourcesRequestObject) (gen.ListSharedCalendarSourcesResponseObject, error) {
	rows, err := h.svc.ListSources(ctx, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListSharedCalendarSources failed", "err", err)
		return nil, apierror.Internal("failed to list shared calendar sources")
	}
	out := make(gen.ListSharedCalendarSources200JSONResponse, len(rows))
	for i, r := range rows {
		out[i] = toGenSharedCalendarSource(r)
	}
	return out, nil
}

// ListSharedCalendarEvents returns ownerTeamId's redacted schedule, if
// ownerTeamId currently grants req.TeamId (the caller's team) calendar
// visibility.
func (h *Handler) ListSharedCalendarEvents(ctx context.Context, req gen.ListSharedCalendarEventsRequestObject) (gen.ListSharedCalendarEventsResponseObject, error) {
	var from, to *time.Time
	if req.Params.From != nil {
		t := req.Params.From.Time
		from = &t
	}
	if req.Params.To != nil {
		t := req.Params.To.Time
		to = &t
	}

	rows, err := h.svc.ListEvents(ctx, req.OwnerTeamId, req.TeamId, from, to)
	if err != nil {
		if errors.Is(err, ErrNoGrant) {
			return gen.ListSharedCalendarEvents404ApplicationProblemPlusJSONResponse{
				NotFoundApplicationProblemPlusJSONResponse: notFoundResponse("not found"),
			}, nil
		}
		h.logger.ErrorContext(ctx, "ListSharedCalendarEvents failed", "err", err)
		return nil, apierror.Internal("failed to list shared calendar events")
	}

	out := make(gen.ListSharedCalendarEvents200JSONResponse, len(rows))
	for i, e := range rows {
		out[i] = gen.SharedCalendarEvent{
			Id:        e.Id,
			Type:      gen.EventType(e.Type),
			Title:     e.Title,
			Date:      openapi_types.Date{Time: e.Date},
			StartTime: e.StartTime,
			EndTime:   e.EndTime,
			Location:  e.Location,
		}
		if e.EndDate != nil {
			out[i].MultiDayEndDate = &openapi_types.Date{Time: *e.EndDate}
		}
	}
	return out, nil
}
