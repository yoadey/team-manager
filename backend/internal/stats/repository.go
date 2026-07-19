package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/attendance"
)

// pgxIface is satisfied by both *pgxpool.Pool and pgx.Tx, letting Repository
// run its queries either directly against the pool or inside a transaction
// (see WithReadTx). Mirrors the identical pattern in finances/repository.go.
type pgxIface interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository handles stats-related DB queries.
type Repository struct {
	// pool is only set on the top-level Repository returned by NewRepository;
	// it is nil on a tx-scoped Repository created by WithReadTx (which has no
	// need to start a nested transaction).
	pool *pgxpool.Pool
	db   pgxIface
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, db: pool}
}

// OverviewReader is the subset of read operations GetOverview needs. WithReadTx
// hands its callback this narrower view (rather than *Repository) so a caller
// can substitute a mock in unit tests without a live transaction.
type OverviewReader interface {
	MemberStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]MemberStatRow, error)
	EventStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]EventStatRow, error)
}

// WithReadTx runs fn with a Repository view backed by a single read-only,
// repeatable-read transaction, so MemberStats and EventStats observe one
// consistent snapshot instead of possibly drifting under concurrent writes
// (e.g. an event created/cancelled or attendance recorded between the two
// queries) -- unlike finances.GetOverview, which already gets this via its
// own WithReadTx.
func (r *Repository) WithReadTx(ctx context.Context, fn func(OverviewReader) error) error {
	if r.pool == nil {
		// Already running inside a transaction (nested call) — reuse it.
		return fn(r)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("stats.Repository.WithReadTx: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(&Repository{db: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// MemberStats returns attendance aggregations for all team members in the date range.
func (r *Repository) MemberStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]MemberStatRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Roster-driven and effective-status based: every current member is counted
	// once per active event in range, with opt_out/absence defaulting applied to
	// members who never explicitly responded -- identical to the event summary
	// (internal/events, GetAttendanceSummary), so a member's quote here matches
	// what the event detail shows. Explicit-only counting used to diverge from
	// the summary (e.g. an opt_out training with no responses showed 0% here but
	// "attending" there).
	rows, err := r.db.Query(ctx, `
		SELECT
			user_id,
			name,
			avatar_color,
			has_photo,
			COUNT(*) FILTER (WHERE eff = 'yes')                 AS yes_count,
			COUNT(*) FILTER (WHERE eff IN ('yes','no','maybe')) AS counted
		FROM (
			SELECT
				u.id           AS user_id,
				u.name         AS name,
				u.avatar_color AS avatar_color,
				(u.photo_object_key IS NOT NULL OR u.photo_data IS NOT NULL) AS has_photo,
				CASE WHEN e.id IS NULL THEN 'pending' ELSE `+attendance.EffectiveStatusExpr+` END AS eff
			FROM memberships m
			JOIN users u ON u.id = m.user_id
			LEFT JOIN events e ON e.team_id = m.team_id
				AND e.date BETWEEN $2 AND $3
				AND e.status = 'active'
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = u.id
			WHERE m.team_id = $1
		) sub
		GROUP BY user_id, name, avatar_color, has_photo
		ORDER BY yes_count DESC, name
	`, teamID, from, to)
	if err != nil {
		return nil, fmt.Errorf("stats.Repository.MemberStats: %w", err)
	}
	defer rows.Close()

	var out []MemberStatRow
	for rows.Next() {
		var s MemberStatRow
		if err := rows.Scan(&s.UserID, &s.Name, &s.AvatarColor, &s.HasPhoto, &s.Yes, &s.Counted); err != nil {
			return nil, fmt.Errorf("stats.Repository.MemberStats scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// EventStats returns per-event attendance counts for the team in the date range.
func (r *Repository) EventStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]EventStatRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Roster-driven and effective-status based, matching MemberStats and the
	// event summary: each active event is scored across its current members
	// (JOIN memberships), so a departed member's retained attendance row no
	// longer inflates the count -- the previous query counted any attendance row
	// including ex-members', which could not reconcile with the member-level
	// aggregation that filtered to current members.
	rows, err := r.db.Query(ctx, `
		SELECT
			event_id,
			title,
			type,
			date,
			COUNT(*) FILTER (WHERE eff = 'yes')                 AS yes_count,
			COUNT(*) FILTER (WHERE eff IN ('yes','no','maybe')) AS counted
		FROM (
			SELECT
				e.id         AS event_id,
				e.title      AS title,
				e.type       AS type,
				e.date::text AS date,
				`+attendance.EffectiveStatusExpr+` AS eff
			FROM events e
			JOIN memberships m ON m.team_id = e.team_id
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
			WHERE e.team_id = $1
			  AND e.date BETWEEN $2 AND $3
			  AND e.status = 'active'
		) sub
		GROUP BY event_id, title, type, date
		ORDER BY date
	`, teamID, from, to)
	if err != nil {
		return nil, fmt.Errorf("stats.Repository.EventStats: %w", err)
	}
	defer rows.Close()

	var out []EventStatRow
	for rows.Next() {
		var s EventStatRow
		if err := rows.Scan(&s.EventID, &s.Title, &s.Type, &s.Date, &s.Yes, &s.Counted); err != nil {
			return nil, fmt.Errorf("stats.Repository.EventStats scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SingleMemberStats returns attendance aggregations for one member in the date range.
func (r *Repository) SingleMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to string) (*MemberStatRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Same roster-driven, effective-status logic as MemberStats, scoped to one
	// member. Joining from memberships (rather than the previous EXISTS guard)
	// both provides the `m` alias the shared expression needs and returns no row
	// -- hence pgx.ErrNoRows -- when the user is not a member of the team.
	s := &MemberStatRow{}
	err := r.db.QueryRow(ctx, `
		SELECT
			user_id,
			name,
			avatar_color,
			has_photo,
			COUNT(*) FILTER (WHERE eff = 'yes')                 AS yes_count,
			COUNT(*) FILTER (WHERE eff IN ('yes','no','maybe')) AS counted
		FROM (
			SELECT
				u.id           AS user_id,
				u.name         AS name,
				u.avatar_color AS avatar_color,
				(u.photo_object_key IS NOT NULL OR u.photo_data IS NOT NULL) AS has_photo,
				CASE WHEN e.id IS NULL THEN 'pending' ELSE `+attendance.EffectiveStatusExpr+` END AS eff
			FROM memberships m
			JOIN users u ON u.id = m.user_id
			LEFT JOIN events e ON e.team_id = m.team_id
				AND e.date BETWEEN $3 AND $4
				AND e.status = 'active'
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = u.id
			WHERE m.team_id = $1 AND m.user_id = $2
		) sub
		GROUP BY user_id, name, avatar_color, has_photo
	`, teamID, userID, from, to).Scan(&s.UserID, &s.Name, &s.AvatarColor, &s.HasPhoto, &s.Yes, &s.Counted)
	if err != nil {
		return nil, fmt.Errorf("stats.Repository.SingleMemberStats: %w", err)
	}
	return s, nil
}
