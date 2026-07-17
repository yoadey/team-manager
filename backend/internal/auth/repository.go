package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	(photo_object_key IS NOT NULL) AS has_photo,
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
		&u.HasPhoto,
		&u.Birthday, &u.Address, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("auth.scanUser: %w", err)
	}
	return u, nil
}

// FindUserByEmail looks up a user by email address. The lookup key is
// normalized to lowercase to match members.Repository.UpdateMember, which
// normalizes before writing -- the DB's UNIQUE constraint on users.email is
// case-sensitive, so without matching normalization on both sides, a user
// who typed their email in a different case at signup than at login would
// fail to find their own account.
func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*UserRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM users WHERE email = $1 AND deleted_at IS NULL`, selectUserFields)
	row := r.pool.QueryRow(ctx, q, strings.ToLower(strings.TrimSpace(email)))
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.FindUserByEmail: %w", err)
	}
	return u, nil
}

// FindUserByID looks up a user by primary key. This is on the hot path --
// invoked on essentially every authenticated request via
// Service.ValidateToken/Handler.AuthMiddleware -- so it only selects a
// HasPhoto boolean rather than the object key; use FindUserPhotoKeyByID for
// the one path that actually needs it.
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

// FindUserPhotoKeyByID returns the object-store key for id's profile photo,
// or pgx.ErrNoRows if the user has no photo set (or does not exist / is
// soft-deleted).
func (r *Repository) FindUserPhotoKeyByID(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var key *string
	err := r.pool.QueryRow(ctx, `SELECT photo_object_key FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&key)
	if err != nil {
		return "", fmt.Errorf("auth.Repository.FindUserPhotoKeyByID: %w", err)
	}
	if key == nil || *key == "" {
		return "", pgx.ErrNoRows
	}
	return *key, nil
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

	// Lock every team the user belongs to, in a deterministic (team_id) order,
	// before running the sole-settings-admin check below. Without this, the
	// check races against members.SetRoles/RemoveMember and
	// roles.UpdateRole/DeleteRole -- each of which locks its team with the
	// same pg_advisory_xact_lock(hashtextextended(teamID, 0)) key before
	// mutating role assignments -- so under READ COMMITTED, a concurrent
	// self-erasure and a role change stripping another member's
	// settings:write could each see a stale "another admin still exists"
	// snapshot and both commit, leaving the team with zero settings:write
	// holders.
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(team_id::text, 0))
		FROM (SELECT DISTINCT team_id FROM memberships WHERE user_id = $1 ORDER BY team_id) t
	`, userID); err != nil {
		return fmt.Errorf("auth.Repository.EraseUser: advisory lock: %w", err)
	}

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
			photo_data = NULL, photo_mime = NULL, photo_object_key = NULL, password_hash = NULL,
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

// UpdateUserPhoto stores the object-store key for the given user's photo.
func (r *Repository) UpdateUserPhoto(ctx context.Context, userID, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(
		ctx,
		`UPDATE users SET photo_object_key = $2 WHERE id = $1`,
		userID, objectKey,
	)
	if err != nil {
		return fmt.Errorf("auth.Repository.UpdateUserPhoto: %w", err)
	}
	return nil
}

// ensure uuid is used (uuid.UUID fields in UserRow/SessionRow).
var _ = uuid.UUID{}
