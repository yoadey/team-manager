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
	(photo_object_key IS NOT NULL AND length(photo_object_key) > 0) AS has_photo,
	birthday, address,
	COALESCE(password_hash, '') AS password_hash,
	created_at, email_verified_at
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
		&u.Birthday, &u.Address, &u.PasswordHash, &u.CreatedAt, &u.EmailVerifiedAt,
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
// HasPhoto boolean rather than the photo's object key; the photo itself is
// served via a presigned object-store URL (see internal/storage), never
// streamed through this lookup.
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

// FindUserPhotoKeyByID returns the object store key for id's photo, or
// pgx.ErrNoRows if the user has no photo set (or does not exist / is
// soft-deleted).
func (r *Repository) FindUserPhotoKeyByID(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var key *string
	err := r.pool.QueryRow(ctx, `SELECT photo_object_key FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
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

// ErrEmailTaken is returned by CreateUnverifiedUser when a user already
// exists with the given email (the ON CONFLICT DO NOTHING branch).
var ErrEmailTaken = errors.New("auth: email already registered")

// CreateUnverifiedUser inserts a new, unverified user row (email_verified_at
// left NULL) with the given bcrypt password hash. name is a placeholder
// display name (the email's local part -- self-registration collects no
// separate name field); the user can change it later via their team member
// profile. Returns ErrEmailTaken if a user with this email already exists --
// the caller (Service.Register) uses that to distinguish the
// already-registered branches of its enumeration-safe response without a
// separate existence check racing the insert.
func (r *Repository) CreateUnverifiedUser(ctx context.Context, name, email, passwordHash string) (*UserRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING
		RETURNING %s
	`, selectUserFields)
	row := r.pool.QueryRow(ctx, q, name, strings.ToLower(strings.TrimSpace(email)), passwordHash)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("auth.Repository.CreateUnverifiedUser: %w", err)
	}
	return u, nil
}

// MarkEmailVerified sets email_verified_at to now() for userID.
func (r *Repository) MarkEmailVerified(ctx context.Context, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email_verified_at = now() WHERE id = $1 AND email_verified_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("auth.Repository.MarkEmailVerified: %w", err)
	}
	return nil
}

// CreateEmailVerificationToken inserts a new verification token row keyed by
// its SHA-256 hash (the raw token is never persisted).
func (r *Repository) CreateEmailVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("auth.Repository.CreateEmailVerificationToken: %w", err)
	}
	return nil
}

// FindEmailVerificationToken returns the token row matching tokenHash,
// provided it has not expired and has not already been consumed. Returns
// pgx.ErrNoRows otherwise (expired, consumed, or never existed -- the caller
// doesn't need to distinguish these, all three are simply "invalid token").
func (r *Repository) FindEmailVerificationToken(ctx context.Context, tokenHash string) (*EmailVerificationTokenRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	t := &EmailVerificationTokenRow{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1 AND expires_at > now() AND consumed_at IS NULL
	`, tokenHash).Scan(&t.Id, &t.UserId, &t.TokenHash, &t.ExpiresAt, &t.ConsumedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("auth.Repository.FindEmailVerificationToken: %w", err)
	}
	return t, nil
}

// ConsumeEmailVerificationToken marks the token identified by tokenHash as
// consumed, guarded by "WHERE consumed_at IS NULL" so a concurrent
// double-submit of the same token can only succeed once. Returns
// pgx.ErrNoRows if the token doesn't exist or was already consumed.
func (r *Repository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(ctx,
		`UPDATE email_verification_tokens SET consumed_at = now() WHERE token_hash = $1 AND consumed_at IS NULL`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("auth.Repository.ConsumeEmailVerificationToken: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
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
			photo_object_key = NULL,
			password_hash = NULL, deleted_at = now()
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

// UpdateUserPhoto stores the object store key for the given user's photo.
// UpdateUserPhoto returns pgx.ErrNoRows if userID has no active (non-erased)
// row -- without the deleted_at guard, a photo upload racing a concurrent
// GDPR erasure (DeleteCurrentUser) could commit after EraseUser had already
// anonymized the row and best-effort deleted the old object, silently
// writing a fresh photo_object_key onto an already-soft-deleted user. The
// image would then be permanently unreachable via the API (every read path
// filters deleted_at IS NULL) but its bytes would linger in the object store
// forever, since no retention job ever revisits an already-erased user --
// undermining the erasure guarantee EraseUser exists to provide. Returning
// pgx.ErrNoRows here lets the caller (Service.UpdatePhoto) clean up the
// just-uploaded object instead of leaving it orphaned.
func (r *Repository) UpdateUserPhoto(ctx context.Context, userID, objectKey string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE users SET photo_object_key = $2 WHERE id = $1 AND deleted_at IS NULL`,
		userID, objectKey,
	)
	if err != nil {
		return fmt.Errorf("auth.Repository.UpdateUserPhoto: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ensure uuid is used (uuid.UUID fields in UserRow/SessionRow).
var _ = uuid.UUID{}
