package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config configures an S3Store.
type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	// UseSSL selects https vs http when talking to Endpoint.
	UseSSL bool
	// UsePathStyle selects path-style addressing (bucket.example.com/key
	// becomes example.com/bucket/key), required by most S3-compatible stores
	// (MinIO, etc.) that don't do virtual-hosted-style DNS.
	UsePathStyle bool
	// PublicBaseURL, when set, replaces the scheme+host of presigned URLs.
	// Needed when Endpoint (used for server-to-store calls, e.g. a MinIO
	// Service DNS name inside the cluster) is not reachable by the browser
	// that follows the presigned redirect.
	PublicBaseURL string
}

// S3Store implements ObjectStore against an S3-compatible bucket via minio-go.
type S3Store struct {
	client        *minio.Client
	bucket        string
	publicBaseURL *url.URL
}

// NewS3Store creates an S3Store from cfg. It does not verify the bucket
// exists or is reachable — callers that want a fail-fast startup check
// should call a HEAD/stat themselves.
func NewS3Store(cfg S3Config) (*S3Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: bucketLookupType(cfg.UsePathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("storage.NewS3Store: %w", err)
	}

	var publicBaseURL *url.URL
	if cfg.PublicBaseURL != "" {
		publicBaseURL, err = url.Parse(cfg.PublicBaseURL)
		if err != nil {
			return nil, fmt.Errorf("storage.NewS3Store: parse S3_PUBLIC_BASE_URL: %w", err)
		}
	}

	return &S3Store{client: client, bucket: cfg.Bucket, publicBaseURL: publicBaseURL}, nil
}

func bucketLookupType(usePathStyle bool) minio.BucketLookupType {
	if usePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupDNS
}

// Put uploads data under key, overwriting any existing object at that key.
func (s *S3Store) Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, data, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("storage.S3Store.Put: %w", err)
	}
	return nil
}

// PresignGet returns a short-lived GET URL for key, or ErrNotFound if no
// object exists at that key. The existence check is a separate StatObject
// call (minio's presign itself never touches the network) so a stale DB
// object-key row that lost its backing object surfaces as a clean 404
// instead of a presigned URL that 403s/404s at the client.
func (s *S3Store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if _, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{}); err != nil {
		var errResp minio.ErrorResponse
		if errors.As(err, &errResp) && (errResp.Code == "NoSuchKey" || errResp.StatusCode == 404) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("storage.S3Store.PresignGet: stat: %w", err)
	}

	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("storage.S3Store.PresignGet: %w", err)
	}
	if s.publicBaseURL != nil {
		u.Scheme = s.publicBaseURL.Scheme
		u.Host = s.publicBaseURL.Host
	}
	return u.String(), nil
}

// Delete removes the object at key. Deleting a key that does not exist is
// not an error.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage.S3Store.Delete: %w", err)
	}
	return nil
}
