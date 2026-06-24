package polls

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all polls-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListByTeam returns all polls for a team.
func (r *Repository) ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*PollRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, team_id, creator_id, question, multiple, anonymous, created_at
		 FROM polls WHERE team_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		teamID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.ListByTeam: %w", err)
	}
	defer rows.Close()

	var result []*PollRow
	for rows.Next() {
		p := &PollRow{}
		if err := rows.Scan(&p.Id, &p.TeamId, &p.CreatorId, &p.Question, &p.Multiple, &p.Anonymous, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("polls.Repository.ListByTeam scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// FindByID returns a single poll.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*PollRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p := &PollRow{}
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, team_id, creator_id, question, multiple, anonymous, created_at FROM polls WHERE id = $1`,
		id,
	).Scan(&p.Id, &p.TeamId, &p.CreatorId, &p.Question, &p.Multiple, &p.Anonymous, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.FindByID: %w", err)
	}
	return p, nil
}

// Create inserts a new poll and its options. Returns the new poll ID.
func (r *Repository) Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var pollID uuid.UUID
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO polls (team_id, creator_id, question, multiple, anonymous) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		teamID, creatorID, question, multiple, anonymous,
	).Scan(&pollID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("polls.Repository.Create: %w", err)
	}
	for i, opt := range options {
		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO poll_options (poll_id, text, sort_order) VALUES ($1, $2, $3)`,
			pollID, opt, i,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("polls.Repository.Create options: %w", err)
		}
	}
	return pollID, nil
}

// Delete removes a poll and its options/votes by ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `DELETE FROM polls WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("polls.Repository.Delete: %w", err)
	}
	return nil
}

// ListOptions returns all options for a poll.
func (r *Repository) ListOptions(ctx context.Context, pollID uuid.UUID) ([]*PollOptionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, poll_id, text, sort_order FROM poll_options WHERE poll_id = $1 ORDER BY sort_order`,
		pollID,
	)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.ListOptions: %w", err)
	}
	defer rows.Close()

	var result []*PollOptionRow
	for rows.Next() {
		o := &PollOptionRow{}
		if err := rows.Scan(&o.Id, &o.PollId, &o.Text, &o.SortOrder); err != nil {
			return nil, fmt.Errorf("polls.Repository.ListOptions scan: %w", err)
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

// ListVotes returns all votes for a poll with user info.
func (r *Repository) ListVotes(ctx context.Context, pollID uuid.UUID) ([]*PollVoteRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(
		ctx,
		`SELECT pv.poll_id, pv.option_id, pv.user_id,
		        u.name, u.avatar_color, COALESCE(u.photo_data, ''::bytea)
		 FROM poll_votes pv
		 JOIN users u ON u.id = pv.user_id
		 WHERE pv.poll_id = $1`,
		pollID,
	)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.ListVotes: %w", err)
	}
	defer rows.Close()

	var result []*PollVoteRow
	for rows.Next() {
		v := &PollVoteRow{}
		if err := rows.Scan(&v.PollId, &v.OptionId, &v.UserId, &v.UserName, &v.UserColor, &v.PhotoData); err != nil {
			return nil, fmt.Errorf("polls.Repository.ListVotes scan: %w", err)
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// ReplaceVotes replaces all votes for a user on a poll.
// If not multiple, deletes all existing votes first.
func (r *Repository) ReplaceVotes(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Always delete existing votes for this user+poll before inserting.
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM poll_votes WHERE poll_id = $1 AND user_id = $2`,
		pollID, userID,
	)
	if err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes delete: %w", err)
	}

	for _, optID := range optionIDs {
		if !multiple && len(optionIDs) > 1 {
			// For single-choice polls only accept the first option.
			break
		}
		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO poll_votes (poll_id, option_id, user_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			pollID, optID, userID,
		)
		if err != nil {
			return fmt.Errorf("polls.Repository.ReplaceVotes insert: %w", err)
		}
	}
	return nil
}
