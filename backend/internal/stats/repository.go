package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles stats-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// MemberStats returns attendance aggregations for all team members in the date range.
func (r *Repository) MemberStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]MemberStatRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT
			u.id,
			u.name,
			u.avatar_color,
			(u.photo_data IS NOT NULL) AS has_photo,
			COUNT(*) FILTER (WHERE a.status = 'yes') AS yes_count,
			COUNT(*) FILTER (WHERE a.status IN ('yes','no','maybe')) AS counted
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN events e ON e.team_id = m.team_id
			AND e.date BETWEEN $2 AND $3
			AND e.status = 'active'
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = u.id
		WHERE m.team_id = $1
		GROUP BY u.id, u.name, u.avatar_color, (u.photo_data IS NOT NULL)
		ORDER BY yes_count DESC, u.name
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
	rows, err := r.pool.Query(ctx, `
		SELECT
			e.id,
			e.title,
			e.type,
			e.date::text,
			COUNT(*) FILTER (WHERE a.status = 'yes') AS yes_count,
			COUNT(*) FILTER (WHERE a.status IN ('yes','no','maybe')) AS counted
		FROM events e
		LEFT JOIN attendance a ON a.event_id = e.id
		WHERE e.team_id = $1
		  AND e.date BETWEEN $2 AND $3
		  AND e.status = 'active'
		GROUP BY e.id, e.title, e.type, e.date
		ORDER BY e.date
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
	s := &MemberStatRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT
			u.id,
			u.name,
			u.avatar_color,
			(u.photo_data IS NOT NULL) AS has_photo,
			COUNT(*) FILTER (WHERE a.status = 'yes') AS yes_count,
			COUNT(*) FILTER (WHERE a.status IN ('yes','no','maybe')) AS counted
		FROM users u
		LEFT JOIN events e ON e.team_id = $1
			AND e.date BETWEEN $3 AND $4
			AND e.status = 'active'
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = u.id
		WHERE u.id = $2
		  AND EXISTS (SELECT 1 FROM memberships m WHERE m.team_id = $1 AND m.user_id = u.id)
		GROUP BY u.id, u.name, u.avatar_color, (u.photo_data IS NOT NULL)
	`, teamID, userID, from, to).Scan(&s.UserID, &s.Name, &s.AvatarColor, &s.HasPhoto, &s.Yes, &s.Counted)
	if err != nil {
		return nil, fmt.Errorf("stats.Repository.SingleMemberStats: %w", err)
	}
	return s, nil
}
