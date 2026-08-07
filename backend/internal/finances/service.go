package finances

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// Sentinel errors for cross-team validation.
var (
	ErrPenaltyNotInTeam      = errors.New("penalty does not belong to this team")
	ErrUserNotInTeam         = errors.New("user is not a member of this team")
	ErrContributionNotInTeam = errors.New("contribution does not belong to this team")
	// ErrContributionRequiresIncome is returned when a transaction's
	// contributionId is set but its type isn't "income" -- linking a fee
	// payment to an expense doesn't make sense (see design.md's "Linking is
	// income-only" decision).
	ErrContributionRequiresIncome      = errors.New("a transaction linked to a contribution must have type income")
	ErrPenaltyAssignmentNotInTeam      = errors.New("penalty assignment does not belong to this team")
	ErrPenaltyAssignmentRequiresIncome = errors.New("a transaction linked to a penalty assignment must have type income")
	ErrTransactionLinksMultipleTargets = errors.New("a transaction can be linked to at most one of contributionId or penaltyAssignmentId")
)

// ErrTooManyTransactions / ErrTooManyAssignments are returned once a team
// hits maxTransactionsPerTeam / maxAssignmentsPerTeam.
var (
	ErrTooManyTransactions  = fmt.Errorf("team has reached the maximum of %d transactions", maxTransactionsPerTeam)
	ErrTooManyAssignments   = fmt.Errorf("team has reached the maximum of %d penalty assignments", maxAssignmentsPerTeam)
	ErrTooManyPenalties     = fmt.Errorf("team has reached the maximum of %d penalty definitions", maxPenaltiesPerTeam)
	ErrTooManyContributions = fmt.Errorf("team has reached the maximum of %d contributions", maxContributionsPerTeam)
)

// maxTransactionsPerTeam / maxAssignmentsPerTeam cap how many rows a single
// team can accumulate in these tables. Both are read on every finance
// overview via unbounded aggregate queries (SumTransactions,
// ListOpenPenaltiesByUser) under a fixed 5s query timeout -- with no cap, a
// team member holding only finances:write (not necessarily an owner/admin)
// could flood either table well past what those aggregates can scan within
// that timeout, degrading or hard-failing the finance overview for every
// member of the team, not just the one flooding it. Both limits are
// generous enough that no legitimate club's multi-year financial history
// should ever approach them -- this exists to stop runaway/malicious
// creation, not to constrain real usage.
//
// maxPenaltiesPerTeam is much smaller: unlike transactions/assignments,
// which are naturally bounded by real financial activity, penalty
// definitions are just a small, hand-curated catalog of fine types (e.g.
// "Zu spät", "Fehltraining") -- a legitimate club needs at most a few dozen.
// GetOverview reads ListPenalties unconditionally too (see ListPenalties'
// doc comment), so it needs the same flood protection as the other lists,
// just at a scale matching what this table is actually for.
const (
	maxTransactionsPerTeam = 100_000
	maxAssignmentsPerTeam  = 100_000
	maxPenaltiesPerTeam    = 500
	// maxContributionsPerTeam mirrors maxTransactionsPerTeam/maxAssignmentsPerTeam:
	// contributions are now created directly by the treasurer (fan-out create),
	// not auto-generated, but a member holding only finances:write could still
	// flood the table via repeated fan-out creates -- this bounds that the same
	// way the other finance tables are bounded, well past what any real club's
	// fee history should ever approach.
	maxContributionsPerTeam = 100_000
)

