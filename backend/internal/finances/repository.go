package finances

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all finance-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── Transactions ─────────────────────────────────────────────────────────────

// maxOverviewRows caps the display lists returned by the finance overview
// (transactions, assignments, contributions) so a team with many years of
// history can't force an unbounded response. Aggregates (income/expense sums,
// open-contribution count) are computed separately via dedicated queries that
// scan the full table, so capping the display list never skews the totals.
const maxOverviewRows = 1000

// ListTransactions returns up to maxOverviewRows most recent transactions for
// the team, ordered by date desc.
func (r *Repository) ListTransactions(ctx context.Context, teamID uuid.UUID) ([]TransactionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, type, title, amount, date, category, created_at
		FROM transactions
		WHERE team_id = $1
		ORDER BY date DESC, created_at DESC
		LIMIT $2
	`, teamID, maxOverviewRows)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.ListTransactions: %w", err)
	}
	defer rows.Close()

	var out []TransactionRow
	for rows.Next() {
		var t TransactionRow
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Type, &t.Title, &t.Amount, &t.Date, &t.Category, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("finances.Repository.ListTransactions scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SumTransactions returns the total income and expense across ALL of the
// team's transactions (not just the capped display list returned by
// ListTransactions), so the finance overview's income/expense/balance figures
// stay accurate regardless of how much history exists.
func (r *Repository) SumTransactions(ctx context.Context, teamID uuid.UUID) (income, expense float64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0),
			COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)
		FROM transactions
		WHERE team_id = $1
	`, teamID).Scan(&income, &expense)
	if err != nil {
		return 0, 0, fmt.Errorf("finances.Repository.SumTransactions: %w", err)
	}
	return income, expense, nil
}

// CreateTransaction inserts a new transaction.
func (r *Repository) CreateTransaction(ctx context.Context, teamID uuid.UUID, txType, title string, amount float64, date time.Time, category *string) (*TransactionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	t := &TransactionRow{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO transactions (team_id, type, title, amount, date, category)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, team_id, type, title, amount, date, category, created_at
	`, teamID, txType, title, amount, date, category).Scan(
		&t.ID, &t.TeamID, &t.Type, &t.Title, &t.Amount, &t.Date, &t.Category, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.CreateTransaction: %w", err)
	}
	return t, nil
}

// UpdateTransaction applies a partial update to a transaction that belongs to teamID.
func (r *Repository) UpdateTransaction(ctx context.Context, id, teamID uuid.UUID, patch TransactionPatch) (*TransactionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setClauses, args, n := []string{}, []any{}, 1

	if patch.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", n))
		args = append(args, *patch.Type)
		n++
	}
	if patch.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", n))
		args = append(args, *patch.Title)
		n++
	}
	if patch.Amount != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", n))
		args = append(args, *patch.Amount)
		n++
	}
	if patch.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", n))
		args = append(args, *patch.Category)
		n++
	}

	if len(setClauses) == 0 {
		return r.getTransactionByID(ctx, id)
	}

	args = append(args, id, teamID)
	t := &TransactionRow{}
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE transactions SET %s WHERE id = $%d AND team_id = $%d
		RETURNING id, team_id, type, title, amount, date, category, created_at
	`, strings.Join(setClauses, ", "), n, n+1), args...).Scan(
		&t.ID, &t.TeamID, &t.Type, &t.Title, &t.Amount, &t.Date, &t.Category, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.UpdateTransaction: %w", err)
	}
	return t, nil
}

