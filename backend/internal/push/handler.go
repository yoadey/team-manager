package push

import (
	"context"
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
