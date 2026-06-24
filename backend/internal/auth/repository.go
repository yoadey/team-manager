package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	q := fmt.Sprintf(`SELECT %s FROM users WHERE email = $1`, selectUserFields)
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
	q := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, selectUserFields)
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
