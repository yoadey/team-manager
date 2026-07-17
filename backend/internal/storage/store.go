// Package storage provides object-storage access for team/user photos and
// logos, replacing the historical Postgres BYTEA columns. Callers hold only
// an object key; bytes live in an S3-compatible bucket and are delivered via
// short-lived presigned GET URLs, never streamed through the app server.
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrNotFound is returned by PresignGet/Delete when the given key does not
// exist in the store.
var ErrNotFound = errors.New("storage: object not found")

// ObjectStore puts, presigns, and deletes objects identified by an opaque
// key. Keys follow the scheme "teams/{teamID}/photo", "teams/{teamID}/logo",
// "users/{userID}/photo" (see key.go).
type ObjectStore interface {
	// Put uploads data under key, overwriting any existing object at that key.
	Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	// PresignGet returns a short-lived GET URL for key, or ErrNotFound if no
	// object exists at that key.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	// Delete removes the object at key. Deleting a key that does not exist is
	// not an error (idempotent, matching the DB-side "clear photo" semantics).
	Delete(ctx context.Context, key string) error
}
