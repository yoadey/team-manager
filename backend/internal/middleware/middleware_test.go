package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/middleware"
)

// ─── RequestID ───────────────────────────────────────────────────────────────

func TestRequestID_GeneratesID(t *testing.T) {
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middleware.GetRequestID(r.Context())
		assert.NotEmpty(t, id, "request ID must be stored in context")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"), "X-Request-ID header must be set on response")
}

func TestRequestID_PropagatesExistingID(t *testing.T) {
	const existingID = "trace-abc-123"

	var capturedID string
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, existingID, capturedID)
	assert.Equal(t, existingID, rec.Header().Get("X-Request-ID"))
}

func TestGetRequestID_EmptyWhenMissing(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	assert.Empty(t, middleware.GetRequestID(req.Context()))
}

// ─── Logger ──────────────────────────────────────────────────────────────────

func TestLogger_LogsRequestFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := middleware.RequestID(middleware.Logger(logger)(inner))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/teams", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NotEmpty(t, buf.String(), "logger must produce output")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))

	assert.Equal(t, "POST", entry["method"])
	assert.Equal(t, "/api/v1/teams", entry["path"])
	assert.EqualValues(t, http.StatusCreated, entry["status"])
	assert.Contains(t, entry, "duration")
	assert.Contains(t, entry, "request_id")
	assert.Equal(t, "request", entry["msg"])
}

func TestLogger_DefaultStatus200(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do not call WriteHeader explicitly
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.EqualValues(t, http.StatusOK, entry["status"])
}

// A panicking handler must still produce a "request" log line (with status
// forced to 500) — Recoverer sits above Logger in the real chain, so without
// a defer in Logger the panic would unwind straight past the logging call.
// The panic is re-raised by Logger, so this test recovers it itself
// (standing in for Recoverer).
func TestLogger_PanicStillLogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.Logger(logger)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/panic-test", http.NoBody)
	rec := httptest.NewRecorder()

	require.Panics(t, func() { handler.ServeHTTP(rec, req) })

	require.NotEmpty(t, buf.String(), "logger must still produce a request log line when the handler panics")
	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "request", entry["msg"])
	assert.EqualValues(t, http.StatusInternalServerError, entry["status"])
}

// ─── CORS ────────────────────────────────────────────────────────────────────

