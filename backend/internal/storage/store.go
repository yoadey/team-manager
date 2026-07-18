// Package storage abstracts image bytes away from Postgres into an
// S3-compatible object store. See openspec/changes/move-images-to-object-storage
// (archived into openspec/specs/image-storage once merged) for the design.
package storage

import (
	"context"
	"errors"
	"time"
)

// PresignTTL is the lifetime of a presigned image GET URL, shared by every
// caller of PresignGet (auth/teams/members). Short enough to bound how long a
// leaked URL stays useful, long enough to comfortably outlive a page load
// (including a browser retrying a slow image fetch).
const PresignTTL = 15 * time.Minute

// ErrObjectNotFound is returned by PresignGet (and may be returned by Delete,
// though implementations here treat deleting a missing key as a no-op) when
// the requested key does not exist in the store.
var ErrObjectNotFound = errors.New("storage: object not found")

// ObjectStore abstracts an S3-compatible object store for image bytes. The
// DB only ever holds the key returned by a successful Put; retrieval always
// goes through PresignGet so the application server never streams image
// bytes itself.
type ObjectStore interface {
	// Put uploads data under key with the given content type, overwriting any
	// existing object at that key.
	Put(ctx context.Context, key string, data []byte, contentType string) error
	// PresignGet returns a short-lived URL granting time-limited GET access to
	// the object at key. Returns ErrObjectNotFound if key does not exist.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	// Delete removes the object at key. Deleting a non-existent key is not an
	// error.
	Delete(ctx context.Context, key string) error
}
