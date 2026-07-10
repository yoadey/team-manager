package absences

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgCheckViolation is the Postgres SQLSTATE for a violated CHECK constraint.
const pgCheckViolation = "23514"

// ErrInvalidDateRange is returned when a partial update would leave an
// absence's to_date before its from_date (violates the absences_date_range
// CHECK constraint). CreateAbsence/UpdateAbsence request bodies are validated
// in the handler when both dates are present in the same request, but a
// partial UpdateAbsence (only one of from/to) can only be caught here, since
// the merge happens inside the UPDATE statement itself.
var ErrInvalidDateRange = errors.New("to date must not be before from date")

// ErrSpanTooLong is returned when a partial update would leave an absence
// spanning more than maxAbsenceSpanDays (violates the
// absences_span_within_limit CHECK constraint, migration 00016). Same
// partial-PATCH gap as ErrInvalidDateRange: the handler only checks the span
// when both from/to are present in the same request.
var ErrSpanTooLong = errors.New("absence span must not exceed 3 years")

// ErrNotMember is returned by Create when userID is not (or is no longer) a
// member of teamID. RequireMembership checks this once at the start of the
// request, but absences is self-service with no module-level write gate
// (see authz.go's routeModule), so nothing re-checks it here -- without this,
// a membership removal racing this request (e.g. an admin's concurrent
// RemoveMember) could still leave an orphaned absence row (with a private
// `reason` text) attached to a team the user no longer belongs to, since
// events.SetAttendance/SetNomination already guard the identical race for
// their own self-service writes.
var ErrNotMember = errors.New("user is not a member of this team")

// absencesSpanConstraint is the CHECK constraint name added by migration
// 00016, used to distinguish an over-long span from an inverted date range
// (both surface as the same SQLSTATE 23514).
const absencesSpanConstraint = "absences_span_within_limit"

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
	(u.photo_data IS NOT NULL AND length(u.photo_data) > 0) AS has_photo,
	r.name AS role_name, r.color AS role_color
`

const absenceJoins = `
	FROM absences a
` + absenceRoleJoins

// absenceRoleJoins enriches a set of absence rows (aliased as a) with the
// author and a single non-system role. Shared by the list queries (which alias
// a keyset-bounded subquery as a) and findByID.
const absenceRoleJoins = `
	JOIN users u ON u.id = a.user_id
	LEFT JOIN memberships m ON m.user_id = a.user_id AND m.team_id = a.team_id
	LEFT JOIN membership_roles mr ON mr.membership_id = m.id
	LEFT JOIN roles r ON r.id = mr.role_id AND r.system = false
