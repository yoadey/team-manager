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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
