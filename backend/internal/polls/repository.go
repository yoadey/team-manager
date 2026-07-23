package polls

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

// ErrOptionNotInPoll is returned when a vote references an optionID that
// does not belong to the poll being voted on.
var ErrOptionNotInPoll = errors.New("option does not belong to poll")

// pgxIface is satisfied by both *pgxpool.Pool and pgx.Tx, letting Repository
// run its queries either directly against the pool or inside a transaction
// (see WithReadTx).
type pgxIface interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Repository handles all polls-related DB operations.
type Repository struct {
	// pool is only set on the top-level Repository returned by NewRepository;
	// it is nil on a tx-scoped Repository created by WithReadTx (which has no
	// need to start a nested transaction), and is also used directly by
	// Create/ReplaceVotes, which need real multi-statement transactions rather
	// than the narrower pgxIface.
	pool *pgxpool.Pool
	db   pgxIface
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, db: pool}
}

// PollListReader is the subset of read operations ListByTeam needs. WithReadTx
// hands its callback this narrower view (rather than *Repository) so a caller
// can substitute a mock in unit tests without a live transaction.
type PollListReader interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*PollRow, error)
	ListOptionsByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*PollOptionRow, error)
	ListVotesByPollIDs(ctx context.Context, pollIDs []uuid.UUID) (map[uuid.UUID][]*PollVoteRow, error)
}

// WithReadTx runs fn with a Repository view backed by a single read-only,
// repeatable-read transaction, so all reads inside fn observe one consistent
// snapshot instead of drifting under concurrent writes (see
// Service.ListByTeam, which issues three independent queries -- polls, their
// options, and their votes -- that a concurrent Delete could otherwise
// interleave with, producing a "ghost" poll with empty options/votes for one
// that no longer exists).
func (r *Repository) WithReadTx(ctx context.Context, fn func(PollListReader) error) error {
	if r.pool == nil {
		// Already running inside a transaction (nested call) -- reuse it.
		return fn(r)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("polls.Repository.WithReadTx: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(&Repository{db: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	rows, err := r.db.Query(
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
	err := r.db.QueryRow(
		ctx,
		`SELECT id, team_id, creator_id, question, multiple, anonymous, created_at FROM polls WHERE id = $1 AND team_id = $2`,
		id, teamID,
	).Scan(&p.Id, &p.TeamId, &p.CreatorId, &p.Question, &p.Multiple, &p.Anonymous, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("polls.Repository.FindByID: %w", err)
	}
	return p, nil
}

// CountByTeam returns the number of polls the team has, used to enforce
// maxPollsPerTeam before an insert.
func (r *Repository) CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM polls WHERE team_id = $1`, teamID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("polls.Repository.CountByTeam: %w", err)
	}
	return count, nil
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
	tag, err := r.db.Exec(ctx, `DELETE FROM polls WHERE id = $1 AND team_id = $2`, id, teamID)
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
	rows, err := r.db.Query(
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
	rows, err := r.db.Query(
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
	rows, err := r.db.Query(
		ctx,
		`SELECT pv.poll_id, pv.option_id, pv.user_id,
		        u.name, u.avatar_color, (u.photo_object_key IS NOT NULL)
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
	rows, err := r.db.Query(
		ctx,
		`SELECT pv.poll_id, pv.option_id, pv.user_id,
		        u.name, u.avatar_color, (u.photo_object_key IS NOT NULL)
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
		// The membership EXISTS guard re-checks that userID currently
		// belongs to the poll's team -- polls/vote is self-service (see
		// authz.go), so RequireMembership only checks membership once at
		// the start of the request. Without this, a membership removal
		// racing this call could still commit a vote for a non-member, and
		// for a non-anonymous poll that ex-member's name/avatar/photo would
		// then be displayed alongside their vote to every remaining team
		// member indefinitely (assemblePoll has no other filter on voters).
		tag, err := tx.Exec(
			ctx,
			`INSERT INTO poll_votes (poll_id, option_id, user_id)
			 SELECT $1, $2, $3
			 WHERE EXISTS (SELECT 1 FROM poll_options WHERE id = $2 AND poll_id = $1)
			   AND EXISTS (
			     SELECT 1 FROM memberships m JOIN polls p ON p.team_id = m.team_id
			     WHERE p.id = $1 AND m.user_id = $3
			   )
			 ON CONFLICT DO NOTHING`,
			pollID, optID, userID,
		)
		if err != nil {
			return fmt.Errorf("polls.Repository.ReplaceVotes insert: %w", err)
		}
		// Rows were just cleared for (pollID, userID) above, so a duplicate-key
		// conflict is impossible here — zero rows affected means one of the
		// two WHERE EXISTS guards rejected the insert. See
		// diagnoseReplaceVotesRejection for what that can mean.
		if tag.RowsAffected() == 0 {
			return diagnoseReplaceVotesRejection(ctx, tx, pollID, userID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes: commit: %w", err)
	}
	return nil
}

// diagnoseReplaceVotesRejection determines why a poll_votes INSERT's WHERE
// EXISTS guards rejected a row (RowsAffected == 0), distinguishing three
// cases that otherwise look identical from the caller's side:
//   - the poll was deleted concurrently (DeletePoll cascades poll_options/
//     poll_votes away, so the option guard finds nothing) -> pgx.ErrNoRows,
//     the same "not found" a real GetByID miss would report.
//   - userID is not (or is no longer) a member of the poll's team (a
//     membership removal racing this self-service write) -> pgx.ErrNoRows
//     too, matching RequireMembership's own "not found, not forbidden"
//     convention for a non-member so as not to confirm a team's existence
//     to someone with no relationship to it.
//   - neither of the above: the option genuinely doesn't belong to this
//     poll -> ErrOptionNotInPoll (422).
func diagnoseReplaceVotesRejection(ctx context.Context, tx pgx.Tx, pollID, userID uuid.UUID) error {
	var pollExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM polls WHERE id = $1)`, pollID).Scan(&pollExists); err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes poll exists check: %w", err)
	}
	if !pollExists {
		return pgx.ErrNoRows
	}
	var isMember bool
	if err := tx.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM memberships m JOIN polls p ON p.team_id = m.team_id WHERE p.id = $1 AND m.user_id = $2)`,
		pollID, userID,
	).Scan(&isMember); err != nil {
		return fmt.Errorf("polls.Repository.ReplaceVotes membership check: %w", err)
	}
	if !isMember {
		return pgx.ErrNoRows
	}
	return ErrOptionNotInPoll
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
