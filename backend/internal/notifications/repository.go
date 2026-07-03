package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all notifications-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListByTeamAndUser returns notifications for the last 62 days, marking unread
// based on the user's last seen timestamp.
func (r *Repository) ListByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) ([]*NotificationRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			n.id, n.team_id, n.type, n.actor_id, n.status, n.title,
			n.event_id, n.event_title, n.event_date, n.note, n.created_at,
			u.name AS actor_name, u.avatar_color AS actor_color,
			COALESCE(u.photo_data IS NOT NULL, false) AS has_photo,
			CASE WHEN ns.seen_at IS NULL OR n.created_at > ns.seen_at THEN true ELSE false END AS unread
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		LEFT JOIN notif_seen ns ON ns.team_id = n.team_id AND ns.user_id = $2
		WHERE n.team_id = $1
		  AND n.created_at >= now() - interval '62 days'
		ORDER BY n.created_at DESC
		LIMIT $3
	`
	// Bounded by a 62-day window already; this LIMIT is a defensive backstop
	// against pathologically high notification volume within that window.
	const maxNotificationRows = 2000
	rows, err := r.pool.Query(ctx, q, teamID, userID, maxNotificationRows)
	if err != nil {
		return nil, fmt.Errorf("notifications.Repository.ListByTeamAndUser: %w", err)
	}
	defer rows.Close()

	var result []*NotificationRow
	for rows.Next() {
		nr := &NotificationRow{}
		err := rows.Scan(
			&nr.Id, &nr.TeamId, &nr.Type, &nr.ActorId, &nr.Status, &nr.Title,
			&nr.EventId, &nr.EventTitle, &nr.EventDate, &nr.Note, &nr.CreatedAt,
			&nr.ActorName, &nr.ActorColor, &nr.HasPhoto,
			&nr.Unread,
		)
		if err != nil {
			return nil, fmt.Errorf("notifications.Repository.ListByTeamAndUser scan: %w", err)
		}
		result = append(result, nr)
	}
	return result, rows.Err()
}

// MarkSeen upserts the seen timestamp for a user in a team.
func (r *Repository) MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO notif_seen (team_id, user_id, seen_at) VALUES ($1, $2, now())
		 ON CONFLICT (team_id, user_id) DO UPDATE SET seen_at = now()`,
		teamID, userID,
	)
	if err != nil {
		return fmt.Errorf("notifications.Repository.MarkSeen: %w", err)
	}
	return nil
}