// DeleteTransaction deletes a transaction that belongs to teamID.
func (r *Repository) DeleteTransaction(ctx context.Context, id, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return fmt.Errorf("finances.Repository.DeleteTransaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) getTransactionByID(ctx context.Context, id uuid.UUID) (*TransactionRow, error) {
	t := &TransactionRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, team_id, type, title, amount, date, category, created_at
		FROM transactions WHERE id = $1
	`, id).Scan(&t.ID, &t.TeamID, &t.Type, &t.Title, &t.Amount, &t.Date, &t.Category, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ─── Penalties ────────────────────────────────────────────────────────────────

// ListPenalties returns all penalty definitions for the team.
func (r *Repository) ListPenalties(ctx context.Context, teamID uuid.UUID) ([]PenaltyRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, label, amount FROM penalties WHERE team_id = $1 ORDER BY label
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.ListPenalties: %w", err)
	}
	defer rows.Close()

	var out []PenaltyRow
	for rows.Next() {
		var p PenaltyRow
		if err := rows.Scan(&p.ID, &p.TeamID, &p.Label, &p.Amount); err != nil {
			return nil, fmt.Errorf("finances.Repository.ListPenalties scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreatePenalty inserts a new penalty definition.
func (r *Repository) CreatePenalty(ctx context.Context, teamID uuid.UUID, label string, amount float64) (*PenaltyRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p := &PenaltyRow{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO penalties (team_id, label, amount)
		VALUES ($1, $2, $3)
		RETURNING id, team_id, label, amount
	`, teamID, label, amount).Scan(&p.ID, &p.TeamID, &p.Label, &p.Amount)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.CreatePenalty: %w", err)
	}
	return p, nil
}

