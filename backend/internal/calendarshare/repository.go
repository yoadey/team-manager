package calendarshare

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrTeamNotFound is returned by Grant when the target team doesn't exist.
var ErrTeamNotFound = errors.New("calendarshare: team not found")

// Repository handles all calendar_shares DB operations, plus the redacted
// event projection query calendar shares are read through.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Grant records a calendar-share grant from ownerTeamID to viewerTeamID,
// idempotently -- granting an already-active share is a no-op that returns
// the existing row's CreatedAt rather than erroring, since re-granting
// isn't a meaningful conflict for the caller.
func (r *Repository) Grant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (*ShareRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var viewerName string
	if err := r.pool.QueryRow(ctx, `SELECT name FROM teams WHERE id = $1`, viewerTeamID).Scan(&viewerName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, fmt.Errorf("calendarshare.Repository.Grant: load viewer team: %w", err)
	}

	row := &ShareRow{TeamId: viewerTeamID, TeamName: viewerName}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO calendar_shares (owner_team_id, viewer_team_id)
		VALUES ($1, $2)
		ON CONFLICT (owner_team_id, viewer_team_id) DO UPDATE
			SET created_at = calendar_shares.created_at
		RETURNING created_at
	`, ownerTeamID, viewerTeamID).Scan(&row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Repository.Grant: %w", err)
	}
	return row, nil
}

// Revoke deletes the grant (ownerTeamID -> viewerTeamID), if any. Idempotent:
// revoking a non-existent grant is a no-op, not an error.
func (r *Repository) Revoke(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		DELETE FROM calendar_shares WHERE owner_team_id = $1 AND viewer_team_id = $2
	`, ownerTeamID, viewerTeamID)
	if err != nil {
		return fmt.Errorf("calendarshare.Repository.Revoke: %w", err)
	}
	return nil
}

// ListGrantedByOwner returns every team ownerTeamID has granted calendar
// visibility to, newest grant first. Each row's TeamId/TeamName is the
// *viewer* team.
func (r *Repository) ListGrantedByOwner(ctx context.Context, ownerTeamID uuid.UUID) ([]ShareRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT cs.viewer_team_id, t.name, cs.created_at
		FROM calendar_shares cs
		JOIN teams t ON t.id = cs.viewer_team_id
		WHERE cs.owner_team_id = $1
		ORDER BY cs.created_at DESC
	`, ownerTeamID)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Repository.ListGrantedByOwner: %w", err)
	}
	defer rows.Close()
	return scanShareRows(rows, "ListGrantedByOwner")
}

// ListGrantedToViewer returns every team that has granted viewerTeamID
// calendar visibility, newest grant first. Each row's TeamId/TeamName is
// the *owner* team.
func (r *Repository) ListGrantedToViewer(ctx context.Context, viewerTeamID uuid.UUID) ([]ShareRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT cs.owner_team_id, t.name, cs.created_at
		FROM calendar_shares cs
		JOIN teams t ON t.id = cs.owner_team_id
		WHERE cs.viewer_team_id = $1
		ORDER BY cs.created_at DESC
	`, viewerTeamID)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Repository.ListGrantedToViewer: %w", err)
	}
	defer rows.Close()
	return scanShareRows(rows, "ListGrantedToViewer")
}

func scanShareRows(rows pgx.Rows, caller string) ([]ShareRow, error) {
	var out []ShareRow
	for rows.Next() {
		var s ShareRow
		if err := rows.Scan(&s.TeamId, &s.TeamName, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("calendarshare.Repository.%s: scan: %w", caller, err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("calendarshare.Repository.%s: rows: %w", caller, err)
	}
	return out, nil
}

// HasGrant reports whether ownerTeamID currently grants viewerTeamID
// calendar visibility.
func (r *Repository) HasGrant(ctx context.Context, ownerTeamID, viewerTeamID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM calendar_shares WHERE owner_team_id = $1 AND viewer_team_id = $2)
	`, ownerTeamID, viewerTeamID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("calendarshare.Repository.HasGrant: %w", err)
	}
	return exists, nil
}

// maxRedactedEvents caps how many of an owner team's events a single
// ListRedactedEvents call returns -- defensive backstop, mirroring
// calendarfeed.Service's maxFeedEvents, against pathologically long-lived
// teams with an unbounded event history when the caller passes no date range.
const maxRedactedEvents = 2000

// ListRedactedEvents returns ownerTeamID's non-cancelled events as the
// schedule-only projection (see RedactedEventRow's doc comment), optionally
// bounded to the inclusive [from, to] date range, earliest first.
func (r *Repository) ListRedactedEvents(ctx context.Context, ownerTeamID uuid.UUID, from, to *time.Time) ([]RedactedEventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	q := `
		SELECT id, type, title, date,
			COALESCE(TO_CHAR(start_time, 'HH24:MI'), '') AS start_time,
			COALESCE(TO_CHAR(end_time, 'HH24:MI'), '') AS end_time,
			location
		FROM events
		WHERE team_id = $1 AND status = 'active'
	`
	args := []any{ownerTeamID}
	if from != nil {
		args = append(args, *from)
		q += fmt.Sprintf(" AND date >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		q += fmt.Sprintf(" AND date <= $%d", len(args))
	}
	args = append(args, maxRedactedEvents)
	q += fmt.Sprintf(" ORDER BY date ASC LIMIT $%d", len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("calendarshare.Repository.ListRedactedEvents: %w", err)
	}
	defer rows.Close()

	var out []RedactedEventRow
	for rows.Next() {
		var e RedactedEventRow
		var startTime, endTime string
		if err := rows.Scan(&e.Id, &e.Type, &e.Title, &e.Date, &startTime, &endTime, &e.Location); err != nil {
			return nil, fmt.Errorf("calendarshare.Repository.ListRedactedEvents: scan: %w", err)
		}
		if startTime != "" {
			e.StartTime = &startTime
		}
		if endTime != "" {
			e.EndTime = &endTime
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("calendarshare.Repository.ListRedactedEvents: rows: %w", err)
	}
	return out, nil
}
