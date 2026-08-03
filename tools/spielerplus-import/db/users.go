package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost matches the backend's own cost factor
// (backend/internal/auth/service.go), so a placeholder hash isn't
// distinguishably cheaper/more-expensive than a real one.
const bcryptCost = 12

// EnsureUser finds an existing user by email, or creates one that's
// immediately usable through Teamverwaltung's forgot-password flow: a
// non-empty but unusable placeholder password_hash (so ForgotPassword
// doesn't treat it as an OIDC-only account) and email_verified_at set
// immediately (so the retention job doesn't purge it as unverified before
// anyone logs in). Returns the user id and whether it was newly created.
func (s *Store) EnsureUser(ctx context.Context, email, name string) (id string, created bool, err error) {
	err = s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err == nil {
		return id, false, nil
	}

	if s.DryRun {
		return dryRunID, true, nil
	}

	placeholderHash, err := placeholderPasswordHash()
	if err != nil {
		return "", false, fmt.Errorf("db: generate placeholder password hash: %w", err)
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO users (id, name, email, password_hash, email_verified_at, created_at)
		VALUES ($1, $2, $3, $4, now(), now())
		ON CONFLICT (email) DO NOTHING
	`, newID, name, email, placeholderHash)
	if err != nil {
		return "", false, fmt.Errorf("db: insert user %s: %w", email, err)
	}

	// Re-select: either our insert won, or a concurrent run/insert raced us
	// (ON CONFLICT DO NOTHING) - either way, the email is now unique in the table.
	err = s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err != nil {
		return "", false, fmt.Errorf("db: select user %s after insert: %w", email, err)
	}
	return id, id == newID, nil
}

func placeholderPasswordHash() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(buf)), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
