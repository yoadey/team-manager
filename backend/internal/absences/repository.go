package absences

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all absence-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectAbsenceFields = `
	a.id, a.user_id, a.team_id, a.from_date, a.to_date, a.reason, a.created_at,
	u.name AS member_name, u.avatar_color AS member_avatar_color,
	COALESCE(u.photo_data, ''::bytea) AS photo_data,
	r.name AS role_name, r.color AS role_color
`

const absenceJoins = `
	FROM absences a
	JOIN users u ON u.id = a.user_id
	LEFT JOIN memberships m ON m.user_id = a.user_id AND m.team_id = a.team_id
	LEFT JOIN membership_roles mr ON mr.membership_id = m.id
	LEFT JOIN roles r ON r.id = mr.role_id AND r.system = false
`

func scanAbsence(row interface{ Scan(dest ...any) error }) (*AbsenceRow, error) {
	ab := &AbsenceRow{}
	err := row.Scan(
		&ab.Id, &ab.UserId, &ab.TeamId, &ab.FromDate, &ab.ToDate, &ab.Reason, &ab.CreatedAt,
		&ab.MemberName, &ab.MemberAvatarColor, &ab.PhotoData,
		&ab.RoleName, &ab.RoleColor,
	)
	if err != nil {
		return nil, err
	}
	return ab, nil
}

// ListByTeam returns all absences for a team, enriched with user and role info.
func (r *Repository) ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*AbsenceRow, error) {
	q := fmt.Sprintf(`SELECT DISTINCT ON (a.id) %s %s WHERE a.team_id = $1 ORDER BY a.id, a.from_date DESC LIMIT $2 OFFSET $3`, selectAbsenceFields, absenceJoins)
	rows, err := r.pool.Query(ctx, q, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.ListByTeam: %w", err)
	}
	defer rows.Close()

	var result []*AbsenceRow
	for rows.Next() {
		ab, err := scanAbsence(rows)
		if err != nil {
			return nil, fmt.Errorf("absences.Repository.ListByTeam scan: %w", err)
		}
		result = append(result, ab)
	}
	return result, rows.Err()
}

// ListByUser returns absences for a specific user in a team.
func (r *Repository) ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit, offset int) ([]*AbsenceRow, error) {
	q := fmt.Sprintf(`SELECT DISTINCT ON (a.id) %s %s WHERE a.team_id = $1 AND a.user_id = $2 ORDER BY a.id, a.from_date DESC LIMIT $3 OFFSET $4`, selectAbsenceFields, absenceJoins)
	rows, err := r.pool.Query(ctx, q, teamID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.ListByUser: %w", err)
	}
	defer rows.Close()

	var result []*AbsenceRow
	for rows.Next() {
		ab, err := scanAbsence(rows)
		if err != nil {
			return nil, fmt.Errorf("absences.Repository.ListByUser scan: %w", err)
		}
		result = append(result, ab)
	}
	return result, rows.Err()
}

// Create inserts a new absence and returns the enriched row.
func (r *Repository) Create(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*AbsenceRow, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`INSERT INTO absences (user_id, team_id, from_date, to_date, reason) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		userID, teamID, fromDate, toDate, reason,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.Create: %w", err)
	}
	return r.findByID(ctx, id)
}

// Update modifies an absence and returns the enriched row.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, fromDate, toDate *string, reason *string) (*AbsenceRow, error) {
	_, err := r.pool.Exec(ctx,
		`UPDATE absences SET
			from_date = COALESCE($2::date, from_date),
			to_date   = COALESCE($3::date, to_date),
			reason    = COALESCE($4, reason)
		 WHERE id = $1`,
		id, fromDate, toDate, reason,
	)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.Update: %w", err)
	}
	return r.findByID(ctx, id)
}

// Delete removes an absence by ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM absences WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("absences.Repository.Delete: %w", err)
	}
	return nil
}

// findByID looks up a single absence with enrichment.
func (r *Repository) findByID(ctx context.Context, id uuid.UUID) (*AbsenceRow, error) {
	q := fmt.Sprintf(`SELECT DISTINCT ON (a.id) %s %s WHERE a.id = $1 ORDER BY a.id`, selectAbsenceFields, absenceJoins)
	row := r.pool.QueryRow(ctx, q, id)
	ab, err := scanAbsence(row)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.findByID: %w", err)
	}
	return ab, nil
}