// UpdatePenalty applies a partial update to a penalty definition that belongs to teamID.
func (r *Repository) UpdatePenalty(ctx context.Context, id, teamID uuid.UUID, patch PenaltyPatch) (*PenaltyRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setClauses, args, n := []string{}, []any{}, 1

	if patch.Label != nil {
		setClauses = append(setClauses, fmt.Sprintf("label = $%d", n))
		args = append(args, *patch.Label)
		n++
	}
	if patch.Amount != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", n))
		args = append(args, *patch.Amount)
		n++
	}

	if len(setClauses) == 0 {
		p := &PenaltyRow{}
		err := r.pool.QueryRow(ctx, `SELECT id, team_id, label, amount FROM penalties WHERE id = $1 AND team_id = $2`, id, teamID).
			Scan(&p.ID, &p.TeamID, &p.Label, &p.Amount)
		return p, err
	}

	args = append(args, id, teamID)
	p := &PenaltyRow{}
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE penalties SET %s WHERE id = $%d AND team_id = $%d
		RETURNING id, team_id, label, amount
	`, strings.Join(setClauses, ", "), n, n+1), args...).Scan(&p.ID, &p.TeamID, &p.Label, &p.Amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.UpdatePenalty: %w", err)
	}
	return p, nil
}

// DeletePenalty deletes a penalty definition that belongs to teamID (cascades to assignments).
func (r *Repository) DeletePenalty(ctx context.Context, id, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM penalties WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return fmt.Errorf("finances.Repository.DeletePenalty: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ─── Penalty Assignments ──────────────────────────────────────────────────────

// ListAssignments returns up to maxOverviewRows most recent penalty assignments
// for the team with member/penalty info joined.
func (r *Repository) ListAssignments(ctx context.Context, teamID uuid.UUID) ([]PenaltyAssignmentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT pa.id, pa.team_id, pa.user_id, pa.penalty_id, pa.paid, pa.date,
		       p.label, p.amount,
		       u.name, u.avatar_color,
		       (u.photo_data IS NOT NULL) AS has_photo
		FROM penalty_assignments pa
		JOIN penalties p ON p.id = pa.penalty_id
		JOIN users u ON u.id = pa.user_id
		WHERE pa.team_id = $1
		ORDER BY pa.date DESC
		LIMIT $2
	`, teamID, maxOverviewRows)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.ListAssignments: %w", err)
	}
	defer rows.Close()

	var out []PenaltyAssignmentRow
	for rows.Next() {
		var a PenaltyAssignmentRow
		if err := rows.Scan(
			&a.ID, &a.TeamID, &a.UserID, &a.PenaltyID, &a.Paid, &a.Date,
			&a.PenaltyLabel, &a.PenaltyAmount,
			&a.MemberName, &a.MemberAvatarColor, &a.HasPhoto,
		); err != nil {
			return nil, fmt.Errorf("finances.Repository.ListAssignments scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAssignmentByID returns a single penalty assignment with joined member/penalty
// data, scoped to teamID.
func (r *Repository) GetAssignmentByID(ctx context.Context, id, teamID uuid.UUID) (*PenaltyAssignmentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	a := &PenaltyAssignmentRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT pa.id, pa.team_id, pa.user_id, pa.penalty_id, pa.paid, pa.date,
		       p.label, p.amount,
		       u.name, u.avatar_color,
		       (u.photo_data IS NOT NULL) AS has_photo
		FROM penalty_assignments pa
		JOIN penalties p ON p.id = pa.penalty_id
		JOIN users u ON u.id = pa.user_id
		WHERE pa.id = $1 AND pa.team_id = $2
	`, id, teamID).Scan(
		&a.ID, &a.TeamID, &a.UserID, &a.PenaltyID, &a.Paid, &a.Date,
		&a.PenaltyLabel, &a.PenaltyAmount,
		&a.MemberName, &a.MemberAvatarColor, &a.HasPhoto,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.GetAssignmentByID: %w", err)
	}
	return a, nil
}

// CreateAssignment inserts a penalty assignment for a user.
func (r *Repository) CreateAssignment(ctx context.Context, teamID, userID, penaltyID uuid.UUID) (*PenaltyAssignmentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	a := &PenaltyAssignmentRow{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO penalty_assignments (team_id, user_id, penalty_id)
		VALUES ($1, $2, $3)
		RETURNING id, team_id, user_id, penalty_id, paid, date
	`, teamID, userID, penaltyID).Scan(&a.ID, &a.TeamID, &a.UserID, &a.PenaltyID, &a.Paid, &a.Date)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.CreateAssignment: %w", err)
	}
	return a, nil
}

// DeleteAssignment deletes a penalty assignment that belongs to teamID.
func (r *Repository) DeleteAssignment(ctx context.Context, id, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM penalty_assignments WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return fmt.Errorf("finances.Repository.DeleteAssignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ToggleAssignmentPaid flips the paid flag on a penalty assignment that belongs to teamID.
func (r *Repository) ToggleAssignmentPaid(ctx context.Context, id, teamID uuid.UUID) (*PenaltyAssignmentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	a := &PenaltyAssignmentRow{}
	err := r.pool.QueryRow(ctx, `
		UPDATE penalty_assignments SET paid = NOT paid WHERE id = $1 AND team_id = $2
		RETURNING id, team_id, user_id, penalty_id, paid, date
	`, id, teamID).Scan(&a.ID, &a.TeamID, &a.UserID, &a.PenaltyID, &a.Paid, &a.Date)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.ToggleAssignmentPaid: %w", err)
	}
	return a, nil
}

// ─── Contributions ────────────────────────────────────────────────────────────

// ListContributions returns up to maxOverviewRows most recent contributions
// for the team with member info joined.
func (r *Repository) ListContributions(ctx context.Context, teamID uuid.UUID) ([]ContributionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.team_id, c.user_id, c.month, c.label, c.amount, c.status,
		       u.name, u.avatar_color,
		       (u.photo_data IS NOT NULL) AS has_photo
		FROM contributions c
		JOIN users u ON u.id = c.user_id
		WHERE c.team_id = $1
		ORDER BY c.month DESC, u.name
		LIMIT $2
	`, teamID, maxOverviewRows)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.ListContributions: %w", err)
	}
	defer rows.Close()

	var out []ContributionRow
	for rows.Next() {
		var c ContributionRow
		if err := rows.Scan(
			&c.ID, &c.TeamID, &c.UserID, &c.Month, &c.Label, &c.Amount, &c.Status,
			&c.MemberName, &c.MemberAvatarColor, &c.HasPhoto,
		); err != nil {
			return nil, fmt.Errorf("finances.Repository.ListContributions scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateContribution applies a partial update to a contribution that belongs to teamID.
func (r *Repository) UpdateContribution(ctx context.Context, id, teamID uuid.UUID, patch ContributionPatch) (*ContributionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setClauses, args, n := []string{}, []any{}, 1

	if patch.Label != nil {
		setClauses = append(setClauses, fmt.Sprintf("label = $%d", n))
		args = append(args, *patch.Label)
		n++
	}
	if patch.Amount != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", n))
		args = append(args, *patch.Amount)
		n++
	}

	if len(setClauses) == 0 {
		return r.getContributionByID(ctx, id)
	}

	args = append(args, id, teamID)
	tag, err := r.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE contributions SET %s WHERE id = $%d AND team_id = $%d
	`, strings.Join(setClauses, ", "), n, n+1), args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.UpdateContribution: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.getContributionByID(ctx, id)
}

// ToggleContributionStatus flips between 'open' and 'paid' for a contribution that belongs to teamID.
func (r *Repository) ToggleContributionStatus(ctx context.Context, id, teamID uuid.UUID) (*ContributionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var currentStatus string
	err := r.pool.QueryRow(ctx, `SELECT status FROM contributions WHERE id = $1 AND team_id = $2`, id, teamID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.ToggleContributionStatus: %w", err)
	}

	newStatus := "paid"
	if currentStatus == "paid" {
		newStatus = "open"
	}

	_, err = r.pool.Exec(ctx, `UPDATE contributions SET status = $1 WHERE id = $2`, newStatus, id)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.ToggleContributionStatus update: %w", err)
	}
	return r.getContributionByID(ctx, id)
}

