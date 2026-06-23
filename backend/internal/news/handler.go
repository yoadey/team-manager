package news

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// newsService is the interface the Handler relies on.
type newsService interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID) ([]gen.NewsItem, error)
	Create(ctx context.Context, teamID, authorID uuid.UUID, body *gen.CreateNewsRequest) (gen.NewsItem, error)
	Update(ctx context.Context, id uuid.UUID, body *gen.UpdateNewsRequest) (gen.NewsItem, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Handler implements the news-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    newsService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc newsService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ListNews returns all news items for the team.
func (h *Handler) ListNews(ctx context.Context, req gen.ListNewsRequestObject) (gen.ListNewsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	items, err := h.svc.ListByTeam(ctx, uuid.UUID(req.TeamId))
	if err != nil {
		h.logger.ErrorContext(ctx, "ListNews failed", "err", err)
		return nil, apierror.Internal("failed to list news")
	}
	return gen.ListNews200JSONResponse(items), nil
}

// CreateNews creates a new news item.
func (h *Handler) CreateNews(ctx context.Context, req gen.CreateNewsRequestObject) (gen.CreateNewsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	item, err := h.svc.Create(ctx, uuid.UUID(req.TeamId), user.Id, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateNews failed", "err", err)
		return nil, apierror.Internal("failed to create news item")
	}
	return gen.CreateNews201JSONResponse(item), nil
}

// UpdateNews modifies an existing news item.
func (h *Handler) UpdateNews(ctx context.Context, req gen.UpdateNewsRequestObject) (gen.UpdateNewsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	item, err := h.svc.Update(ctx, uuid.UUID(req.NewsId), req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "UpdateNews failed", "err", err)
		return nil, apierror.Internal("failed to update news item")
	}
	return gen.UpdateNews200JSONResponse(item), nil
}

// DeleteNews removes a news item.
func (h *Handler) DeleteNews(ctx context.Context, req gen.DeleteNewsRequestObject) (gen.DeleteNewsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.Delete(ctx, uuid.UUID(req.NewsId)); err != nil {
		h.logger.ErrorContext(ctx, "DeleteNews failed", "err", err)
		return nil, apierror.Internal("failed to delete news item")
	}
	return gen.DeleteNews204Response{}, nil
}