// financeRepo is the interface the Service relies on.
type financeRepo interface {
	ListTransactions(ctx context.Context, teamID uuid.UUID) ([]TransactionRow, error)
	ListTransactionsPage(ctx context.Context, teamID uuid.UUID, limit int, cur *TxCursor) ([]TransactionRow, error)
	SumTransactions(ctx context.Context, teamID uuid.UUID) (income, expense int64, err error)
	CountTransactions(ctx context.Context, teamID uuid.UUID) (int, error)
	CreateTransaction(ctx context.Context, teamID uuid.UUID, txType, title string, amount int64, date time.Time, category *string, contributionID, penaltyAssignmentID *uuid.UUID) (*TransactionRow, error)
	UpdateTransaction(ctx context.Context, id, teamID uuid.UUID, patch TransactionPatch) (*TransactionRow, error)
	DeleteTransaction(ctx context.Context, id, teamID uuid.UUID) error

	ListPenalties(ctx context.Context, teamID uuid.UUID) ([]PenaltyRow, error)
	CountPenalties(ctx context.Context, teamID uuid.UUID) (int, error)
	CreatePenalty(ctx context.Context, teamID uuid.UUID, label string, amount int64) (*PenaltyRow, error)
	UpdatePenalty(ctx context.Context, id, teamID uuid.UUID, patch PenaltyPatch) (*PenaltyRow, error)
	DeletePenalty(ctx context.Context, id, teamID uuid.UUID) error
	PenaltyBelongsToTeam(ctx context.Context, penaltyID, teamID uuid.UUID) (bool, error)

	ListAssignments(ctx context.Context, teamID uuid.UUID) ([]PenaltyAssignmentRow, error)
	GetAssignmentByID(ctx context.Context, id, teamID uuid.UUID) (*PenaltyAssignmentRow, error)
	CountAssignments(ctx context.Context, teamID uuid.UUID) (int, error)
	CreateAssignment(ctx context.Context, teamID, userID, penaltyID uuid.UUID, date time.Time, note *string) (*PenaltyAssignmentRow, error)
	DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error
	PenaltyAssignmentBelongsToTeam(ctx context.Context, assignmentID, teamID uuid.UUID) (bool, error)
	UserIsMemberOfTeam(ctx context.Context, userID, teamID uuid.UUID) (bool, error)

	ListContributions(ctx context.Context, teamID uuid.UUID) ([]ContributionRow, error)
	CountContributions(ctx context.Context, teamID uuid.UUID) (int, error)
	CountOpenContributions(ctx context.Context, teamID uuid.UUID) (int, error)
	CreateContributions(ctx context.Context, teamID uuid.UUID, name string, amount int64, dueDate *time.Time, userIDs []uuid.UUID) ([]ContributionRow, error)
	UpdateContribution(ctx context.Context, id, teamID uuid.UUID, patch ContributionPatch) (*ContributionRow, error)
	DeleteContribution(ctx context.Context, id, teamID uuid.UUID) error
	ContributionBelongsToTeam(ctx context.Context, contributionID, teamID uuid.UUID) (bool, error)

	ListOpenPenaltiesByUser(ctx context.Context, teamID uuid.UUID) ([]OpenPenaltyAggregate, error)

	WithReadTx(ctx context.Context, fn func(OverviewReader) error) error
}

// Service implements finance business logic.
type Service struct {
	repo   financeRepo
	pager  *pagination.Paginator
	logger *slog.Logger
}

// NewService creates a new Service. pager encodes/decodes the keyset cursors
// used by ListTransactions; pass the shared Paginator so signed-cursor config
// (PAGINATION_HMAC_KEY) applies uniformly across list endpoints.
func NewService(repo financeRepo, pager *pagination.Paginator, logger *slog.Logger) *Service {
	return &Service{repo: repo, pager: pager, logger: logger}
}

// ─── Overview ─────────────────────────────────────────────────────────────────

