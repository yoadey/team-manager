package push

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// pushService is the interface the Handler relies on.
type pushService interface {
	Register(ctx context.Context, userID uuid.UUID, sub Subscription) error
	Unregister(ctx context.Context, userID uuid.UUID, endpoint string) error
	GetPreferences(ctx context.Context, teamID, userID uuid.UUID) (CategoryPreferences, error)
	SetPreferences(ctx context.Context, teamID, userID uuid.UUID, prefs CategoryPreferences) error
}

// Handler implements the push-subscription methods of gen.StrictServerInterface.
type Handler struct {
	svc    pushService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc pushService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterPushSubscription registers (or updates) the caller's Web Push
// subscription. Self-service and user-scoped: any authenticated user can
// register a subscription for themselves, covering every team they belong
// to.
func (h *Handler) RegisterPushSubscription(ctx context.Context, req gen.RegisterPushSubscriptionRequestObject) (gen.RegisterPushSubscriptionResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	sub := Subscription{
		Endpoint: req.Body.Endpoint,
		P256dh:   req.Body.Keys.P256dh,
		AuthKey:  req.Body.Keys.Auth,
	}
	if sub.Endpoint == "" || sub.P256dh == "" || sub.AuthKey == "" {
		return nil, apierror.BadRequest("endpoint and keys are required")
	}
	if err := h.svc.Register(ctx, user.Id, sub); err != nil {
		h.logger.ErrorContext(ctx, "RegisterPushSubscription failed", "err", err)
		return nil, apierror.Internal("failed to register push subscription")
	}
	return gen.RegisterPushSubscription204Response{}, nil
}

// DeletePushSubscription unregisters the caller's subscription for the given
// endpoint. Scoped to the caller's own subscriptions -- deleting an
// endpoint that belongs to a different user (or doesn't exist) is a no-op.
func (h *Handler) DeletePushSubscription(ctx context.Context, req gen.DeletePushSubscriptionRequestObject) (gen.DeletePushSubscriptionResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.Unregister(ctx, user.Id, req.Params.Endpoint); err != nil {
		h.logger.ErrorContext(ctx, "DeletePushSubscription failed", "err", err)
		return nil, apierror.Internal("failed to delete push subscription")
	}
	return gen.DeletePushSubscription204Response{}, nil
}

// GetPushPreferences returns the caller's per-category push preferences for
// teamId. Self-service and public-module: membership in the team (enforced
// by RequirePermission before this handler runs) is the only requirement.
func (h *Handler) GetPushPreferences(ctx context.Context, req gen.GetPushPreferencesRequestObject) (gen.GetPushPreferencesResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	prefs, err := h.svc.GetPreferences(ctx, req.TeamId, user.Id)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetPushPreferences failed", "err", err)
		return nil, apierror.Internal("failed to get push preferences")
	}
	return gen.GetPushPreferences200JSONResponse(toGenPreferences(prefs)), nil
}

// SetPushPreferences saves the caller's per-category push preferences for
// teamId.
func (h *Handler) SetPushPreferences(ctx context.Context, req gen.SetPushPreferencesRequestObject) (gen.SetPushPreferencesResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	hoursBefore := req.Body.EventReminderHoursBefore
	if hoursBefore < MinEventReminderHoursBefore || hoursBefore > MaxEventReminderHoursBefore {
		return nil, apierror.BadRequest(fmt.Sprintf(
			"eventReminderHoursBefore must be between %d and %d", MinEventReminderHoursBefore, MaxEventReminderHoursBefore,
		))
	}
	prefs := CategoryPreferences{
		Attendance:               req.Body.Attendance,
		Events:                   req.Body.Events,
		News:                     req.Body.News,
		Polls:                    req.Body.Polls,
		Absence:                  req.Body.Absence,
		EventReminderEnabled:     req.Body.EventReminderEnabled,
		EventReminderHoursBefore: int16(hoursBefore),
	}
	if err := h.svc.SetPreferences(ctx, req.TeamId, user.Id, prefs); err != nil {
		h.logger.ErrorContext(ctx, "SetPushPreferences failed", "err", err)
		return nil, apierror.Internal("failed to set push preferences")
	}
	return gen.SetPushPreferences204Response{}, nil
}

// toGenPreferences maps a CategoryPreferences to the generated wire type.
func toGenPreferences(p CategoryPreferences) gen.PushCategoryPreferences {
	return gen.PushCategoryPreferences{
		Attendance:               p.Attendance,
		Events:                   p.Events,
		News:                     p.News,
		Polls:                    p.Polls,
		Absence:                  p.Absence,
		EventReminderEnabled:     p.EventReminderEnabled,
		EventReminderHoursBefore: int(p.EventReminderHoursBefore),
	}
}