`

func scanAbsence(row interface{ Scan(dest ...any) error }) (*AbsenceRow, error) {
	ab := &AbsenceRow{}
	err := row.Scan(
		&ab.Id, &ab.UserId, &ab.TeamId, &ab.FromDate, &ab.ToDate, &ab.Reason, &ab.CreatedAt,
		&ab.MemberName, &ab.MemberAvatarColor, &ab.HasPhoto,
		&ab.RoleName, &ab.RoleColor,
	)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return ab, nil
}

// ListCursor is the keyset position for absence pagination
// (ORDER BY from_date DESC, id DESC).
//
// Known, accepted limitation: from_date is mutable (self-service Update lets
// a user change their own absence's start date). If a row's from_date
// changes to fall on the other side of an in-progress pagination's cursor
// while a caller is mid-page, that row can be skipped or, less likely,
// duplicated across pages -- the same tradeoff any keyset pagination scheme
// accepts when sorting by an editable column. The window is self-healing
// (a fresh list call is always fully correct) and low-impact (an admin
// viewing the team's absence list, not a security or data-integrity issue),
// so this is deliberately not being architected around.
type ListCursor struct {
	FromDate time.Time `json:"f"`
	ID       uuid.UUID `json:"i"`
}

// scanAbsenceRows drains rows into AbsenceRow slices.
func scanAbsenceRows(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}, where string,
) ([]*AbsenceRow, error) {
	var result []*AbsenceRow
	for rows.Next() {
		ab, err := scanAbsence(rows)
		if err != nil {
			return nil, fmt.Errorf("absences.Repository.%s scan: %w", where, err)
		}
		result = append(result, ab)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("absences.Repository.%s rows: %w", where, err)
	}
	return result, nil
}

// ListByTeam returns up to limit absences for a team (enriched with user and
// role info), newest first, starting after cur (nil = first page). The inner
// DISTINCT ON dedups the role join; the outer query applies the keyset order.
func (r *Repository) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*AbsenceRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	args := []any{teamID, limit}
	predicate := ""
	if cur != nil {
		predicate = "AND (from_date, id) < ($3, $4)"
		args = append(args, cur.FromDate, cur.ID)
	}
	// Bound the keyset scan to the page on the absences table (PK, no join
	// fan-out) first, then enrich only those rows and re-apply the page order.
	q := fmt.Sprintf(`
		SELECT * FROM (
			SELECT DISTINCT ON (a.id) %s
			FROM (
				SELECT * FROM absences
				WHERE team_id = $1 %s
				ORDER BY from_date DESC, id DESC
				LIMIT $2
			) a
			%s
			ORDER BY a.id
		) sub
		ORDER BY from_date DESC, id DESC`, selectAbsenceFields, predicate, absenceRoleJoins)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.ListByTeam: %w", err)
	}
	defer rows.Close()
	return scanAbsenceRows(rows, "ListByTeam")
}

// ListByUser returns up to limit absences for a specific user in a team,
// newest first, starting after cur (nil = first page).
func (r *Repository) ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit int, cur *ListCursor) ([]*AbsenceRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	args := []any{teamID, userID, limit}
	predicate := ""
	if cur != nil {
		predicate = "AND (from_date, id) < ($4, $5)"
		args = append(args, cur.FromDate, cur.ID)
	}
	q := fmt.Sprintf(`
		SELECT * FROM (
			SELECT DISTINCT ON (a.id) %s
			FROM (
				SELECT * FROM absences
				WHERE team_id = $1 AND user_id = $2 %s
				ORDER BY from_date DESC, id DESC
				LIMIT $3
			) a
			%s
			ORDER BY a.id
		) sub
		ORDER BY from_date DESC, id DESC`, selectAbsenceFields, predicate, absenceRoleJoins)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("absences.Repository.ListByUser: %w", err)
	}
	defer rows.Close()
	return scanAbsenceRows(rows, "ListByUser")
}

// Create inserts a new absence and returns the enriched row. Returns
// ErrNotMember if userID is not a member of teamID (see ErrNotMember's doc
// comment for why this re-check is needed despite RequireMembership already
// having run once at the start of the request).
func (r *Repository) Create(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*AbsenceRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var id uuid.UUID
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO absences (user_id, team_id, from_date, to_date, reason)
		SELECT $1, $2, $3, $4, $5
		WHERE EXISTS (SELECT 1 FROM memberships WHERE team_id = $2 AND user_id = $1)
		RETURNING id`,
		userID, teamID, fromDate, toDate, reason,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotMember
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation {
			if pgErr.ConstraintName == absencesSpanConstraint {
				return nil, ErrSpanTooLong
			}
			return nil, ErrInvalidDateRange
		}
		return nil, fmt.Errorf("absences.Repository.Create: %w", err)
	}
	return r.findByID(ctx, id)
}

// Update modifies an absence that belongs to teamID and userID (self-service:
// a member may only update their own absence entries) and returns the
// enriched row. Returns pgx.ErrNoRows if no matching absence exists.
func (r *Repository) Update(ctx context.Context, id, teamID, userID uuid.UUID, fromDate, toDate, reason *string) (*AbsenceRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE absences SET
			from_date = COALESCE($4::date, from_date),
			to_date   = COALESCE($5::date, to_date),
			reason    = COALESCE($6, reason)
		 WHERE id = $1 AND team_id = $2 AND user_id = $3`,
		id, teamID, userID, fromDate, toDate, reason,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation {
			if pgErr.ConstraintName == absencesSpanConstraint {
				return nil, ErrSpanTooLong
			}
			return nil, ErrInvalidDateRange
		}
		return nil, fmt.Errorf("absences.Repository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.findByID(ctx, id)
}

// Delete removes an absence by ID that belongs to teamID and userID
// (self-service: a member may only delete their own absence entries).
// Returns pgx.ErrNoRows if no matching absence exists.
func (r *Repository) Delete(ctx context.Context, id, teamID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM absences WHERE id = $1 AND team_id = $2 AND user_id = $3`, id, teamID, userID)
	if err != nil {
		return fmt.Errorf("absences.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
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
