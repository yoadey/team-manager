package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
// anyone logs in). birthday is written for a newly created user only, if
// non-zero (a NULL/zero birthday leaves the column NULL, and an existing
// user's birthday is left untouched - "Existing account is left alone", see
// spec.md). Returns the user id and whether it was newly created.
func (s *Store) EnsureUser(ctx context.Context, email, name string, birthday time.Time) (id string, created bool, err error) {
	err = s.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, fmt.Errorf("db: look up user %s: %w", email, err)
	}

	if s.DryRun {
		return dryRunID, true, nil
	}

	placeholderHash, err := placeholderPasswordHash()
	if err != nil {
		return "", false, fmt.Errorf("db: generate placeholder password hash: %w", err)
	}

	var birthdayArg any
	if !birthday.IsZero() {
		birthdayArg = birthday
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO users (id, name, email, password_hash, birthday, email_verified_at, created_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		ON CONFLICT (email) DO NOTHING
	`, newID, name, email, placeholderHash, birthdayArg)
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

// UserPhotoKey returns the object-store key a user's photo is written
// under, matching the backend's own convention exactly
// (backend/internal/auth/service.go's userPhotoKey) so a photo imported by
// this tool is retrievable through the normal app the same way a
// user-uploaded one would be.
func UserPhotoKey(userID string) string {
	return "users/" + userID + "/photo"
}

// SetUserPhoto points userID's photo_object_key at key. Callers are
// responsible for having already uploaded the object at key to the object
// store - this only updates the DB pointer, mirroring the "S3 put before DB
// write" order the backend itself uses (backend/internal/auth/service.go's
// UpdatePhoto) so a DB-write failure never leaves the DB referencing a
// missing object; on the reverse, an uploaded-but-not-pointed-at object is
// merely unreachable, not corrupting, so this importer accepts that failure
// mode rather than also implementing best-effort delete-on-failure.
func (s *Store) SetUserPhoto(ctx context.Context, userID, key string) error {
	if s.DryRun {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `UPDATE users SET photo_object_key = $2 WHERE id = $1`, userID, key)
	if err != nil {
		return fmt.Errorf("db: set photo for user %s: %w", userID, err)
	}
	return nil
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
