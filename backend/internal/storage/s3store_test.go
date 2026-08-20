package storage_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5" // used only for a stub ETag on a test-only fake S3 server, not real cryptography; gosec is excluded for _test.go files repo-wide
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/storage"
)

// fakeS3Server is a minimal S3-compatible HTTP server covering exactly the
// operations S3Store issues (PUT/HEAD/GET on a single-part object, path-style
// addressing). It does not verify request signatures -- these tests exercise
// S3Store's own logic (error mapping, URL handling), not minio-go's request
// signing, so an unauthenticated fake is sufficient. Modeled on the
// httptest.Server-based fake used for the webpush HTTP dependency in
// internal/push/webpush_test.go, since this codebase fakes HTTP-based
// external services with a handler rather than pulling in a dependency
// like gofakes3.
type fakeS3Server struct {
	mu     sync.Mutex
	bucket string
	// objects and contentTypes are keyed by object key (not full path).
	objects      map[string][]byte
	contentTypes map[string]string
}

func newFakeS3Server(t *testing.T, bucket string) *httptest.Server {
	t.Helper()
	f := &fakeS3Server{
		bucket:       bucket,
		objects:      map[string][]byte{},
		contentTypes: map[string]string{},
	}
	prefix := "/" + bucket + "/"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, prefix)

		f.mu.Lock()
		defer f.mu.Unlock()

		switch r.Method {
		case http.MethodPut:
			body, err := readObjectBody(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			f.objects[key] = body
			f.contentTypes[key] = r.Header.Get("Content-Type")
			w.Header().Set("ETag", etagFor(body))
			w.WriteHeader(http.StatusOK)
		case http.MethodHead:
			data, ok := f.objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeObjectHeaders(w, key, data, f.contentTypes[key])
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			data, ok := f.objects[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeObjectHeaders(w, key, data, f.contentTypes[key])
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		case http.MethodDelete:
			// Real S3's DeleteObject returns 204 unconditionally, including for
			// a key that never existed -- mirrored here so the fake matches
			// what S3Store.Delete relies on (see store.go's doc comment).
			delete(f.objects, key)
			delete(f.contentTypes, key)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func writeObjectHeaders(w http.ResponseWriter, key string, data []byte, contentType string) {
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("ETag", etagFor(data))
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	_ = key
}

// readObjectBody reads a PUT request body, transparently decoding the
// "aws-chunked" streaming-signature framing minio-go's PutObject uses by
// default for SigV4 requests with a known Content-Length (each chunk is
// "<hex-size>;chunk-signature=<hex>\r\n<data>\r\n", terminated by a
// zero-size chunk) -- real S3/MinIO decode this server-side; this fake must
// too, or every Put in these tests would store the signed chunk framing
// instead of the actual object bytes.
func readObjectBody(r *http.Request) ([]byte, error) {
	if strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") ||
		strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
		return decodeAWSChunkedBody(r.Body)
	}
	return io.ReadAll(r.Body)
}

func decodeAWSChunkedBody(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(r)
	var out bytes.Buffer
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading chunk header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		sizeHex := line
		if idx := strings.IndexByte(line, ';'); idx >= 0 {
			sizeHex = line[:idx]
		}
		size, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk size %q: %w", sizeHex, err)
		}
		if size == 0 {
			break // final chunk; any trailing headers are ignored (none in these tests)
		}
		buf := make([]byte, size)
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, fmt.Errorf("reading chunk data: %w", err)
		}
		out.Write(buf)
		if _, err := br.ReadString('\n'); err != nil { // trailing CRLF after chunk data
			return nil, fmt.Errorf("reading chunk trailer: %w", err)
		}
	}
	return out.Bytes(), nil
}

func etagFor(data []byte) string {
	return fmt.Sprintf(`"%x"`, md5.Sum(data)) // stub ETag for a test-only fake, not a security control
}

// newTestS3Store builds an S3Store against server, with sensible defaults
// tests can override via mutate. The bucket is always "photos", matching
// every fake server these tests spin up via newFakeS3Server(t, "photos").
func newTestS3Store(t *testing.T, server *httptest.Server, mutate func(*storage.S3Config)) *storage.S3Store {
	t.Helper()
	cfg := storage.S3Config{
		Endpoint:        server.URL, // includes "http://" prefix -- exercises splitEndpointScheme's real parsing.
		Region:          "us-east-1",
		Bucket:          "photos",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		UsePathStyle:    true,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	s, err := storage.NewS3Store(cfg)
	require.NoError(t, err)
	return s
}

// TestSplitEndpointScheme exercises splitEndpointScheme's scheme-defaulting
// logic indirectly (it's unexported, and this package's tests are
// intentionally black-box like the rest of internal/storage's *_test.go
// files). The fake server in newFakeS3Server only speaks plain HTTP, so a
// config that resolves to secure=true (bare host, or an explicit "https://"
// prefix) must fail with a TLS-handshake-shaped connection error against it,
// while "http://" must succeed -- proving the scheme was actually parsed and
// plumbed into the minio client's Secure option, not just cosmetically
// stripped from the host string.
func TestSplitEndpointScheme(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "test-bucket")
	bareHost := strings.TrimPrefix(server.URL, "http://")

	cases := []struct {
		name       string
		endpoint   string
		wantSecure bool
	}{
		{"bare host defaults to secure (https)", bareHost, true},
		{"explicit https:// prefix is treated as secure", "https://" + bareHost, true},
		{"explicit http:// prefix is treated as insecure", "http://" + bareHost, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := storage.NewS3Store(storage.S3Config{
				Endpoint:        c.endpoint,
				Region:          "us-east-1",
				Bucket:          "test-bucket",
				AccessKeyID:     "ak",
				SecretAccessKey: "sk",
				UsePathStyle:    true,
			})
			require.NoError(t, err)

			putErr := s.Put(context.Background(), "k", []byte("v"), "text/plain")
			if c.wantSecure {
				require.Error(t, putErr, "a TLS client talking to a plain-HTTP fake server must fail the handshake")
			} else {
				require.NoError(t, putErr)
			}
		})
	}
}

