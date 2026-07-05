package finances

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// financeService is the interface the Handler relies on.
type financeService interface {
	GetOverview(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error)
	CreateTransaction(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error)
	UpdateTransaction(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error)
	DeleteTransaction(ctx context.Context, id, teamID uuid.UUID) error
	CreatePenalty(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error)
	UpdatePenalty(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error)
	DeletePenalty(ctx context.Context, id, teamID uuid.UUID) error
	CreateAssignment(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error)
	DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error
	ToggleAssignmentPaid(ctx context.Context, teamID, id uuid.UUID) (*gen.PenaltyAssignment, error)
	UpdateContribution(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error)
	ToggleContribution(ctx context.Context, id, teamID uuid.UUID) (*gen.Contribution, error)
}

// Handler implements the finance-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    financeService
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler. al is the shared audit logger; when nil a
// log-only logger is created from logger.
func NewHandler(svc financeService, logger *slog.Logger, al *audit.Logger) *Handler {
	if al == nil {
		al = audit.New(logger)
	}
	return &Handler{svc: svc, logger: logger, audit: al}
}

// recordFinance emits a finance.mutation audit event for the acting user,
// tagging the specific write with an "operation" attribute.
func (h *Handler) recordFinance(ctx context.Context, operation string, attrs ...slog.Attr) {
	actor := ""
	if u, ok := auth.UserFromContext(ctx); ok {
		actor = u.Id.String()
	}
	h.audit.Record(ctx, audit.EventFinanceMutation, audit.Success, actor,
		append([]slog.Attr{slog.String("operation", operation)}, attrs...)...)
}

// recordFinanceFailure emits a failure audit event for a finance mutation.
func (h *Handler) recordFinanceFailure(ctx context.Context, operation, reason string) {
	actor := ""
	if u, ok := auth.UserFromContext(ctx); ok {
		actor = u.Id.String()
	}
	h.audit.Record(ctx, audit.EventFinanceMutation, audit.Failure, actor,
		slog.String("operation", operation), slog.String("reason", reason))
}

// GetFinanceOverview returns the full finance overview for a team.
func (h *Handler) GetFinanceOverview(ctx context.Context, req gen.GetFinanceOverviewRequestObject) (gen.GetFinanceOverviewResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	overview, err := h.svc.GetOverview(ctx, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetFinanceOverview failed", "err", err)
		return nil, apierror.Internal("failed to get finance overview")
	}
	return gen.GetFinanceOverview200JSONResponse(*overview), nil
}