// GetOverview assembles the FinanceOverview for a team. Display lists
// (transactions, assignments, contributions) are capped at maxOverviewRows by
// the repository; the income/expense/balance and open-contribution figures
// are computed via dedicated aggregate queries so they stay accurate
// regardless of that cap.
func (s *Service) GetOverview(ctx context.Context, teamID uuid.UUID) (*gen.FinanceOverview, error) {
	var (
		txs           []TransactionRow
		income        int64
		expense       int64
		penalties     []PenaltyRow
		assignments   []PenaltyAssignmentRow
		contributions []ContributionRow
		contribOpen   int
		openByUser    []OpenPenaltyAggregate
	)

	// Run every read inside one read-only transaction so the display lists and
	// the separately computed aggregates observe a single consistent snapshot,
	// instead of possibly drifting under concurrent writes.
	err := s.repo.WithReadTx(ctx, func(repo OverviewReader) error {
		var err error

		txs, err = repo.ListTransactions(ctx, teamID)
		if err != nil {
			return fmt.Errorf("transactions: %w", err)
		}

		income, expense, err = repo.SumTransactions(ctx, teamID)
		if err != nil {
			return fmt.Errorf("transaction totals: %w", err)
		}

		penalties, err = repo.ListPenalties(ctx, teamID)
		if err != nil {
			return fmt.Errorf("penalties: %w", err)
		}

		assignments, err = repo.ListAssignments(ctx, teamID)
		if err != nil {
			return fmt.Errorf("assignments: %w", err)
		}

		contributions, err = repo.ListContributions(ctx, teamID)
		if err != nil {
			return fmt.Errorf("contributions: %w", err)
		}

		contribOpen, err = repo.CountOpenContributions(ctx, teamID)
		if err != nil {
			return fmt.Errorf("open contributions: %w", err)
		}

		openByUser, err = repo.ListOpenPenaltiesByUser(ctx, teamID)
		if err != nil {
			return fmt.Errorf("open penalties: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("finances.Service.GetOverview: %w", err)
	}

	genTxs := make([]gen.Transaction, 0, len(txs))
	for _, t := range txs {
		genTxs = append(genTxs, toGenTransaction(t))
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
	for _, c := range contributions {
		genContributions = append(genContributions, toGenContribution(c))
	}

	genOpen := make([]gen.OpenPenalty, 0, len(openByUser))
	var openPenaltySum int64
	for _, o := range openByUser {
		hp := o.HasPhoto
		genOpen = append(genOpen, gen.OpenPenalty{
			UserId:      o.UserID,
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

// ListTransactions returns a keyset page of transactions plus the cursor for
// the next page (nil on the last page). cursor is the opaque token from a prior
// page ("" = first page). Unlike GetOverview's embedded transaction list, this
// has no hard visibility cap — the whole history is reachable by paging.
func (s *Service) ListTransactions(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.Transaction, *string, error) {
	var cur *TxCursor
	var decoded TxCursor
	if ok, err := s.pager.Decode(cursor, &decoded); err != nil {
		return nil, nil, fmt.Errorf("finances.Service.ListTransactions: %w", err)
	} else if ok {
		cur = &decoded
	}

	// Fetch one extra row to detect whether a further page exists.
	rows, err := s.repo.ListTransactionsPage(ctx, teamID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("finances.Service.ListTransactions: %w", err)
	}

	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		token, err := s.pager.Encode(TxCursor{Date: last.Date, CreatedAt: last.CreatedAt, ID: last.ID})
		if err != nil {
			return nil, nil, fmt.Errorf("finances.Service.ListTransactions: %w", err)
		}
		next = &token
	}

	result := make([]gen.Transaction, 0, len(rows))
	for _, t := range rows {
		result = append(result, toGenTransaction(t))
	}
	return result, next, nil
}

// CreateTransaction creates a new transaction (date defaults to today when
// the client omits it). A transaction may link to at most one of
// ContributionId / PenaltyAssignmentId (ErrTransactionLinksMultipleTargets
// otherwise); whichever is set must have type income
// (ErrContributionRequiresIncome / ErrPenaltyAssignmentRequiresIncome
// otherwise) and must belong to teamID (ErrContributionNotInTeam /
// ErrPenaltyAssignmentNotInTeam otherwise) -- see design.md's "Linking is
// income-only, and only enforced at creation".
func (s *Service) CreateTransaction(ctx context.Context, teamID uuid.UUID, body *gen.CreateTransactionJSONRequestBody) (*gen.Transaction, error) {
	count, err := s.repo.CountTransactions(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateTransaction: %w", err)
	}
	if count >= maxTransactionsPerTeam {
		return nil, ErrTooManyTransactions
	}

	if body.ContributionId != nil && body.PenaltyAssignmentId != nil {
		return nil, ErrTransactionLinksMultipleTargets
	}

	if body.ContributionId != nil {
		if body.Type != gen.Income {
			return nil, ErrContributionRequiresIncome
		}
		ok, err := s.repo.ContributionBelongsToTeam(ctx, *body.ContributionId, teamID)
		if err != nil {
			return nil, fmt.Errorf("finances.Service.CreateTransaction: %w", err)
		}
		if !ok {
			return nil, ErrContributionNotInTeam
		}
	}

	if body.PenaltyAssignmentId != nil {
		if body.Type != gen.Income {
			return nil, ErrPenaltyAssignmentRequiresIncome
		}
		ok, err := s.repo.PenaltyAssignmentBelongsToTeam(ctx, *body.PenaltyAssignmentId, teamID)
		if err != nil {
			return nil, fmt.Errorf("finances.Service.CreateTransaction: %w", err)
		}
		if !ok {
			return nil, ErrPenaltyAssignmentNotInTeam
		}
	}

	date := time.Now()
	if body.Date != nil {
		date = body.Date.Time
	}
	t, err := s.repo.CreateTransaction(ctx, teamID, string(body.Type), body.Title, body.Amount, date, body.Category, body.ContributionId, body.PenaltyAssignmentId)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateTransaction: %w", err)
	}
	result := toGenTransaction(*t)
	return &result, nil
}

// UpdateTransaction applies a patch to a transaction that belongs to teamID.
func (s *Service) UpdateTransaction(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateTransactionJSONRequestBody) (*gen.Transaction, error) {
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
	if body.Date != nil {
		d := body.Date.Time
		patch.Date = &d
	}
	t, err := s.repo.UpdateTransaction(ctx, id, teamID, patch)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.UpdateTransaction: %w", err)
	}
	result := toGenTransaction(*t)
	return &result, nil
}

// DeleteTransaction deletes a transaction that belongs to teamID.
func (s *Service) DeleteTransaction(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.DeleteTransaction(ctx, id, teamID); err != nil {
		return fmt.Errorf("finances.Service.DeleteTransaction: %w", err)
	}
	return nil
}

// ─── Penalties ────────────────────────────────────────────────────────────────

// CreatePenalty creates a new penalty definition.
func (s *Service) CreatePenalty(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	count, err := s.repo.CountPenalties(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreatePenalty: %w", err)
	}
	if count >= maxPenaltiesPerTeam {
		return nil, ErrTooManyPenalties
	}

	p, err := s.repo.CreatePenalty(ctx, teamID, body.Label, body.Amount)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreatePenalty: %w", err)
	}
	result := toGenPenalty(*p)
	return &result, nil
}

// UpdatePenalty applies a patch to a penalty definition that belongs to teamID.
func (s *Service) UpdatePenalty(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdatePenaltyJSONRequestBody) (*gen.Penalty, error) {
	patch := PenaltyPatch{Label: body.Label, Amount: body.Amount}
	p, err := s.repo.UpdatePenalty(ctx, id, teamID, patch)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.UpdatePenalty: %w", err)
	}
	result := toGenPenalty(*p)
	return &result, nil
}

// DeletePenalty deletes a penalty definition that belongs to teamID.
func (s *Service) DeletePenalty(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.DeletePenalty(ctx, id, teamID); err != nil {
		return fmt.Errorf("finances.Service.DeletePenalty: %w", err)
	}
	return nil
}

// ─── Assignments ──────────────────────────────────────────────────────────────

// CreateAssignment creates a penalty assignment after validating cross-team ownership.
func (s *Service) CreateAssignment(ctx context.Context, teamID uuid.UUID, body *gen.CreatePenaltyAssignmentJSONRequestBody) (*gen.PenaltyAssignment, error) {
	penaltyID := body.PenaltyId
	userID := body.UserId

	ok, err := s.repo.PenaltyBelongsToTeam(ctx, penaltyID, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateAssignment: %w", err)
	}
	if !ok {
		return nil, ErrPenaltyNotInTeam
	}

	isMember, err := s.repo.UserIsMemberOfTeam(ctx, userID, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateAssignment: %w", err)
	}
	if !isMember {
		return nil, ErrUserNotInTeam
	}

	count, err := s.repo.CountAssignments(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateAssignment: %w", err)
	}
	if count >= maxAssignmentsPerTeam {
		return nil, ErrTooManyAssignments
	}

	date := time.Now()
	if body.Date != nil {
		date = body.Date.Time
	}
	a, err := s.repo.CreateAssignment(ctx, teamID, userID, penaltyID, date, body.Note)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The membership check above passed, but the atomic WHERE
			// EXISTS re-check inside the INSERT itself failed -- the user
			// was removed from the team in the narrow window between the
			// two. Surface the same error the earlier check would have.
			return nil, ErrUserNotInTeam
		}
		return nil, fmt.Errorf("finances.Service.CreateAssignment: %w", err)
	}
	// Reload the single row with joined member/penalty data.
	full, err := s.repo.GetAssignmentByID(ctx, a.ID, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The row we just created is already gone -- a concurrent
			// DeletePenalty cascaded it away in the narrow window between
			// the insert and this reload. This is materially different
			// from a transient reload failure below: returning the
			// un-joined fallback here would be a 200 OK for an assignment
			// that no longer exists in the database, with blank
			// label/amount/member fields. Propagate ErrNoRows so the
			// handler's existing "not found" mapping applies instead.
			return nil, pgx.ErrNoRows
		}
		// The write already succeeded; any other reload failure (e.g. a
		// deadline hit right after the insert) must not fail the request,
		// but silently returning the un-joined fallback (penalty
		// label/amount/member name/photo all omitted) with no trace was a
		// real observability gap -- an operator would never know it happened.
		s.logger.Warn("finances: failed to reload assignment after create, returning partial result",
			slog.String("assignmentId", a.ID.String()), slog.String("error", err.Error()))
		result := toGenAssignment(*a)
		return &result, nil
	}
	result := toGenAssignment(*full)
	return &result, nil
}

