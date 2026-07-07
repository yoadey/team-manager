package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrSoleSettingsAdmin is returned by EraseUser when the caller is the only
// living settings:write holder of at least one team they belong to --
// erasing them would leave that team permanently unable to manage its own
// roles/members/settings, since every one of those mutations itself
// requires settings:write held by an authenticatable account.
var ErrSoleSettingsAdmin = errors.New("cannot erase the account: you are the only settings administrator of a team")

// SoleSettingsAdminError wraps ErrSoleSettingsAdmin with the specific team
// IDs that blocked the erasure, so callers (e.g. the audit log) can record
// which team(s) need a second settings admin before the user can self-erase,
// without support having to re-run the underlying query by hand.
type SoleSettingsAdminError struct {
	TeamIDs []string
}

func (e *SoleSettingsAdminError) Error() string { return ErrSoleSettingsAdmin.Error() }
func (e *SoleSettingsAdminError) Unwrap() error { return ErrSoleSettingsAdmin }

// Repository handles all auth-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectUserFields = `
	id, name, email, phone, avatar_color,
	COALESCE(photo_data, ''::bytea) AS photo_data,
	COALESCE(photo_mime, '') AS photo_mime,
	birthday, address,
	COALESCE(password_hash, '') AS password_hash,
	created_at
`

// scanUser scans a row into a UserRow. The row must select the columns in
// the order defined by selectUserFields.
func scanUser(row interface {
	Scan(dest ...any) error
},
) (*UserRow, error) {
	u := &UserRow{}
	err := row.Scan(
		&u.Id, &u.Name, &u.Email, &u.Phone, &u.AvatarColor,
		&u.PhotoData, &u.PhotoMime,
		&u.Birthday, &u.Address, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auth.scanUser: %w", err)
	}
	return u, nil
}

// FindUserByEmail looks up a user by email address.
func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*UserRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM users WHERE email = $1 AND deleted_at IS NULL`, selectUserFields)
	row := r.pool.QueryRow(ctx, q, email)
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.FindUserByEmail: %w", err)
	}
	return u, nil
}

// FindUserByID looks up a user by primary key.
func (r *Repository) FindUserByID(ctx context.Context, id string) (*UserRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1 AND deleted_at IS NULL`, selectUserFields)
	row := r.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.FindUserByID: %w", err)
	}
	return u, nil
}

// CreateSession inserts a new session row and returns it.
func (r *Repository) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*SessionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		INSERT INTO sessions (user_id, token_hash, provider, expires_at)
		VALUES ($1, $2, 'password', $3)
		RETURNING id, user_id, token_hash, provider, expires_at, created_at
	`
	s := &SessionRow{}
	err := r.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).Scan(
		&s.Id, &s.UserId, &s.TokenHash, &s.Provider, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.CreateSession: %w", err)
	}
	return s, nil
}

// FindSession returns the session matching tokenHash that has not yet expired.
func (r *Repository) FindSession(ctx context.Context, tokenHash string) (*SessionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT id, user_id, token_hash, provider, expires_at, created_at
		FROM sessions
		WHERE token_hash = $1 AND expires_at > now()
	`
	s := &SessionRow{}
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(
		&s.Id, &s.UserId, &s.TokenHash, &s.Provider, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.FindSession: %w", err)
	}
	return s, nil
}

// DeleteSession removes the session identified by tokenHash.
func (r *Repository) DeleteSession(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("auth.Repository.DeleteSession: %w", err)
	}
	return nil
}

// EraseUser performs GDPR Art. 17 erasure by anonymization in a single
// transaction: it overwrites the user's personal data in place, strips
// free-text PII from their comments, attendance reasons and absence reasons,
// and deletes their sessions. Shared records (memberships, attendance, finance)
// keep their foreign key but no longer resolve to an identifiable person, so
// team statistics and legally retained accounting data stay intact.
func (r *Repository) EraseUser(ctx context.Context, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth.Repository.EraseUser: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var soleAdminTeamIDs []string
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(ARRAY_AGG(DISTINCT m.team_id), '{}')
		FROM memberships m
		JOIN membership_roles mr ON mr.membership_id = m.id
		JOIN roles r ON r.id = mr.role_id
		WHERE m.user_id = $1 AND r.permissions->>'settings' = 'write'
		AND NOT EXISTS (
			SELECT 1
			FROM memberships m2
			JOIN membership_roles mr2 ON mr2.membership_id = m2.id
			JOIN roles r2 ON r2.id = mr2.role_id
			JOIN users u2 ON u2.id = m2.user_id
			WHERE m2.team_id = m.team_id
			  AND m2.user_id != m.user_id
			  AND r2.permissions->>'settings' = 'write'
			  AND u2.deleted_at IS NULL
		)
	`, userID).Scan(&soleAdminTeamIDs)
	if err != nil {
		return fmt.Errorf("auth.Repository.EraseUser: check settings admin: %w", err)
	}
	if len(soleAdminTeamIDs) > 0 {
		return &SoleSettingsAdminError{TeamIDs: soleAdminTeamIDs}
	}

	const anonName = "Gelöschtes Mitglied"
	steps := []struct {
		sql  string
		args []any
	}{
		{`UPDATE users SET
			name = $2, email = 'deleted+' || id::text || '@invalid',
			phone = NULL, birthday = NULL, address = NULL,
			photo_data = NULL, photo_mime = NULL, password_hash = NULL,
			deleted_at = now()
		  WHERE id = $1 AND deleted_at IS NULL`, []any{userID, anonName}},
		{`UPDATE event_comments SET text = '' WHERE user_id = $1`, []any{userID}},
		{`UPDATE attendance SET reason = NULL WHERE user_id = $1`, []any{userID}},
		{`UPDATE absences SET reason = NULL WHERE user_id = $1`, []any{userID}},
		{`DELETE FROM sessions WHERE user_id = $1`, []any{userID}},
	}
	for _, s := range steps {
		if _, err := tx.Exec(ctx, s.sql, s.args...); err != nil {
			return fmt.Errorf("auth.Repository.EraseUser: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth.Repository.EraseUser: commit: %w", err)
	}
	return nil
}

// UpdateUserPhoto stores raw photo bytes and MIME type for the given user.
func (r *Repository) UpdateUserPhoto(ctx context.Context, userID string, data []byte, mime string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`UPDATE users SET photo_data = $2, photo_mime = $3 WHERE id = $1`,
		userID, data, mime,
	)
	if err != nil {
		return fmt.Errorf("auth.Repository.UpdateUserPhoto: %w", err)
	}
	return nil
}

// ensure uuid is used (uuid.UUID fields in UserRow/SessionRow).
var _ = uuid.UUID{}
