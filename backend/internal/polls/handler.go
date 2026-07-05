package polls

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// pollService is the interface the Handler relies on.
type pollService interface {
	ListByTeam(ctx context.Context, teamID, currentUserID uuid.UUID, limit int, cursor string) ([]gen.Poll, *string, error)
	Create(ctx context.Context, teamID, creatorID uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error)
	Vote(ctx context.Context, pollID, teamID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error)
	Delete(ctx context.Context, id, teamID uuid.UUID) error
}

// Handler implements the polls-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    pollService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc pollService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ListPolls returns paginated polls for the team.
func (h *Handler) ListPolls(ctx context.Context, req gen.ListPollsRequestObject) (gen.ListPollsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	limit := pagination.ParseLimit(req.Params.Limit)
	cursor := ""
	if req.Params.Cursor != nil {
		cursor = *req.Params.Cursor
	}
	polls, next, err := h.svc.ListByTeam(ctx, req.TeamId, user.Id, limit, cursor)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			return nil, apierror.BadRequest("invalid cursor")
		}
		h.logger.ErrorContext(ctx, "ListPolls failed", "err", err)
		return nil, apierror.Internal("failed to list polls")
	}
	return gen.ListPolls200JSONResponse{Items: polls, NextCursor: next}, nil
}

// CreatePoll creates a new poll.
func (h *Handler) CreatePoll(ctx context.Context, req gen.CreatePollRequestObject) (gen.CreatePollResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.RequireNonEmpty(req.Body.Question, "question"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.MaxLen(req.Body.Question, 1000, "question"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if len(req.Body.Options) < 2 || len(req.Body.Options) > 4 {
		return nil, apierror.BadRequest("polls must have between 2 and 4 options")
	}
	for i, opt := range req.Body.Options {
		if err := validate.RequireNonEmpty(opt, "option"); err != nil {
			_ = i
			return nil, apierror.BadRequest(err.Error())
		}
		if err := validate.MaxLen(opt, 500, "option"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	poll, err := h.svc.Create(ctx, req.TeamId, user.Id, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreatePoll failed", "err", err)
		return nil, apierror.Internal("failed to create poll")
	}
	metrics.TeamEvents.WithLabelValues("poll", "create").Inc()
	return gen.CreatePoll201JSONResponse(poll), nil
}

// VotePoll casts or replaces a vote on a poll.
func (h *Handler) VotePoll(ctx context.Context, req gen.VotePollRequestObject) (gen.VotePollResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.UUIDItems(len(req.Body.OptionIds), "optionIds"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	optionIDs := append([]uuid.UUID(nil), req.Body.OptionIds...)
	poll, err := h.svc.Vote(ctx, req.PollId, req.TeamId, user.Id, optionIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("poll not found")
		}
		if errors.Is(err, ErrSingleChoiceMultipleOptions) {
			return nil, apierror.UnprocessableEntity(err.Error())
		}
		if errors.Is(err, ErrOptionNotInPoll) {
			return nil, apierror.UnprocessableEntity(err.Error())
		}
		h.logger.ErrorContext(ctx, "VotePoll failed", "err", err)
		return nil, apierror.Internal("failed to vote on poll")
	}
	metrics.TeamEvents.WithLabelValues("poll", "update").Inc()
	return gen.VotePoll200JSONResponse(poll), nil
}

// DeletePoll removes a poll.
func (h *Handler) DeletePoll(ctx context.Context, req gen.DeletePollRequestObject) (gen.DeletePollResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.Delete(ctx, req.PollId, req.TeamId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("poll not found")
		}
		h.logger.ErrorContext(ctx, "DeletePoll failed", "err", err)
		return nil, apierror.Internal("failed to delete poll")
	}
	metrics.TeamEvents.WithLabelValues("poll", "delete").Inc()
	return gen.DeletePoll204Response{}, nil
}
