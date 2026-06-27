package finances

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

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
	UpdateTransaction(ctx context.Context, id uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error)
	DeleteTransaction(ctx context.Context, id uuid.UUID) error
	CreatePenalty(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error)
	UpdatePenalty(ctx context.Context, id uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error)
	DeletePenalty(ctx context.Context, id uuid.UUID) error
	CreateAssignment(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error)
	DeleteAssignment(ctx context.Context, id uuid.UUID) error
	ToggleAssignmentPaid(ctx context.Context, teamID, id uuid.UUID) (*gen.PenaltyAssignment, error)
	UpdateContribution(ctx context.Context, id uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error)
	ToggleContribution(ctx context.Context, id uuid.UUID) (*gen.Contribution, error)
}

// Handler implements the finance-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    financeService
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc financeService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger, audit: audit.New(logger)}
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
	t, err := h.svc.CreateTransaction(ctx, req.TeamId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateTransaction failed", "err", err)
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
	t, err := h.svc.UpdateTransaction(ctx, req.TransactionId, req.Body)
	if err != nil {
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
	if err := h.svc.DeleteTransaction(ctx, req.TransactionId); err != nil {
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
	p, err := h.svc.CreatePenalty(ctx, req.TeamId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreatePenalty failed", "err", err)
		return nil, apierror.Internal("failed to create penalty")
	}
	h.recordFinance(ctx, "penalty.create",
		slog.String("teamId", req.TeamId.String()), slog.String("penaltyId", p.Id.String()))
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
	p, err := h.svc.UpdatePenalty(ctx, req.PenaltyId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "UpdatePenalty failed", "err", err)
		return nil, apierror.Internal("failed to update penalty")
	}
	h.recordFinance(ctx, "penalty.update", slog.String("penaltyId", req.PenaltyId.String()))
	return gen.UpdatePenalty200JSONResponse(*p), nil
}

// DeletePenalty removes a penalty definition.
func (h *Handler) DeletePenalty(ctx context.Context, req gen.DeletePenaltyRequestObject) (gen.DeletePenaltyResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeletePenalty(ctx, req.PenaltyId); err != nil {
		h.logger.ErrorContext(ctx, "DeletePenalty failed", "err", err)
		return nil, apierror.Internal("failed to delete penalty")
	}
	h.recordFinance(ctx, "penalty.delete", slog.String("penaltyId", req.PenaltyId.String()))
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
		h.logger.ErrorContext(ctx, "CreatePenaltyAssignment failed", "err", err)
		return nil, apierror.Internal("failed to create penalty assignment")
	}
	h.recordFinance(ctx, "assignment.create",
		slog.String("teamId", req.TeamId.String()), slog.String("assignmentId", a.Id.String()))
	return gen.CreatePenaltyAssignment201JSONResponse(*a), nil
}

// DeletePenaltyAssignment removes a penalty assignment.
func (h *Handler) DeletePenaltyAssignment(ctx context.Context, req gen.DeletePenaltyAssignmentRequestObject) (gen.DeletePenaltyAssignmentResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeleteAssignment(ctx, req.AssignmentId); err != nil {
		h.logger.ErrorContext(ctx, "DeletePenaltyAssignment failed", "err", err)
		return nil, apierror.Internal("failed to delete penalty assignment")
	}
	h.recordFinance(ctx, "assignment.delete", slog.String("assignmentId", req.AssignmentId.String()))
	return gen.DeletePenaltyAssignment204Response{}, nil
}

// TogglePenaltyPaid flips the paid flag on a penalty assignment.
func (h *Handler) TogglePenaltyPaid(ctx context.Context, req gen.TogglePenaltyPaidRequestObject) (gen.TogglePenaltyPaidResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	a, err := h.svc.ToggleAssignmentPaid(ctx, req.TeamId, req.AssignmentId)
	if err != nil {
		h.logger.ErrorContext(ctx, "TogglePenaltyPaid failed", "err", err)
		return nil, apierror.Internal("failed to toggle penalty paid status")
	}
	h.recordFinance(ctx, "assignment.toggle_paid",
		slog.String("teamId", req.TeamId.String()), slog.String("assignmentId", req.AssignmentId.String()))
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
	c, err := h.svc.UpdateContribution(ctx, req.ContributionId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "UpdateContribution failed", "err", err)
		return nil, apierror.Internal("failed to update contribution")
	}
	h.recordFinance(ctx, "contribution.update", slog.String("contributionId", req.ContributionId.String()))
	return gen.UpdateContribution200JSONResponse(*c), nil
}

// ToggleContribution flips a contribution's status between open and paid.
func (h *Handler) ToggleContribution(ctx context.Context, req gen.ToggleContributionRequestObject) (gen.ToggleContributionResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	c, err := h.svc.ToggleContribution(ctx, req.ContributionId)
	if err != nil {
		h.logger.ErrorContext(ctx, "ToggleContribution failed", "err", err)
		return nil, apierror.Internal("failed to toggle contribution status")
	}
	h.recordFinance(ctx, "contribution.toggle", slog.String("contributionId", req.ContributionId.String()))
	return gen.ToggleContribution200JSONResponse(*c), nil
}
