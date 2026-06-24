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

// ListTransactions returns all transactions for the team, ordered by date desc.
func (r *Repository) ListTransactions(ctx context.Context, teamID uuid.UUID) ([]TransactionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, type, title, amount, date, category, created_at
		FROM transactions
		WHERE team_id = $1
		ORDER BY date DESC, created_at DESC
	`, teamID)
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

// UpdateTransaction applies a partial update to a transaction.
func (r *Repository) UpdateTransaction(ctx context.Context, id uuid.UUID, patch TransactionPatch) (*TransactionRow, error) {
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

	args = append(args, id)
	t := &TransactionRow{}
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE transactions SET %s WHERE id = $%d
		RETURNING id, team_id, type, title, amount, date, category, created_at
	`, strings.Join(setClauses, ", "), n), args...).Scan(
		&t.ID, &t.TeamID, &t.Type, &t.Title, &t.Amount, &t.Date, &t.Category, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("finances.Repository.UpdateTransaction: %w", err)
	}
	return t, nil
}

// DeleteTransaction deletes a transaction by ID.
func (r *Repository) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id = $1`, id)
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

// UpdatePenalty applies a partial update to a penalty definition.
func (r *Repository) UpdatePenalty(ctx context.Context, id uuid.UUID, patch PenaltyPatch) (*PenaltyRow, error) {
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
		err := r.pool.QueryRow(ctx, `SELECT id, team_id, label, amount FROM penalties WHERE id = $1`, id).
			Scan(&p.ID, &p.TeamID, &p.Label, &p.Amount)
		return p, err
	}

	args = append(args, id)
	p := &PenaltyRow{}
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE penalties SET %s WHERE id = $%d
		RETURNING id, team_id, label, amount
	`, strings.Join(setClauses, ", "), n), args...).Scan(&p.ID, &p.TeamID, &p.Label, &p.Amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.UpdatePenalty: %w", err)
	}
	return p, nil
}

// DeletePenalty deletes a penalty definition (cascades to assignments).
func (r *Repository) DeletePenalty(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM penalties WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("finances.Repository.DeletePenalty: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ─── Penalty Assignments ──────────────────────────────────────────────────────

// ListAssignments returns all penalty assignments for the team with member/penalty info joined.
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
	`, teamID)
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

// DeleteAssignment deletes a penalty assignment by ID.
func (r *Repository) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM penalty_assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("finances.Repository.DeleteAssignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ToggleAssignmentPaid flips the paid flag on a penalty assignment and returns the updated row.
func (r *Repository) ToggleAssignmentPaid(ctx context.Context, id uuid.UUID) (*PenaltyAssignmentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	a := &PenaltyAssignmentRow{}
	err := r.pool.QueryRow(ctx, `
		UPDATE penalty_assignments SET paid = NOT paid WHERE id = $1
		RETURNING id, team_id, user_id, penalty_id, paid, date
	`, id).Scan(&a.ID, &a.TeamID, &a.UserID, &a.PenaltyID, &a.Paid, &a.Date)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.ToggleAssignmentPaid: %w", err)
	}
	return a, nil
}

// ─── Contributions ────────────────────────────────────────────────────────────

// ListContributions returns all contributions for the team with member info joined.
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
	`, teamID)
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

// UpdateContribution applies a partial update to a contribution.
func (r *Repository) UpdateContribution(ctx context.Context, id uuid.UUID, patch ContributionPatch) (*ContributionRow, error) {
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

	args = append(args, id)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE contributions SET %s WHERE id = $%d
	`, strings.Join(setClauses, ", "), n), args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("finances.Repository.UpdateContribution: %w", err)
	}
	return r.getContributionByID(ctx, id)
}

// ToggleContributionStatus flips between 'open' and 'paid'.
func (r *Repository) ToggleContributionStatus(ctx context.Context, id uuid.UUID) (*ContributionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var currentStatus string
	err := r.pool.QueryRow(ctx, `SELECT status FROM contributions WHERE id = $1`, id).Scan(&currentStatus)
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