// CountOpenContributions returns the number of contributions with status
// 'open' for the team, independent of the capped display list returned by
// ListContributions.
func (r *Repository) CountOpenContributions(ctx context.Context, teamID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contributions WHERE team_id = $1 AND status = 'open'`,
		teamID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("finances.Repository.CountOpenContributions: %w", err)
	}
	return count, nil
}

func (r *Repository) getContributionByID(ctx context.Context, id uuid.UUID) (*ContributionRow, error) {
	c := &ContributionRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT c.id, c.team_id, c.user_id, c.month, c.label, c.amount, c.status,
		       u.name, u.avatar_color,
		       (u.photo_data IS NOT NULL) AS has_photo
		FROM contributions c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1
	`, id).Scan(
		&c.ID, &c.TeamID, &c.UserID, &c.Month, &c.Label, &c.Amount, &c.Status,
		&c.MemberName, &c.MemberAvatarColor, &c.HasPhoto,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// PenaltyBelongsToTeam returns true when the penalty exists and belongs to teamID.
func (r *Repository) PenaltyBelongsToTeam(ctx context.Context, penaltyID, teamID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM penalties WHERE id = $1 AND team_id = $2)`,
		penaltyID, teamID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("finances.Repository.PenaltyBelongsToTeam: %w", err)
	}
	return exists, nil
}

// UserIsMemberOfTeam returns true when userID is an active member of teamID.
func (r *Repository) UserIsMemberOfTeam(ctx context.Context, userID, teamID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM team_members WHERE user_id = $1 AND team_id = $2)`,
		userID, teamID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("finances.Repository.UserIsMemberOfTeam: %w", err)
	}
	return exists, nil
}

// ─── Aggregates ───────────────────────────────────────────────────────────────

// OpenPenaltyAggregate represents the summed unpaid penalties per user.
type OpenPenaltyAggregate struct {
	UserID      uuid.UUID
	Name        string
	AvatarColor string
	HasPhoto    bool
	TotalAmount float64
}

// ListOpenPenaltiesByUser returns unpaid penalty amounts aggregated per user for the team.
func (r *Repository) ListOpenPenaltiesByUser(ctx context.Context, teamID uuid.UUID) ([]OpenPenaltyAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT pa.user_id, u.name, u.avatar_color,
		       (u.photo_data IS NOT NULL) AS has_photo,
		       SUM(p.amount) AS total_amount
		FROM penalty_assignments pa
		JOIN penalties p ON p.id = pa.penalty_id
		JOIN users u ON u.id = pa.user_id
		WHERE pa.team_id = $1 AND pa.paid = false
		GROUP BY pa.user_id, u.name, u.avatar_color, (u.photo_data IS NOT NULL)
		ORDER BY total_amount DESC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.ListOpenPenaltiesByUser: %w", err)
	}
	defer rows.Close()

	var out []OpenPenaltyAggregate
	for rows.Next() {
		var a OpenPenaltyAggregate
		if err := rows.Scan(&a.UserID, &a.Name, &a.AvatarColor, &a.HasPhoto, &a.TotalAmount); err != nil {
			return nil, fmt.Errorf("finances.Repository.ListOpenPenaltiesByUser scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
