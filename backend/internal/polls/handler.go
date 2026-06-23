package polls

import (
	"context"
	"log/slog"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// pollService is the interface the Handler relies on.
type pollService interface {
	ListByTeam(ctx context.Context, teamID, currentUserID uuid.UUID) ([]gen.Poll, error)
	Create(ctx context.Context, teamID, creatorID uuid.UUID, body *gen.CreatePollRequest) (gen.Poll, error)
	Vote(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID) (gen.Poll, error)
	Delete(ctx context.Context, id uuid.UUID) error
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

// ListPolls returns all polls for the team.
func (h *Handler) ListPolls(ctx context.Context, req gen.ListPollsRequestObject) (gen.ListPollsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	polls, err := h.svc.ListByTeam(ctx, uuid.UUID(req.TeamId), user.Id)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListPolls failed", "err", err)
		return nil, apierror.Internal("failed to list polls")
	}
	return gen.ListPolls200JSONResponse(polls), nil
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
	poll, err := h.svc.Create(ctx, uuid.UUID(req.TeamId), user.Id, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreatePoll failed", "err", err)
		return nil, apierror.Internal("failed to create poll")
	}
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
	optionIDs := make([]uuid.UUID, 0, len(req.Body.OptionIds))
	for _, oid := range req.Body.OptionIds {
		optionIDs = append(optionIDs, uuid.UUID(openapi_types.UUID(oid)))
	}
	poll, err := h.svc.Vote(ctx, uuid.UUID(req.PollId), user.Id, optionIDs)
	if err != nil {
		h.logger.ErrorContext(ctx, "VotePoll failed", "err", err)
		return nil, apierror.Internal("failed to vote on poll")
	}
	return gen.VotePoll200JSONResponse(poll), nil
}

// DeletePoll removes a poll.
func (h *Handler) DeletePoll(ctx context.Context, req gen.DeletePollRequestObject) (gen.DeletePollResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.Delete(ctx, uuid.UUID(req.PollId)); err != nil {
		h.logger.ErrorContext(ctx, "DeletePoll failed", "err", err)
		return nil, apierror.Internal("failed to delete poll")
	}
	return gen.DeletePoll204Response{}, nil
}
