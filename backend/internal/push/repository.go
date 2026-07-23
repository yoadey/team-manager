package push

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all push_subscriptions DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Upsert inserts a subscription for userID, or updates its keys in place if
// the endpoint (a browser re-subscribing after a key rotation) already
// exists -- an endpoint is globally unique, never duplicated per user.
func (r *Repository) Upsert(ctx context.Context, userID uuid.UUID, sub Subscription) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth_key)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth_key = EXCLUDED.auth_key
	`, userID, sub.Endpoint, sub.P256dh, sub.AuthKey)
	if err != nil {
		return fmt.Errorf("push.Repository.Upsert: %w", err)
	}
	return nil
}

// Delete removes userID's subscription for the given endpoint. A request
// naming an endpoint that belongs to a different user (or doesn't exist)
// affects no rows -- deleting is scoped to (user_id, endpoint), never
// endpoint alone, so a user can never remove another user's subscription.
func (r *Repository) Delete(ctx context.Context, userID uuid.UUID, endpoint string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`,
		userID, endpoint,
	)
	if err != nil {
		return fmt.Errorf("push.Repository.Delete: %w", err)
	}
	return nil
}

// DeleteByID removes a subscription outright, regardless of owner -- used by
// the push delivery worker to prune a subscription the push service has
// reported gone (404/410), where the caller has no user context at all.
func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("push.Repository.DeleteByID: %w", err)
	}
	return nil
}

// ListForTeamExcludingUser returns every push subscription belonging to a
// current member of teamID, other than excludeUserID (the notification's
// actor -- a member typically doesn't want to be pushed their own action).
// Joining against memberships here, rather than requiring a separate
// "list team member user IDs" call, keeps team-membership scoping in a
// single query.
func (r *Repository) ListForTeamExcludingUser(ctx context.Context, teamID, excludeUserID uuid.UUID) ([]SubscriptionForUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT ps.id, ps.user_id, ps.endpoint, ps.p256dh, ps.auth_key
		FROM push_subscriptions ps
		JOIN memberships m ON m.user_id = ps.user_id AND m.team_id = $1
		WHERE ps.user_id != $2
	`, teamID, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("push.Repository.ListForTeamExcludingUser: %w", err)
	}
	defer rows.Close()

	var result []SubscriptionForUser
	for rows.Next() {
		var s SubscriptionForUser
		if err := rows.Scan(&s.Id, &s.UserId, &s.Subscription.Endpoint, &s.Subscription.P256dh, &s.Subscription.AuthKey); err != nil {
			return nil, fmt.Errorf("push.Repository.ListForTeamExcludingUser scan: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
