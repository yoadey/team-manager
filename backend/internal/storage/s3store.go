package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config holds the connection details for NewS3Store.
type S3Config struct {
	// Endpoint is the S3-compatible host, optionally prefixed with "http://"
	// or "https://" to override the default (secure). Region-qualified AWS
	// endpoints (e.g. "s3.eu-central-1.amazonaws.com") and bare host:port
	// values (e.g. "minio:9000" for local/in-cluster MinIO) are both valid.
	Endpoint string
	Region   string
	Bucket   string
	// AccessKeyID/SecretAccessKey are static credentials. Left blank to fall
	// back to no credentials, which only works against a publicly writable
	// bucket -- not a supported production configuration.
	AccessKeyID     string
	SecretAccessKey string
	// UsePathStyle forces path-style addressing (https://host/bucket/key)
	// instead of virtual-hosted-style (https://bucket.host/key). Required for
	// most self-hosted S3-compatible stores (MinIO) that don't do wildcard
	// DNS/TLS for arbitrary bucket subdomains.
	UsePathStyle bool
	// PublicBaseURL, when set, replaces the scheme+host of presigned URLs
	// after signing. Needed when the endpoint the backend connects to
	// (in-cluster/Compose service DNS, e.g. "minio:9000") differs from the
	// endpoint a browser can actually reach (e.g. "https://minio.example.com"
	// or a CDN in front of the bucket) -- the signature itself doesn't cover
	// the host, so swapping it post-signing is safe.
	PublicBaseURL string
}

// S3Store implements ObjectStore against an S3-compatible endpoint (AWS S3 or
// MinIO) via minio-go.
type S3Store struct {
	client        *minio.Client
	bucket        string
	publicBaseURL *url.URL
}

// NewS3Store creates an S3Store from cfg.
func NewS3Store(cfg S3Config) (*S3Store, error) {
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
		return nil, fmt.Errorf("storage.NewS3Store: %w", err)
	}

	s := &S3Store{client: client, bucket: cfg.Bucket}
	if cfg.PublicBaseURL != "" {
		u, err := url.Parse(cfg.PublicBaseURL)
		if err != nil {
			return nil, fmt.Errorf("storage.NewS3Store: invalid PublicBaseURL: %w", err)
		}
		s.publicBaseURL = u
	}
	return s, nil
}

// splitEndpointScheme strips an "http://"/"https://" prefix from raw,
// returning the bare host and whether TLS should be used. Endpoints given
// without a scheme default to secure (matching real S3/production MinIO
// behind TLS); local dev opts in to plaintext by specifying "http://".
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
func (s *S3Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("storage.S3Store.Put: %w", err)
	}
	return nil
}

// PresignGet returns a short-lived URL granting time-limited GET access to
// the object at key.
func (s *S3Store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if _, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{}); err != nil {
		var errResp minio.ErrorResponse
		if errors.As(err, &errResp) && errResp.Code == "NoSuchKey" {
			return "", ErrObjectNotFound
		}
		return "", fmt.Errorf("storage.S3Store.PresignGet: stat: %w", err)
	}

	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("storage.S3Store.PresignGet: %w", err)
	}
	if s.publicBaseURL != nil {
		u.Scheme = s.publicBaseURL.Scheme
		u.Host = s.publicBaseURL.Host
	}
	return u.String(), nil
}

// Delete removes the object at key. Deleting a non-existent key is not an
// error (minio-go's RemoveObject already treats a missing key as success).
func (s *S3Store) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage.S3Store.Delete: %w", err)
	}
	return nil
}
