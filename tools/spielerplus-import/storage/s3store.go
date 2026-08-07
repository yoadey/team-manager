// Package storage uploads member photos to the same S3-compatible object
// store Teamverwaltung's backend uses (backend/internal/storage), so an
// imported photo is retrievable through the normal app the same way a
// user-uploaded one would be. A separate, minimal (write-only) client
// rather than importing the backend's package directly: this tool is its
// own Go module, and Go's internal/ import restriction wouldn't allow it
// anyway (see design.md).
package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds the same connection details as the backend's own
// S3-compatible object store, using identical env var names
// (S3_ENDPOINT/S3_REGION/S3_BUCKET/S3_ACCESS_KEY_ID/S3_SECRET_ACCESS_KEY/
// S3_USE_PATH_STYLE - see CLAUDE.md) so an operator running this tool
// alongside the backend can reuse the same values.
type Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

// Store uploads photo bytes to an S3-compatible bucket. Write-only (Put
// only) - this importer never needs to read back or delete an object it
// just wrote.
type Store struct {
	client *minio.Client
	bucket string
}

// New creates a Store from cfg.
func New(cfg Config) (*Store, error) {
	host, secure := splitEndpointScheme(cfg.Endpoint)

	lookup := minio.BucketLookupAuto
	if cfg.UsePathStyle {
		lookup = minio.BucketLookupPath
	}

	client, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:       secure,
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.New: %w", err)
	}
	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// splitEndpointScheme strips an "http://"/"https://" prefix from raw,
// returning the bare host and whether TLS should be used - matching the
// backend's own S3Store (backend/internal/storage/s3store.go) exactly, so
// the same S3_ENDPOINT value works unmodified in both.
func splitEndpointScheme(raw string) (host string, secure bool) {
	if rest, ok := strings.CutPrefix(raw, "https://"); ok {
		return rest, true
	}
	if rest, ok := strings.CutPrefix(raw, "http://"); ok {
		return rest, false
	}
	return raw, true
}

// Put uploads data under key with the given content type, overwriting any
// existing object at that key.
func (s *Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("storage.Store.Put: %w", err)
	}
	return nil
}
