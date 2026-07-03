package polls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrOptionNotInPoll is returned when a vote references an optionID that
// does not belong to the poll being voted on.
var ErrOptionNotInPoll = errors.New("option does not belong to poll")

// Repository handles all polls-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListByTeam returns all polls for a team.
// ListCursor is the keyset position for poll pagination
// (ORDER BY created_at DESC, id DESC).
type ListCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

func (r *Repository) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*PollRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	args := []any{teamID, limit}
	predicate := ""
	if cur != nil {
		predicate = "AND (created_at, id) < ($3, $4)"
		args = append(args, cur.CreatedAt, cur.ID)
	}
	rows, err := r.pool.Query(
		ctx,
		fmt.Sprintf(`SELECT id, team_id, creator_id, question, multiple, anonymous, created_at
		 FROM polls WHERE team_id = $1 %s ORDER BY created_at DESC, id DESC LIMIT $2`, predicate),
		args...,
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

// FindByID returns a single poll scoped to teamID.
func (r *Repository) FindByID(ctx context.Context, id, teamID uuid.UUID) (*PollRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p := &PollRow{}
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, team_id, creator_id, question, multiple, anonymous, created_at FROM polls WHERE id = $1 AND team_id = $2`,
		id, teamID,
	).Scan(&p.Id, &p.TeamId, &p.CreatorId, &p.Question, &p.Multiple, &p.Anonymous, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.FindByID: %w", err)
	}
	return p, nil
}

// Create inserts a new poll and its options. Returns the new poll ID.
// The poll row and all option rows are inserted within a single transaction
// so that a failure partway through the options loop cannot leave an
// orphaned/incomplete poll behind.
func (r *Repository) Create(ctx context.Context, teamID, creatorID uuid.UUID, question string, multiple, anonymous bool, options []string) (uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("polls.Repository.Create: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var pollID uuid.UUID
	err = tx.QueryRow(
		ctx,
		`INSERT INTO polls (team_id, creator_id, question, multiple, anonymous) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		teamID, creatorID, question, multiple, anonymous,
	).Scan(&pollID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("polls.Repository.Create: %w", err)
	}
	for i, opt := range options {
		_, err := tx.Exec(
			ctx,
			`INSERT INTO poll_options (poll_id, text, sort_order) VALUES ($1, $2, $3)`,
			pollID, opt, i,
		)
		if err != nil {
			return uuid.Nil, fmt.Errorf("polls.Repository.Create options: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("polls.Repository.Create: commit: %w", err)
	}
	return pollID, nil
}

// Delete removes a poll and its options/votes by ID, scoped to teamID.
func (r *Repository) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM polls WHERE id = $1 AND team_id = $2`, id, teamID)
	if err != nil {
		return fmt.Errorf("polls.Repository.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
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

// ListOptionsByPollIDs returns all options for the given polls in a single
// query, keyed by poll ID, so a page of N polls costs one round trip instead
// of N.
func (r *Repository) ListOptionsByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*PollOptionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result := make(map[uuid.UUID][]*PollOptionRow, len(pollIDs))
	if len(pollIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, poll_id, text, sort_order FROM poll_options WHERE poll_id = ANY($1) ORDER BY poll_id, sort_order`,
		pollIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.ListOptionsByPollIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		o := &PollOptionRow{}
		if err := rows.Scan(&o.Id, &o.PollId, &o.Text, &o.SortOrder); err != nil {
			return nil, fmt.Errorf("polls.Repository.ListOptionsByPollIDs scan: %w", err)
		}
		result[o.PollId] = append(result[o.PollId], o)
	}
	return result, rows.Err()
}

// ListVotesByPollIDs returns all votes for the given polls (with user info)
// in a single query, keyed by poll ID.
func (r *Repository) ListVotesByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*PollVoteRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result := make(map[uuid.UUID][]*PollVoteRow, len(pollIDs))
	if len(pollIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(
		ctx,
		`SELECT pv.poll_id, pv.option_id, pv.user_id,
		        u.name, u.avatar_color, (u.photo_data IS NOT NULL)
		 FROM poll_votes pv
		 JOIN users u ON u.id = pv.user_id
		 WHERE pv.poll_id = ANY($1)`,
		pollIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.ListVotesByPollIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		v := &PollVoteRow{}
		if err := rows.Scan(&v.PollId, &v.OptionId, &v.UserId, &v.UserName, &v.UserColor, &v.HasPhoto); err != nil {
			return nil, fmt.Errorf("polls.Repository.ListVotesByPollIDs scan: %w", err)
		}
		result[v.PollId] = append(result[v.PollId], v)
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
		        u.name, u.avatar_color, (u.photo_data IS NOT NULL)
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
		if err := rows.Scan(&v.PollId, &v.OptionId, &v.UserId, &v.UserName, &v.UserColor, &v.HasPhoto); err != nil {
			return nil, fmt.Errorf("polls.Repository.ListVotes scan: %w", err)
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// ReplaceVotes replaces all votes for a user on a poll within a single
// transaction, so a concurrent call for the same user+poll can never observe
// (or leave behind) a state where the old votes are deleted but the new ones
// aren't inserted yet. Callers must have already rejected optionIDs with
// len > 1 for single-choice (multiple=false) polls; this method trusts that
// invariant instead of silently truncating input to the first option.
//
// A transaction-scoped advisory lock keyed on (pollID, userID) serializes
// concurrent calls for the same user+poll. Without it, two concurrent votes
// for different options on a single-choice poll can each run their DELETE
// before either INSERT commits (Read Committed isolation), so neither
// observes the other's row and both succeed — leaving the user with two
// votes on a poll that's supposed to allow only one. The lock is released
// automatically on commit/rollback.
func (r *Repository) ReplaceVotes(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID, multiple bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1 || $2, 0))`,
		pollID.String(), userID.String(),
	); err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes lock: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM poll_votes WHERE poll_id = $1 AND user_id = $2`,
		pollID, userID,
	); err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes delete: %w", err)
	}

	// Dedupe: the "duplicate-key conflict is impossible" invariant the loop
	// below relies on only holds if optionIDs itself has no duplicates. The
	// OpenAPI schema doesn't declare uniqueItems, so a client can legally
	// submit e.g. ["A","A","B"] — without this, the second insert of "A"
	// hits ON CONFLICT DO NOTHING (RowsAffected=0), which the loop
	// misreads as "A doesn't belong to this poll" and aborts the whole
	// vote, silently dropping the legitimate "B" selection too.
	optionIDs = dedupeUUIDs(optionIDs)

	for _, optID := range optionIDs {
		tag, err := tx.Exec(
			ctx,
			`INSERT INTO poll_votes (poll_id, option_id, user_id)
			 SELECT $1, $2, $3
			 WHERE EXISTS (SELECT 1 FROM poll_options WHERE id = $2 AND poll_id = $1)
			 ON CONFLICT DO NOTHING`,
			pollID, optID, userID,
		)
		if err != nil {
			return fmt.Errorf("polls.Repository.ReplaceVotes insert: %w", err)
		}
		// Rows were just cleared for (pollID, userID) above, so a duplicate-key
		// conflict is impossible here — zero rows affected can only mean the
		// WHERE EXISTS guard rejected an optionID that doesn't belong to pollID.
		if tag.RowsAffected() == 0 {
			return ErrOptionNotInPoll
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes: commit: %w", err)
	}
	return nil
}

// dedupeUUIDs returns ids with duplicates removed, preserving first-seen order.
func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
