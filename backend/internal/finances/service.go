package finances

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

// financeRepo is the interface the Service relies on.
type financeRepo interface {
	ListTransactions(ctx context.Context, teamID uuid.UUID) ([]TransactionRow, error)
	CreateTransaction(ctx context.Context, teamID uuid.UUID, txType, title string, amount float64, date time.Time, category *string) (*TransactionRow, error)
	UpdateTransaction(ctx context.Context, id uuid.UUID, patch TransactionPatch) (*TransactionRow, error)
	DeleteTransaction(ctx context.Context, id uuid.UUID) error

	ListPenalties(ctx context.Context, teamID uuid.UUID) ([]PenaltyRow, error)
	CreatePenalty(ctx context.Context, teamID uuid.UUID, label string, amount float64) (*PenaltyRow, error)
	UpdatePenalty(ctx context.Context, id uuid.UUID, patch PenaltyPatch) (*PenaltyRow, error)
	DeletePenalty(ctx context.Context, id uuid.UUID) error

	ListAssignments(ctx context.Context, teamID uuid.UUID) ([]PenaltyAssignmentRow, error)
	CreateAssignment(ctx context.Context, teamID, userID, penaltyID uuid.UUID) (*PenaltyAssignmentRow, error)
	DeleteAssignment(ctx context.Context, id uuid.UUID) error
	ToggleAssignmentPaid(ctx context.Context, id uuid.UUID) (*PenaltyAssignmentRow, error)

	ListContributions(ctx context.Context, teamID uuid.UUID) ([]ContributionRow, error)
	UpdateContribution(ctx context.Context, id uuid.UUID, patch ContributionPatch) (*ContributionRow, error)
	ToggleContributionStatus(ctx context.Context, id uuid.UUID) (*ContributionRow, error)

	ListOpenPenaltiesByUser(ctx context.Context, teamID uuid.UUID) ([]OpenPenaltyAggregate, error)
}

// Service implements finance business logic.
type Service struct {
	repo financeRepo
}

// NewService creates a new Service.
func NewService(repo financeRepo) *Service {
	return &Service{repo: repo}
}

// ─── Overview ─────────────────────────────────────────────────────────────────

// GetOverview assembles the full FinanceOverview for a team.
func (s *Service) GetOverview(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error) {
	txs, err := s.repo.ListTransactions(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.GetOverview transactions: %w", err)
	}

	penalties, err := s.repo.ListPenalties(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.GetOverview penalties: %w", err)
	}

	assignments, err := s.repo.ListAssignments(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.GetOverview assignments: %w", err)
	}

	contributions, err := s.repo.ListContributions(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.GetOverview contributions: %w", err)
	}

	openByUser, err := s.repo.ListOpenPenaltiesByUser(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.GetOverview open penalties: %w", err)
	}

	var income, expense float64
	genTxs := make([]gen.Transaction, 0, len(txs))
	for _, t := range txs {
		genTxs = append(genTxs, toGenTransaction(t))
		if t.Type == "income" {
			income += t.Amount
		} else {
			expense += t.Amount
		}
	}

	genPenalties := make([]gen.Penalty, 0, len(penalties))
	for _, p := range penalties {
		genPenalties = append(genPenalties, toGenPenalty(p))
	}

	genAssignments := make([]gen.PenaltyAssignment, 0, len(assignments))
	for _, a := range assignments {
		genAssignments = append(genAssignments, toGenAssignment(a))
	}

	genContributions := make([]gen.Contribution, 0, len(contributions))
	contribOpen := 0
	for _, c := range contributions {
		genContributions = append(genContributions, toGenContribution(c))
		if c.Status == "open" {
			contribOpen++
		}
	}

	genOpen := make([]gen.OpenPenalty, 0, len(openByUser))
	var openPenaltySum float64
	for _, o := range openByUser {
		hp := o.HasPhoto
		genOpen = append(genOpen, gen.OpenPenalty{
			UserId:      openapi_types.UUID(o.UserID),
			Name:        o.Name,
			AvatarColor: o.AvatarColor,
			HasPhoto:    &hp,
			Amount:      o.TotalAmount,
		})
		openPenaltySum += o.TotalAmount
	}

	return &gen.FinanceOverview{
		Transactions:   genTxs,
		Penalties:      genPenalties,
		Assignments:    genAssignments,
		Contributions:  genContributions,
		OpenPenalties:  genOpen,
		OpenPenaltySum: openPenaltySum,
		Income:         income,
		Expense:        expense,
		Balance:        income - expense,
		ContribOpen:    contribOpen,
	}, nil
}

// ─── Transactions ─────────────────────────────────────────────────────────────

// CreateTransaction creates a new transaction (date defaults to today).
func (s *Service) CreateTransaction(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
	t, err := s.repo.CreateTransaction(ctx, teamID, string(body.Type), body.Title, body.Amount, time.Now(), body.Category)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateTransaction: %w", err)
	}
	result := toGenTransaction(*t)
	return &result, nil
}

// UpdateTransaction applies a patch to a transaction.
func (s *Service) UpdateTransaction(ctx context.Context, id uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
	patch := TransactionPatch{}
	if body.Type != nil {
		st := string(*body.Type)
		patch.Type = &st
	}
	if body.Title != nil {
		patch.Title = body.Title
	}
	if body.Amount != nil {
		patch.Amount = body.Amount
	}
	if body.Category != nil {
		patch.Category = body.Category
	}
	t, err := s.repo.UpdateTransaction(ctx, id, patch)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.UpdateTransaction: %w", err)
	}
	result := toGenTransaction(*t)
	return &result, nil
}

