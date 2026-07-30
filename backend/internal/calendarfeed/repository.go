package calendarfeed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all calendar_feed_tokens DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// generateToken returns a 32-byte, hex-encoded random token -- stored in the
// clear (unlike session tokens), mirroring invites.code: a leaked feed link
// is meant to be trivially revocable and replaceable, not something the DB
// needs to protect against its own compromise.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// IssueToken revokes any existing active token for (userID, teamID) and
// inserts a fresh one, returning the new bare token. Wrapped in a
// transaction so a concurrent issue for the same pair can't leave two
// simultaneously-active rows (the partial unique index on
// (user_id, team_id) WHERE revoked_at IS NULL is the final backstop if it
// somehow did). Carries the previous active row's content selection
// (types/include_birthdays) forward to the new row, if one existed --
// rotating a link (a security action) must not silently reset a
// customized feed back to "everything on".
func (r *Repository) IssueToken(ctx context.Context, userID, teamID uuid.UUID) (token string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tok, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("calendarfeed.Repository.IssueToken: generate token: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("calendarfeed.Repository.IssueToken: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	types := defaultFeedTypes
	includeBirthdays := true
	err = tx.QueryRow(ctx, `
		SELECT types, include_birthdays FROM calendar_feed_tokens
		WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL
	`, userID, teamID).Scan(&types, &includeBirthdays)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("calendarfeed.Repository.IssueToken: read existing selection: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE calendar_feed_tokens SET revoked_at = now()
		WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL
	`, userID, teamID); err != nil {
		return "", fmt.Errorf("calendarfeed.Repository.IssueToken: revoke existing: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO calendar_feed_tokens (user_id, team_id, token, types, include_birthdays)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, teamID, tok, types, includeBirthdays); err != nil {
		return "", fmt.Errorf("calendarfeed.Repository.IssueToken: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("calendarfeed.Repository.IssueToken: commit: %w", err)
	}
	return tok, nil
}

// Revoke invalidates the active token, if any, for (userID, teamID).
func (r *Repository) Revoke(ctx context.Context, userID, teamID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		UPDATE calendar_feed_tokens SET revoked_at = now()
		WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL
	`, userID, teamID)
	if err != nil {
		return fmt.Errorf("calendarfeed.Repository.Revoke: %w", err)
	}
	return nil
}

// FindActiveByToken looks up a non-revoked token row by its bare token
// value. Returns pgx.ErrNoRows (unwrapped, so callers can errors.Is it) when
// no active row matches -- an unknown, already-revoked, or previously
// rotated-away token all look identical to the caller, which is the point:
// the feed handler must not distinguish "never existed" from "no longer
// valid".
func (r *Repository) FindActiveByToken(ctx context.Context, token string) (*TokenRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := &TokenRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, team_id, token, created_at, revoked_at, types, include_birthdays
		FROM calendar_feed_tokens
		WHERE token = $1 AND revoked_at IS NULL
	`, token).Scan(&row.Id, &row.UserId, &row.TeamId, &row.Token, &row.CreatedAt, &row.RevokedAt, &row.Types, &row.IncludeBirthdays)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("calendarfeed.Repository.FindActiveByToken: %w", err)
	}
	return row, nil
}

// GetSettings returns the active token's content selection for (userID,
// teamID). Returns pgx.ErrNoRows (unwrapped) if no active token exists --
// the caller decides whether that means "use the defaults" or "not found".
func (r *Repository) GetSettings(ctx context.Context, userID, teamID uuid.UUID) (types []string, includeBirthdays bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = r.pool.QueryRow(ctx, `
		SELECT types, include_birthdays FROM calendar_feed_tokens
		WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL
	`, userID, teamID).Scan(&types, &includeBirthdays)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, pgx.ErrNoRows
		}
		return nil, false, fmt.Errorf("calendarfeed.Repository.GetSettings: %w", err)
	}
	return types, includeBirthdays, nil
}

// UpdateSettings overwrites the active token's content selection for
// (userID, teamID), applying immediately to the existing subscription URL.
// Returns pgx.ErrNoRows (unwrapped) if no active token exists to update.
func (r *Repository) UpdateSettings(ctx context.Context, userID, teamID uuid.UUID, types []string, includeBirthdays bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE calendar_feed_tokens SET types = $3, include_birthdays = $4
		WHERE user_id = $1 AND team_id = $2 AND revoked_at IS NULL
	`, userID, teamID, types, includeBirthdays)
	if err != nil {
		return fmt.Errorf("calendarfeed.Repository.UpdateSettings: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