func TestS3Store_PutGetDelete_Roundtrip(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, nil)
	ctx := context.Background()

	require.NoError(t, s.Put(ctx, "teams/t1/photo", []byte("hello"), "image/jpeg"))

	rc, contentType, err := s.Get(ctx, "teams/t1/photo")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	assert.Equal(t, "image/jpeg", contentType)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), data)

	require.NoError(t, s.Delete(ctx, "teams/t1/photo"))
}

func TestS3Store_Get_NotFound(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, nil)

	_, _, err := s.Get(context.Background(), "does/not/exist")
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)
}

func TestS3Store_PresignGet_NotFound(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, nil)

	_, err := s.PresignGet(context.Background(), "does/not/exist", time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)
}

// TestS3Store_PresignGet_GenericErrorIsNotObjectNotFound is the key
// regression test for the finding this file addresses: a connection-level
// failure (e.g. the object store being unreachable) must not be conflated
// with "the key doesn't exist" -- callers (see teams/members handlers) branch
// on ErrObjectNotFound to return a 404 versus a 5xx for anything else, so
// mapping a transient infra failure to ErrObjectNotFound would incorrectly
// tell clients an existing photo/logo doesn't exist.
func TestS3Store_PresignGet_GenericErrorIsNotObjectNotFound(t *testing.T) {
	t.Parallel()

	// Start and immediately close a server to get a syntactically valid but
	// unreachable endpoint -- connections to it are refused.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	s := newTestS3Store(t, server, nil)

	_, err := s.PresignGet(context.Background(), "some/key", time.Minute)
	require.Error(t, err)
	assert.NotErrorIs(t, err, storage.ErrObjectNotFound,
		"a connection failure must surface as a generic error, not be conflated with a missing object")
}

func TestS3Store_Get_GenericErrorIsNotObjectNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	s := newTestS3Store(t, server, nil)

	_, _, err := s.Get(context.Background(), "some/key")
	require.Error(t, err)
	assert.NotErrorIs(t, err, storage.ErrObjectNotFound,
		"a connection failure must surface as a generic error, not be conflated with a missing object")
}

func TestS3Store_PresignGet_DefaultHost(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, nil) // no PublicBaseURL set
	ctx := context.Background()

	require.NoError(t, s.Put(ctx, "k", []byte("v"), "text/plain"))
	presigned, err := s.PresignGet(ctx, "k", time.Minute)
	require.NoError(t, err)

	u, err := url.Parse(presigned)
	require.NoError(t, err)
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	assert.Equal(t, serverURL.Host, u.Host, "without PublicBaseURL the presigned URL host must be the endpoint itself")
	assert.Equal(t, serverURL.Scheme, u.Scheme)
	assert.Contains(t, u.Path, "k")
}

func TestS3Store_PresignGet_PublicBaseURLOverride(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, func(cfg *storage.S3Config) {
		cfg.PublicBaseURL = "https://cdn.example.com"
	})
	ctx := context.Background()

	require.NoError(t, s.Put(ctx, "teams/t1/logo", []byte("v"), "image/png"))
	presigned, err := s.PresignGet(ctx, "teams/t1/logo", time.Minute)
	require.NoError(t, err)

	u, err := url.Parse(presigned)
	require.NoError(t, err)
	assert.Equal(t, "https", u.Scheme, "PublicBaseURL's scheme must replace the endpoint's")
	assert.Equal(t, "cdn.example.com", u.Host, "PublicBaseURL's host must replace the endpoint's")
	assert.Contains(t, u.Path, "teams/t1/logo", "the signed path/query must be preserved, only scheme+host swap")
	assert.NotEmpty(t, u.RawQuery, "the presign signature/query params must survive the host swap")
}

func TestS3Store_PresignGet_PublicBaseURLNotFoundStillChecksFirst(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, func(cfg *storage.S3Config) {
		cfg.PublicBaseURL = "https://cdn.example.com"
	})

	// Never Put -- PresignGet must still stat first and report not-found,
	// rather than happily presigning a URL for an object that was never
	// uploaded.
	_, err := s.PresignGet(context.Background(), "missing", time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrObjectNotFound)
}

func TestS3Store_Delete_NonExistentKeyIsNotError(t *testing.T) {
	t.Parallel()

	server := newFakeS3Server(t, "photos")
	s := newTestS3Store(t, server, nil)

	require.NoError(t, s.Delete(context.Background(), "never/uploaded"))
}