func TestCORS_AllowedOrigin(t *testing.T) {
	allowed := []string{"https://app.example.com", "https://staging.example.com"}
	handler := middleware.CORS(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCORS_UnknownOrigin_NoHeaders(t *testing.T) {
	allowed := []string{"https://app.example.com"}
	handler := middleware.CORS(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "unknown origin must not receive CORS headers")
	assert.Equal(t, http.StatusOK, rec.Code, "request must still be served (browser enforces the block)")
}

func TestCORS_Preflight_ReturnsNoContent(t *testing.T) {
	allowed := []string{"https://app.example.com"}
	handler := middleware.CORS(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler must not be called for preflight")
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORS_Preflight_UnknownOrigin(t *testing.T) {
	allowed := []string{"https://app.example.com"}
	var innerCalled bool
	handler := middleware.CORS(allowed)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://attacker.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, innerCalled, "inner handler is called for OPTIONS from unknown origin (no short-circuit)")
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

// ─── CSRF Origin check ─────────────────────────────────────────────────────────

func TestCSRFOriginCheck_AllowsSafeMethods(t *testing.T) {
	var called bool
	handler := middleware.CSRFOriginCheck([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://evil.example.com") // disallowed but GET is safe
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFOriginCheck_BlocksDisallowedOriginOnMutation(t *testing.T) {
	handler := middleware.CSRFOriginCheck([]string{"https://app.example.com"})(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			t.Fatal("inner handler must not run for a blocked cross-origin request")
		}),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCSRFOriginCheck_AllowsWhitelistedOriginOnMutation(t *testing.T) {
	var called bool
	handler := middleware.CSRFOriginCheck([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/data", http.NoBody)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFOriginCheck_AllowsMissingOriginOnMutation(t *testing.T) {
	var called bool
	handler := middleware.CSRFOriginCheck([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	// No Origin header — non-browser/API client must keep working.
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/data", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// A browser that classifies a state-changing request as cross-site must be
// blocked even if a (disallowed) Origin header happens to be absent -- the
// Sec-Fetch-Site metadata header is the authoritative browser signal.
func TestCSRFOriginCheck_BlocksCrossSiteFetchMetadata(t *testing.T) {
	handler := middleware.CSRFOriginCheck([]string{"https://app.example.com"})(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			t.Fatal("inner handler must not run for a cross-site request")
		}),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/data", http.NoBody)
	req.Header.Set("Sec-Fetch-Site", "cross-site") // no Origin header
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// A same-origin fetch (the app's own frontend) must be allowed.
func TestCSRFOriginCheck_AllowsSameOriginFetchMetadata(t *testing.T) {
	var called bool
	handler := middleware.CSRFOriginCheck([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/data", http.NoBody)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ─── Recoverer ───────────────────────────────────────────────────────────────

func TestRecoverer_CatchesPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.RequestID(
		middleware.Recoverer(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something exploded")
		})),
	)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/crash", http.NoBody)
	rec := httptest.NewRecorder()

	// Must not panic the test itself.
	require.NotPanics(t, func() {
		handler.ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.EqualValues(t, http.StatusInternalServerError, body["status"])
	// Regression: this response used to hardcode a literal "type" URI instead
	// of going through apierror, so it never honored ERROR_TYPE_BASE_URI.
	assert.Equal(t, apierror.Internal("").Type, body["type"],
		"panic-recovery response must use apierror's type URI, not a hardcoded literal")

	logOutput := buf.String()
	assert.True(t, strings.Contains(logOutput, "panic"), "log must mention the panic")
	assert.True(t, strings.Contains(logOutput, "stack"), "log must contain a stack trace field")
}

func TestRecoverer_NoPanic_PassesThrough(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := middleware.Recoverer(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, buf.String(), "no log output expected when there is no panic")
}

// ─── APIVersion ───────────────────────────────────────────────────────────────

func TestAPIVersion_SetsVersionHeader(t *testing.T) {
	handler := middleware.APIVersion("v1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/teams", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "v1", rec.Header().Get("API-Version"), "API-Version header must be set on every response")
}

func TestAPIVersion_DifferentVersion(t *testing.T) {
	handler := middleware.APIVersion("v2")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v2/teams", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "v2", rec.Header().Get("API-Version"))
}

func TestAPIVersion_DeprecationHeaders_WhenEnvSet(t *testing.T) {
	t.Setenv("API_DEPRECATION_DATE", "2027-01-01")

	handler := middleware.APIVersion("v1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/teams", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "v1", rec.Header().Get("API-Version"))
	assert.Equal(t, `date="2027-01-01"`, rec.Header().Get("Deprecation"), "Deprecation header must be set when API_DEPRECATION_DATE is configured")
	assert.Equal(t, "2027-01-01", rec.Header().Get("Sunset"), "Sunset header must be set when API_DEPRECATION_DATE is configured")
}

func TestAPIVersion_NoDeprecationHeaders_WhenEnvUnset(t *testing.T) {
	t.Setenv("API_DEPRECATION_DATE", "")

	handler := middleware.APIVersion("v1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/teams", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "v1", rec.Header().Get("API-Version"))
	assert.Empty(t, rec.Header().Get("Deprecation"), "Deprecation header must not be set when API_DEPRECATION_DATE is not configured")
	assert.Empty(t, rec.Header().Get("Sunset"), "Sunset header must not be set when API_DEPRECATION_DATE is not configured")
}

// ─── RateLimit trusted-proxy key func ──────────────────────────────────────
//
// Regression tests for the cross-tenant rate-limit bypass where
// httprate.KeyByRealIP trusted X-Forwarded-For/X-Real-IP unconditionally,
// letting any direct client spoof a fresh IP per request to dodge both the
// global limiter and the login brute-force limiter.

func newRateLimitedHandler(t *testing.T, trustedCIDRs []string) http.Handler {
	t.Helper()
	trusted, err := middleware.ParseTrustedProxies(trustedCIDRs)
	require.NoError(t, err)
	// requestsPerSecond=1 so a second request within the same window is
	// blocked only if it hashes to the same rate-limit key as the first.
	limiter := middleware.RateLimit(1, trusted)
	return limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestRateLimit_UntrustedPeer_IgnoresSpoofedForwardedFor(t *testing.T) {
	// No trusted proxies configured (the safe default) — a direct client
	// varying X-Forwarded-For per request must not be able to dodge the
	// limiter; both requests must key on the same raw RemoteAddr.
	handler := newRateLimitedHandler(t, nil)

	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "203.0.113.10:1111"
	req1.Header.Set("X-Forwarded-For", "1.1.1.1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code, "first request should be allowed")

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "203.0.113.10:2222" // same peer IP, different port
	req2.Header.Set("X-Forwarded-For", "2.2.2.2")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec2.Code,
		"second request from the same untrusted peer must be blocked despite a different spoofed X-Forwarded-For")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &body))
	// Regression: this response used to hardcode a literal "type" URI instead
	// of going through apierror, so it never honored ERROR_TYPE_BASE_URI.
	assert.Equal(t, apierror.New(http.StatusTooManyRequests, "Too Many Requests", "").Type, body["type"],
		"rate-limit response must use apierror's type URI, not a hardcoded literal")
}

func TestRateLimit_TrustedPeer_HonorsForwardedFor(t *testing.T) {
	// The peer is within the trusted CIDR (simulating a real reverse proxy),
	// so X-Forwarded-For is honored and each distinct client IP behind it
	// gets its own bucket.
	handler := newRateLimitedHandler(t, []string{"203.0.113.0/24"})

	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "203.0.113.10:1111"
	req1.Header.Set("X-Forwarded-For", "1.1.1.1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "203.0.113.10:2222" // same trusted proxy
	req2.Header.Set("X-Forwarded-For", "2.2.2.2")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code,
		"a different client IP behind a trusted proxy must get its own rate-limit bucket")
}

func TestRateLimit_TrustedPeer_SameForwardedFor_StillLimited(t *testing.T) {
	handler := newRateLimitedHandler(t, []string{"203.0.113.0/24"})

	for i, wantStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "203.0.113.10:1111"
		req.Header.Set("X-Forwarded-For", "9.9.9.9")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, wantStatus, rec.Code, "request %d", i)
	}
}

// Regression test: httprate.KeyByRealIP trusts a client-supplied
// True-Client-IP header verbatim, with no guarantee that a real reverse
// proxy/ingress overwrites rather than passes it through. A client sending a
// fresh True-Client-IP value on every request through a trusted proxy that
// doesn't scrub it could dodge both the global limiter and the login
// brute-force limiter entirely, even though TRUSTED_PROXY_CIDRS is
// correctly configured. Since no X-Forwarded-For is present here, both
// requests must fall back to the trusted peer's own address and share a
// bucket regardless of what True-Client-IP claims.
func TestRateLimit_TrustedPeer_IgnoresSpoofedTrueClientIP(t *testing.T) {
	handler := newRateLimitedHandler(t, []string{"203.0.113.0/24"})

	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "203.0.113.10:1111"
	req1.Header.Set("True-Client-IP", "1.1.1.1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "203.0.113.10:2222" // same trusted peer, different port
	req2.Header.Set("True-Client-IP", "2.2.2.2")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec2.Code,
		"a spoofed True-Client-IP must not let a request from the same trusted peer dodge the limiter")
}

// Same regression as above, for X-Real-IP.
func TestRateLimit_TrustedPeer_IgnoresSpoofedXRealIP(t *testing.T) {
	handler := newRateLimitedHandler(t, []string{"203.0.113.0/24"})

	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "203.0.113.10:1111"
	req1.Header.Set("X-Real-IP", "1.1.1.1")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "203.0.113.10:2222" // same trusted peer, different port
	req2.Header.Set("X-Real-IP", "2.2.2.2")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec2.Code,
		"a spoofed X-Real-IP must not let a request from the same trusted peer dodge the limiter")
}

// Regression test: httprate.KeyByRealIP takes X-Forwarded-For's left-most
// entry, which is exactly the part of the header a client controls. Behind
// a chain of trusted proxies (each appending its own hop), the right-most
// entry not itself in the trusted CIDR set is the actual client -- picking
// the left-most instead lets a client dodge the limiter by varying its own
// claimed value while the real (trusted) hop stays constant.
func TestRateLimit_TrustedPeer_ForwardedFor_UsesRightmostUntrustedHop(t *testing.T) {
	handler := newRateLimitedHandler(t, []string{"203.0.113.0/24"})

	// Same real client (5.5.5.5, the right-most/last-appended hop from the
	// trusted proxy chain) on both requests, but a different attacker-chosen
	// left-most entry each time -- must still be recognized as the same
	// client and blocked on the second request.
	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "203.0.113.10:1111"
	req1.Header.Set("X-Forwarded-For", "1.1.1.1, 5.5.5.5")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "203.0.113.10:2222"
	req2.Header.Set("X-Forwarded-For", "9.9.9.9, 5.5.5.5")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec2.Code,
		"varying the client-controlled left-most X-Forwarded-For entry must not dodge the limiter when the real (right-most) hop is unchanged")

	// A genuinely different real client (different right-most hop) must
	// still get its own bucket.
	req3 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req3.RemoteAddr = "203.0.113.10:3333"
	req3.Header.Set("X-Forwarded-For", "1.1.1.1, 6.6.6.6")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code,
		"a different real (right-most) client hop must get its own rate-limit bucket")
}

// Regression test: http.Header.Get returns only the FIRST stored value for a
// header name, but HTTP permits X-Forwarded-For to arrive as several
// separate header lines rather than one comma-joined value -- per RFC 7230
// §3.2.2 those are semantically equivalent to a single header with all
// values joined in appearance order. realForwardedIP used Header.Get, so a
// trusted proxy that appends its own hop as a NEW header line (rather than
// joining it onto the client's existing value) had that hop silently
// dropped: the code walked only the client's own, fully-controlled first
// line as if it were the complete chain, reopening the exact
// client-controlled-hop spoof the right-to-left walk exists to close.
func TestRateLimit_TrustedPeer_ForwardedFor_UsesSecondHeaderLine(t *testing.T) {
	handler := newRateLimitedHandler(t, []string{"203.0.113.0/24"})

	// Same real client (5.5.5.5) on both requests, appended as a SEPARATE
	// X-Forwarded-For header line (not comma-joined) by the trusted proxy,
	// with a different attacker-chosen first line each time -- must still be
	// recognized as the same client and blocked on the second request.
	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "203.0.113.10:1111"
	req1.Header.Add("X-Forwarded-For", "1.1.1.1")
	req1.Header.Add("X-Forwarded-For", "5.5.5.5")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "203.0.113.10:2222"
	req2.Header.Add("X-Forwarded-For", "9.9.9.9")
	req2.Header.Add("X-Forwarded-For", "5.5.5.5")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusTooManyRequests, rec2.Code,
		"a trusted proxy's hop appended as a second X-Forwarded-For header line must still be found, not dropped by only reading the first line")
}

// ─── Metrics ─────────────────────────────────────────────────────────────────

// TestMetrics_LabelsUseRoutePatternNotRawPath is a regression test: recording
// metrics with the raw request path (e.g. containing a UUID) instead of the
// matched chi route pattern gives every distinct ID its own Prometheus label
// combination, an unbounded-cardinality memory-growth vector reachable by
// any caller since this middleware runs before auth.
// A panicking handler must not leak the in-flight gauge or skip the
// request/duration sample — Recoverer sits above Metrics in the real chain,
// so without a defer in Metrics the panic would unwind straight past its
// bookkeeping. The panic is re-raised by Metrics, so this test recovers it
// itself (standing in for Recoverer).
func TestMetrics_PanicStillRecordsRequestAndDecrementsInFlight(t *testing.T) {
	handler := middleware.Metrics(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/panic-test", http.NoBody)
	rec := httptest.NewRecorder()

	require.Panics(t, func() { handler.ServeHTTP(rec, req) })

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	var inFlight, total *dto.MetricFamily
	for _, f := range families {
		switch f.GetName() {
		case "teammanager_http_requests_in_flight":
			inFlight = f
		case "teammanager_http_requests_total":
			total = f
		}
	}
	require.NotNil(t, inFlight)
	require.Len(t, inFlight.Metric, 1)
	assert.Equal(t, float64(0), inFlight.Metric[0].GetGauge().GetValue(),
		"in-flight gauge must be decremented even when the handler panics")

	require.NotNil(t, total)
	var found500 bool
	for _, m := range total.Metric {
		for _, l := range m.Label {
			if l.GetName() == "status" && l.GetValue() == "500" {
				found500 = true
			}
		}
	}
	assert.True(t, found500, "a panicking request must still be recorded, with status forced to 500")
}

func TestMetrics_LabelsUseRoutePatternNotRawPath(t *testing.T) {
	router := chi.NewRouter()
	router.Use(middleware.Metrics)
	router.Get("/api/v1/teams/{teamId}/events/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, id := range []string{"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
			"/api/v1/teams/team-a/events/"+id, http.NoBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	var found *dto.MetricFamily
	for _, f := range families {
		if f.GetName() == "teammanager_http_requests_total" {
			found = f
			break
		}
	}
	require.NotNil(t, found, "teammanager_http_requests_total must be registered")

	var matchingSeries int
	for _, m := range found.Metric {
		for _, l := range m.Label {
			if l.GetName() == "path" {
				assert.NotContains(t, l.GetValue(), "11111111-1111-1111-1111-111111111111",
					"path label must not contain the raw UUID from the request path")
				if l.GetValue() == "/api/v1/teams/{teamId}/events/{id}" {
					matchingSeries++
				}
			}
		}
	}
	// Both requests (different UUIDs) must collapse into the single route-pattern series.
	assert.Equal(t, 1, matchingSeries, "both requests must share one label series keyed by route pattern")
}