// CreateTransaction creates a new financial transaction.
func (h *Handler) CreateTransaction(ctx context.Context, req gen.CreateTransactionRequestObject) (gen.CreateTransactionResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.RequireNonEmpty(req.Body.Title, "title"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.MaxLen(req.Body.Title, 255, "title"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if req.Body.Category != nil {
		if err := validate.MaxLen(*req.Body.Category, 255, "category"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if !req.Body.Type.Valid() {
		return nil, apierror.BadRequest("type: not a valid transaction type")
	}
	if err := validate.PositiveAmount(req.Body.Amount, "amount"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	t, err := h.svc.CreateTransaction(ctx, req.TeamId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateTransaction failed", "err", err)
		h.recordFinanceFailure(ctx, "transaction.create", "internal error")
		return nil, apierror.Internal("failed to create transaction")
	}
	h.recordFinance(ctx, "transaction.create",
		slog.String("teamId", req.TeamId.String()), slog.String("transactionId", t.Id.String()))
	metrics.TeamEvents.WithLabelValues("finance", "create").Inc()
	return gen.CreateTransaction201JSONResponse(*t), nil
}

// UpdateTransaction applies a partial update to a transaction.
func (h *Handler) UpdateTransaction(ctx context.Context, req gen.UpdateTransactionRequestObject) (gen.UpdateTransactionResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.Title != nil {
		if err := validate.RequireNonEmpty(*req.Body.Title, "title"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
		if err := validate.MaxLen(*req.Body.Title, 255, "title"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Category != nil {
		if err := validate.MaxLen(*req.Body.Category, 255, "category"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Type != nil && !req.Body.Type.Valid() {
		return nil, apierror.BadRequest("type: not a valid transaction type")
	}
	if req.Body.Amount != nil {
		if err := validate.PositiveAmount(*req.Body.Amount, "amount"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	t, err := h.svc.UpdateTransaction(ctx, req.TransactionId, req.TeamId, req.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "transaction.update", "not found")
			return nil, apierror.NotFound("transaction not found")
		}
		h.recordFinanceFailure(ctx, "transaction.update", "internal error")
		h.logger.ErrorContext(ctx, "UpdateTransaction failed", "err", err)
		return nil, apierror.Internal("failed to update transaction")
	}
	h.recordFinance(ctx, "transaction.update", slog.String("transactionId", req.TransactionId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "update").Inc()
	return gen.UpdateTransaction200JSONResponse(*t), nil
}

// DeleteTransaction removes a transaction.
func (h *Handler) DeleteTransaction(ctx context.Context, req gen.DeleteTransactionRequestObject) (gen.DeleteTransactionResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeleteTransaction(ctx, req.TransactionId, req.TeamId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "transaction.delete", "not found")
			return nil, apierror.NotFound("transaction not found")
		}
		h.recordFinanceFailure(ctx, "transaction.delete", "internal error")
		h.logger.ErrorContext(ctx, "DeleteTransaction failed", "err", err)
		return nil, apierror.Internal("failed to delete transaction")
	}
	h.recordFinance(ctx, "transaction.delete", slog.String("transactionId", req.TransactionId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "delete").Inc()
	return gen.DeleteTransaction204Response{}, nil
}

// CreatePenalty creates a new penalty definition.
func (h *Handler) CreatePenalty(ctx context.Context, req gen.CreatePenaltyRequestObject) (gen.CreatePenaltyResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.RequireNonEmpty(req.Body.Label, "label"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.MaxLen(req.Body.Label, 255, "label"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.PositiveAmount(req.Body.Amount, "amount"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	p, err := h.svc.CreatePenalty(ctx, req.TeamId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreatePenalty failed", "err", err)
		h.recordFinanceFailure(ctx, "penalty.create", "internal error")
		return nil, apierror.Internal("failed to create penalty")
	}
	h.recordFinance(ctx, "penalty.create",
		slog.String("teamId", req.TeamId.String()), slog.String("penaltyId", p.Id.String()))
	metrics.TeamEvents.WithLabelValues("finance", "create").Inc()
	return gen.CreatePenalty201JSONResponse(*p), nil
}

// UpdatePenalty applies a partial update to a penalty definition.
func (h *Handler) UpdatePenalty(ctx context.Context, req gen.UpdatePenaltyRequestObject) (gen.UpdatePenaltyResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.Label != nil {
		if err := validate.RequireNonEmpty(*req.Body.Label, "label"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
		if err := validate.MaxLen(*req.Body.Label, 255, "label"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Amount != nil {
		if err := validate.PositiveAmount(*req.Body.Amount, "amount"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	p, err := h.svc.UpdatePenalty(ctx, req.PenaltyId, req.TeamId, req.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "penalty.update", "not found")
			return nil, apierror.NotFound("penalty not found")
		}
		h.recordFinanceFailure(ctx, "penalty.update", "internal error")
		h.logger.ErrorContext(ctx, "UpdatePenalty failed", "err", err)
		return nil, apierror.Internal("failed to update penalty")
	}
	h.recordFinance(ctx, "penalty.update", slog.String("penaltyId", req.PenaltyId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "update").Inc()
	return gen.UpdatePenalty200JSONResponse(*p), nil
}

// DeletePenalty removes a penalty definition.
func (h *Handler) DeletePenalty(ctx context.Context, req gen.DeletePenaltyRequestObject) (gen.DeletePenaltyResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeletePenalty(ctx, req.PenaltyId, req.TeamId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "penalty.delete", "not found")
			return nil, apierror.NotFound("penalty not found")
		}
		h.recordFinanceFailure(ctx, "penalty.delete", "internal error")
		h.logger.ErrorContext(ctx, "DeletePenalty failed", "err", err)
		return nil, apierror.Internal("failed to delete penalty")
	}
	h.recordFinance(ctx, "penalty.delete", slog.String("penaltyId", req.PenaltyId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "delete").Inc()
	return gen.DeletePenalty204Response{}, nil
}

// CreatePenaltyAssignment assigns a penalty to a team member.
func (h *Handler) CreatePenaltyAssignment(ctx context.Context, req gen.CreatePenaltyAssignmentRequestObject) (gen.CreatePenaltyAssignmentResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	a, err := h.svc.CreateAssignment(ctx, req.TeamId, req.Body)
	if err != nil {
		if errors.Is(err, ErrPenaltyNotInTeam) || errors.Is(err, ErrUserNotInTeam) {
			h.recordFinanceFailure(ctx, "assignment.create", err.Error())
			return nil, apierror.UnprocessableEntity(err.Error())
		}
		h.recordFinanceFailure(ctx, "assignment.create", "internal error")
		h.logger.ErrorContext(ctx, "CreatePenaltyAssignment failed", "err", err)
		return nil, apierror.Internal("failed to create penalty assignment")
	}
	h.recordFinance(ctx, "assignment.create",
		slog.String("teamId", req.TeamId.String()), slog.String("assignmentId", a.Id.String()))
	metrics.TeamEvents.WithLabelValues("finance", "create").Inc()
	return gen.CreatePenaltyAssignment201JSONResponse(*a), nil
}

// DeletePenaltyAssignment removes a penalty assignment.
func (h *Handler) DeletePenaltyAssignment(ctx context.Context, req gen.DeletePenaltyAssignmentRequestObject) (gen.DeletePenaltyAssignmentResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeleteAssignment(ctx, req.AssignmentId, req.TeamId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "assignment.delete", "not found")
			return nil, apierror.NotFound("penalty assignment not found")
		}
		h.recordFinanceFailure(ctx, "assignment.delete", "internal error")
		h.logger.ErrorContext(ctx, "DeletePenaltyAssignment failed", "err", err)
		return nil, apierror.Internal("failed to delete penalty assignment")
	}
	h.recordFinance(ctx, "assignment.delete", slog.String("assignmentId", req.AssignmentId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "delete").Inc()
	return gen.DeletePenaltyAssignment204Response{}, nil
}

// TogglePenaltyPaid flips the paid flag on a penalty assignment.
func (h *Handler) TogglePenaltyPaid(ctx context.Context, req gen.TogglePenaltyPaidRequestObject) (gen.TogglePenaltyPaidResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	a, err := h.svc.ToggleAssignmentPaid(ctx, req.TeamId, req.AssignmentId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "assignment.toggle_paid", "not found")
			return nil, apierror.NotFound("penalty assignment not found")
		}
		h.recordFinanceFailure(ctx, "assignment.toggle_paid", "internal error")
		h.logger.ErrorContext(ctx, "TogglePenaltyPaid failed", "err", err)
		return nil, apierror.Internal("failed to toggle penalty paid status")
	}
	h.recordFinance(ctx, "assignment.toggle_paid",
		slog.String("teamId", req.TeamId.String()), slog.String("assignmentId", req.AssignmentId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "update").Inc()
	return gen.TogglePenaltyPaid200JSONResponse(*a), nil
}

// UpdateContribution applies a partial update to a contribution.
func (h *Handler) UpdateContribution(ctx context.Context, req gen.UpdateContributionRequestObject) (gen.UpdateContributionResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.Label != nil {
		if err := validate.RequireNonEmpty(*req.Body.Label, "label"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
		if err := validate.MaxLen(*req.Body.Label, 255, "label"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Amount != nil {
		if err := validate.PositiveAmount(*req.Body.Amount, "amount"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	c, err := h.svc.UpdateContribution(ctx, req.ContributionId, req.TeamId, req.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "contribution.update", "not found")
			return nil, apierror.NotFound("contribution not found")
		}
		h.recordFinanceFailure(ctx, "contribution.update", "internal error")
		h.logger.ErrorContext(ctx, "UpdateContribution failed", "err", err)
		return nil, apierror.Internal("failed to update contribution")
	}
	h.recordFinance(ctx, "contribution.update", slog.String("contributionId", req.ContributionId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "update").Inc()
	return gen.UpdateContribution200JSONResponse(*c), nil
}

// ToggleContribution flips a contribution's status between open and paid.
func (h *Handler) ToggleContribution(ctx context.Context, req gen.ToggleContributionRequestObject) (gen.ToggleContributionResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	c, err := h.svc.ToggleContribution(ctx, req.ContributionId, req.TeamId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordFinanceFailure(ctx, "contribution.toggle", "not found")
			return nil, apierror.NotFound("contribution not found")
		}
		h.recordFinanceFailure(ctx, "contribution.toggle", "internal error")
		h.logger.ErrorContext(ctx, "ToggleContribution failed", "err", err)
		return nil, apierror.Internal("failed to toggle contribution status")
	}
	h.recordFinance(ctx, "contribution.toggle", slog.String("contributionId", req.ContributionId.String()))
	metrics.TeamEvents.WithLabelValues("finance", "update").Inc()
	return gen.ToggleContribution200JSONResponse(*c), nil
}