// DeleteTransaction deletes a transaction.
func (s *Service) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTransaction(ctx, id)
}

// ─── Penalties ────────────────────────────────────────────────────────────────

// CreatePenalty creates a new penalty definition.
func (s *Service) CreatePenalty(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	p, err := s.repo.CreatePenalty(ctx, teamID, body.Label, body.Amount)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreatePenalty: %w", err)
	}
	result := toGenPenalty(*p)
	return &result, nil
}

// UpdatePenalty applies a patch to a penalty definition.
func (s *Service) UpdatePenalty(ctx context.Context, id uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	patch := PenaltyPatch{Label: body.Label, Amount: body.Amount}
	p, err := s.repo.UpdatePenalty(ctx, id, patch)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.UpdatePenalty: %w", err)
	}
	result := toGenPenalty(*p)
	return &result, nil
}

// DeletePenalty deletes a penalty definition.
func (s *Service) DeletePenalty(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePenalty(ctx, id)
}

// ─── Assignments ──────────────────────────────────────────────────────────────

// CreateAssignment creates a penalty assignment.
func (s *Service) CreateAssignment(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error) {
	penaltyID := uuid.UUID(body.PenaltyId)
	userID := uuid.UUID(body.UserId)
	a, err := s.repo.CreateAssignment(ctx, teamID, userID, penaltyID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateAssignment: %w", err)
	}
	// Reload with joined data.
	assignments, err := s.repo.ListAssignments(ctx, teamID)
	if err != nil {
		result := toGenAssignment(*a)
		return &result, nil
	}
	for _, full := range assignments {
		if full.ID == a.ID {
			result := toGenAssignment(full)
			return &result, nil
		}
	}
	result := toGenAssignment(*a)
	return &result, nil
}

// DeleteAssignment deletes a penalty assignment.
func (s *Service) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteAssignment(ctx, id)
}

// ToggleAssignmentPaid flips the paid flag.
func (s *Service) ToggleAssignmentPaid(ctx context.Context, teamID, id uuid.UUID) (*gen.PenaltyAssignment, error) {
	a, err := s.repo.ToggleAssignmentPaid(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.ToggleAssignmentPaid: %w", err)
	}
	// Reload with joined data.
	assignments, err := s.repo.ListAssignments(ctx, teamID)
	if err != nil {
		result := toGenAssignment(*a)
		return &result, nil
	}
	for _, full := range assignments {
		if full.ID == a.ID {
			result := toGenAssignment(full)
			return &result, nil
		}
	}
	result := toGenAssignment(*a)
	return &result, nil
}

// ─── Contributions ────────────────────────────────────────────────────────────

// UpdateContribution applies a patch to a contribution.
func (s *Service) UpdateContribution(ctx context.Context, id uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
	patch := ContributionPatch{Label: body.Label, Amount: body.Amount}
	c, err := s.repo.UpdateContribution(ctx, id, patch)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.UpdateContribution: %w", err)
	}
	result := toGenContribution(*c)
	return &result, nil
}

// ToggleContribution flips the contribution status between open and paid.
func (s *Service) ToggleContribution(ctx context.Context, id uuid.UUID) (*gen.Contribution, error) {
	c, err := s.repo.ToggleContributionStatus(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.ToggleContribution: %w", err)
	}
	result := toGenContribution(*c)
	return &result, nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toGenTransaction(t TransactionRow) gen.Transaction {
	return gen.Transaction{
		Id:       openapi_types.UUID(t.ID),
		TeamId:   openapi_types.UUID(t.TeamID),
		Type:     gen.TransactionType(t.Type),
		Title:    t.Title,
		Amount:   t.Amount,
		Date:     openapi_types.Date{Time: t.Date},
		Category: t.Category,
	}
}

func toGenPenalty(p PenaltyRow) gen.Penalty {
	return gen.Penalty{
		Id:     openapi_types.UUID(p.ID),
		TeamId: openapi_types.UUID(p.TeamID),
		Label:  p.Label,
		Amount: p.Amount,
	}
}

func toGenAssignment(a PenaltyAssignmentRow) gen.PenaltyAssignment {
	return gen.PenaltyAssignment{
		Id:                openapi_types.UUID(a.ID),
		TeamId:            openapi_types.UUID(a.TeamID),
		UserId:            openapi_types.UUID(a.UserID),
		PenaltyId:         openapi_types.UUID(a.PenaltyID),
		Paid:              a.Paid,
		Date:              openapi_types.Date{Time: a.Date},
		Label:             a.PenaltyLabel,
		Amount:            a.PenaltyAmount,
		MemberName:        a.MemberName,
		MemberAvatarColor: a.MemberAvatarColor,
		HasPhoto:          a.HasPhoto,
	}
}

func toGenContribution(c ContributionRow) gen.Contribution {
	return gen.Contribution{
		Id:                openapi_types.UUID(c.ID),
		TeamId:            openapi_types.UUID(c.TeamID),
		UserId:            openapi_types.UUID(c.UserID),
		Month:             c.Month,
		Label:             c.Label,
		Amount:            c.Amount,
		Status:            gen.ContributionStatus(c.Status),
		MemberName:        c.MemberName,
		MemberAvatarColor: c.MemberAvatarColor,
		HasPhoto:          c.HasPhoto,
	}
}
