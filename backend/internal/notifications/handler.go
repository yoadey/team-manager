package notifications

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// notifService is the interface the Handler relies on.
type notifService interface {
	List(ctx context.Context, teamID, userID uuid.UUID) (gen.NotificationsResult, error)
	MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error
}

// Handler implements the notifications-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    notifService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc notifService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ListNotifications returns the team's notifications for the current user.
func (h *Handler) ListNotifications(ctx context.Context, req gen.ListNotificationsRequestObject) (gen.ListNotificationsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	result, err := h.svc.List(ctx, uuid.UUID(req.TeamId), user.Id)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListNotifications failed", "err", err)
		return nil, apierror.Internal("failed to list notifications")
	}
	return gen.ListNotifications200JSONResponse(result), nil
}

// MarkNotificationsSeen marks all notifications as seen for the current user.
func (h *Handler) MarkNotificationsSeen(ctx context.Context, req gen.MarkNotificationsSeenRequestObject) (gen.MarkNotificationsSeenResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.MarkSeen(ctx, uuid.UUID(req.TeamId), user.Id); err != nil {
		h.logger.ErrorContext(ctx, "MarkNotificationsSeen failed", "err", err)
		return nil, apierror.Internal("failed to mark notifications seen")
	}
	return gen.MarkNotificationsSeen204Response{}, nil
}
