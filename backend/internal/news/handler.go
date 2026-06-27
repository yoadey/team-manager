package news

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// newsService is the interface the Handler relies on.
type newsService interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.NewsItem, *string, error)
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

// ListNews returns paginated news items for the team.
func (h *Handler) ListNews(ctx context.Context, req gen.ListNewsRequestObject) (gen.ListNewsResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	limit := pagination.ParseLimit(req.Params.Limit)
	cursor := ""
	if req.Params.Cursor != nil {
		cursor = *req.Params.Cursor
	}
	items, next, err := h.svc.ListByTeam(ctx, req.TeamId, limit, cursor)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			return nil, apierror.BadRequest("invalid cursor")
		}
		h.logger.ErrorContext(ctx, "ListNews failed", "err", err)
		return nil, apierror.Internal("failed to list news")
	}
	return gen.ListNews200JSONResponse{Items: items, NextCursor: next}, nil
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
	if err := validate.Text(req.Body.Title, "title"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.Text(req.Body.Body, "body"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	item, err := h.svc.Create(ctx, req.TeamId, user.Id, req.Body)
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
	if req.Body.Title != nil {
		if err := validate.Text(*req.Body.Title, "title"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Body != nil {
		if err := validate.Text(*req.Body.Body, "body"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	item, err := h.svc.Update(ctx, req.NewsId, req.Body)
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
	if err := h.svc.Delete(ctx, req.NewsId); err != nil {
		h.logger.ErrorContext(ctx, "DeleteNews failed", "err", err)
		return nil, apierror.Internal("failed to delete news item")
	}
	return gen.DeleteNews204Response{}, nil
}
