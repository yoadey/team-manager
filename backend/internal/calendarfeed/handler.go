package calendarfeed

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// feedService is the interface the Handler relies on.
type feedService interface {
	IssueToken(ctx context.Context, userID, teamID uuid.UUID) (string, error)
	RevokeToken(ctx context.Context, userID, teamID uuid.UUID) error
	ServeFeed(ctx context.Context, token string) ([]byte, error)
}

// Handler implements the calendar-feed methods of gen.StrictServerInterface.
type Handler struct {
	svc    feedService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc feedService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// IssueCalendarFeedToken mints (rotating any existing one) the caller's
// calendar feed link for this team. Self-service: any member with events
// read access can obtain their own link.
func (h *Handler) IssueCalendarFeedToken(ctx context.Context, req gen.IssueCalendarFeedTokenRequestObject) (gen.IssueCalendarFeedTokenResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	url, err := h.svc.IssueToken(ctx, user.Id, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "IssueCalendarFeedToken failed", "err", err)
		return nil, apierror.Internal("failed to issue calendar feed token")
	}
	return gen.IssueCalendarFeedToken200JSONResponse{Url: url}, nil
}

// RevokeCalendarFeedToken invalidates the caller's calendar feed link for
// this team, if any.
func (h *Handler) RevokeCalendarFeedToken(ctx context.Context, req gen.RevokeCalendarFeedTokenRequestObject) (gen.RevokeCalendarFeedTokenResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.RevokeToken(ctx, user.Id, req.TeamId); err != nil {
		h.logger.ErrorContext(ctx, "RevokeCalendarFeedToken failed", "err", err)
		return nil, apierror.Internal("failed to revoke calendar feed token")
	}
	return gen.RevokeCalendarFeedToken204Response{}, nil
}

// GetCalendarFeed serves the iCalendar feed for a bare token. Deliberately
// unauthenticated -- see cmd/server/main.go's router wiring and
// Service.ServeFeed's doc comment for the authorization model.
func (h *Handler) GetCalendarFeed(ctx context.Context, req gen.GetCalendarFeedRequestObject) (gen.GetCalendarFeedResponseObject, error) {
	ics, err := h.svc.ServeFeed(ctx, req.Token)
	if err != nil {
		if errors.Is(err, ErrFeedUnavailable) {
			detail := "not found"
			e := apierror.NotFound(detail)
			return gen.GetCalendarFeed404ApplicationProblemPlusJSONResponse{
				NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
					Title:  &e.Title,
					Detail: &detail,
					Status: &e.Status,
					Type:   &e.Type,
				},
			}, nil
		}
		h.logger.ErrorContext(ctx, "GetCalendarFeed failed", "err", err)
		return nil, apierror.Internal("failed to render calendar feed")
	}
	return gen.GetCalendarFeed200TextcalendarResponse{
		Body:          bytes.NewReader(ics),
		ContentLength: int64(len(ics)),
	}, nil
}