// DeleteAssignment deletes a penalty assignment that belongs to teamID.
func (s *Service) DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.DeleteAssignment(ctx, id, teamID); err != nil {
		return fmt.Errorf("finances.Service.DeleteAssignment: %w", err)
	}
	return nil
}

// ─── Contributions ────────────────────────────────────────────────────────────

// CreateContributions creates a membership fee for one or more members in a
// single call -- one Contribution row per id in body.UserIds, all sharing
// the given name/amount/dueDate.
func (s *Service) CreateContributions(ctx context.Context, teamID uuid.UUID, body *gen.CreateContributionsJSONRequestBody) ([]gen.Contribution, error) {
	count, err := s.repo.CountContributions(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateContributions: %w", err)
	}
	if count+len(body.UserIds) > maxContributionsPerTeam {
		return nil, ErrTooManyContributions
	}

	var dueDate *time.Time
	if body.DueDate != nil {
		d := body.DueDate.Time
		dueDate = &d
	}

	rows, err := s.repo.CreateContributions(ctx, teamID, body.Name, body.Amount, dueDate, body.UserIds)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.CreateContributions: %w", err)
	}
	result := make([]gen.Contribution, 0, len(rows))
	for _, c := range rows {
		result = append(result, toGenContribution(c))
	}
	return result, nil
}

