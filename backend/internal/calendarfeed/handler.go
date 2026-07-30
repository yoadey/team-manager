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
	GetSettings(ctx context.Context, userID, teamID uuid.UUID) ([]string, bool, error)
	UpdateSettings(ctx context.Context, userID, teamID uuid.UUID, types []string, includeBirthdays bool) error
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

// GetCalendarFeedSettings returns the caller's calendar feed content
// selection for this team, defaulting to "everything" if none was
// customized yet.
func (h *Handler) GetCalendarFeedSettings(ctx context.Context, req gen.GetCalendarFeedSettingsRequestObject) (gen.GetCalendarFeedSettingsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	types, includeBirthdays, err := h.svc.GetSettings(ctx, user.Id, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetCalendarFeedSettings failed", "err", err)
		return nil, apierror.Internal("failed to get calendar feed settings")
	}
	return gen.GetCalendarFeedSettings200JSONResponse{
		Types:            toGenEventTypes(types),
		IncludeBirthdays: includeBirthdays,
	}, nil
}

// UpdateCalendarFeedSettings updates the caller's calendar feed content
// selection for this team, applying to the existing subscription URL.
func (h *Handler) UpdateCalendarFeedSettings(ctx context.Context, req gen.UpdateCalendarFeedSettingsRequestObject) (gen.UpdateCalendarFeedSettingsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("request body required")
	}
	types := make([]string, len(req.Body.Types))
	for i, t := range req.Body.Types {
		types[i] = string(t)
	}
	if err := h.svc.UpdateSettings(ctx, user.Id, req.TeamId, types, req.Body.IncludeBirthdays); err != nil {
		if errors.Is(err, ErrInvalidEventType) {
			return nil, apierror.BadRequest(err.Error())
		}
		if errors.Is(err, ErrNoActiveToken) {
			return nil, apierror.NotFound("no active calendar feed token")
		}
		h.logger.ErrorContext(ctx, "UpdateCalendarFeedSettings failed", "err", err)
		return nil, apierror.Internal("failed to update calendar feed settings")
	}
	return gen.UpdateCalendarFeedSettings200JSONResponse{
		Types:            req.Body.Types,
		IncludeBirthdays: req.Body.IncludeBirthdays,
	}, nil
}

func toGenEventTypes(types []string) []gen.EventType {
	out := make([]gen.EventType, len(types))
	for i, t := range types {
		out[i] = gen.EventType(t)
	}
	return out
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