// UpdateContribution applies a patch to a contribution that belongs to teamID.
func (s *Service) UpdateContribution(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateContributionJSONRequestBody) (*gen.Contribution, error) {
	patch := ContributionPatch{Name: body.Name, Amount: body.Amount}
	if body.DueDate != nil {
		d := body.DueDate.Time
		patch.DueDate = &d
	}
	c, err := s.repo.UpdateContribution(ctx, id, teamID, patch)
	if err != nil {
		return nil, fmt.Errorf("finances.Service.UpdateContribution: %w", err)
	}
	result := toGenContribution(*c)
	return &result, nil
}

// DeleteContribution deletes a contribution that belongs to teamID. Any
// transaction linked to it is unlinked, not deleted (see
// Repository.DeleteContribution).
func (s *Service) DeleteContribution(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.DeleteContribution(ctx, id, teamID); err != nil {
		return fmt.Errorf("finances.Service.DeleteContribution: %w", err)
	}
	return nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toGenTransaction(t TransactionRow) gen.Transaction {
	return gen.Transaction{
		Id:                  t.ID,
		TeamId:              t.TeamID,
		Type:                gen.TransactionType(t.Type),
		Title:               t.Title,
		Amount:              t.Amount,
		Date:                openapi_types.Date{Time: t.Date},
		Category:            t.Category,
		ContributionId:      t.ContributionID,
		PenaltyAssignmentId: t.PenaltyAssignmentID,
	}
}

func toGenPenalty(p PenaltyRow) gen.Penalty {
	return gen.Penalty{
		Id:     p.ID,
		TeamId: p.TeamID,
		Label:  p.Label,
		Amount: p.Amount,
	}
}

// assignmentPaid derives a penalty assignment's paid flag from its paid
// amount vs. its snapshotted amount -- never stored, see
// PenaltyAssignmentRow's doc comment. A nil amount (the column is nullable,
// though every real insert path always populates it -- see
// PenaltyAssignmentRow.PenaltyID's doc comment on the sibling column) can
// never be satisfied, so such a row is treated as not paid rather than
// trivially "paid" by any amount >= a missing threshold.
func assignmentPaid(paidAmount int64, amount *int64) bool {
	return amount != nil && paidAmount >= *amount
}

func toGenAssignment(a PenaltyAssignmentRow) gen.PenaltyAssignment {
	return gen.PenaltyAssignment{
		Id:                a.ID,
		TeamId:            a.TeamID,
		UserId:            a.UserID,
		PenaltyId:         a.PenaltyID,
		Paid:              assignmentPaid(a.PaidAmount, a.PenaltyAmount),
		PaidAmount:        a.PaidAmount,
		Date:              openapi_types.Date{Time: a.Date},
		Note:              a.Note,
		Label:             a.PenaltyLabel,
		Amount:            a.PenaltyAmount,
		MemberName:        a.MemberName,
		MemberAvatarColor: a.MemberAvatarColor,
		HasPhoto:          a.HasPhoto,
	}
}

// contributionStatus derives a contribution's status from its paid amount
// vs. its full amount -- never stored, see ContributionRow's doc comment.
func contributionStatus(paidAmount, amount int64) gen.ContributionStatus {
	switch {
	case paidAmount <= 0:
		return gen.Open
	case paidAmount >= amount:
		return gen.Paid
	default:
		return gen.Partial
	}
}

func toGenContribution(c ContributionRow) gen.Contribution {
	var dueDate *openapi_types.Date
	if c.DueDate != nil {
		dueDate = &openapi_types.Date{Time: *c.DueDate}
	}
	return gen.Contribution{
		Id:                c.ID,
		TeamId:            c.TeamID,
		UserId:            c.UserID,
		Name:              c.Name,
		Amount:            c.Amount,
		DueDate:           dueDate,
		PaidAmount:        c.PaidAmount,
		Status:            contributionStatus(c.PaidAmount, c.Amount),
		MemberName:        c.MemberName,
		MemberAvatarColor: c.MemberAvatarColor,
		HasPhoto:          c.HasPhoto,
	}
}
