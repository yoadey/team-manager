// Package middleware provides composable HTTP middleware for the team-manager API.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/yoadey/team-manager/backend/internal/metrics"
)

// ─── Request ID ──────────────────────────────────────────────────────────────

type contextKey string

const requestIDKey contextKey = "request_id"

// errPanicRecovered is the static base error recorded on the active trace span
// when a panic is caught (wrapped with the recovered value).
var errPanicRecovered = errors.New("panic recovered")

// RequestID generates a UUID v4 for each request, injects it into the context,
// and echoes it via the X-Request-ID response header.
// If the incoming request already carries an X-Request-ID header its value is
// preferred, which enables end-to-end tracing across service boundaries.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID stored in ctx by RequestID middleware.
// Returns an empty string when no ID is present.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// ─── Structured Logger ───────────────────────────────────────────────────────

// responseWriter wraps http.ResponseWriter to capture the HTTP status code
// written by the downstream handler.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// Logger returns structured request logging middleware backed by the supplied
// slog.Logger. Each request produces a single log record containing the HTTP
// method, path, status code, duration, and request ID (when present).
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", GetRequestID(r.Context())),
			}
			// Correlate logs with traces when a span is active.
			if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
				attrs = append(attrs,
					slog.String("trace_id", sc.TraceID().String()),
					slog.String("span_id", sc.SpanID().String()),
				)
			}
			logger.InfoContext(r.Context(), "request", attrs...)
		})
	}
}

// ─── Rate Limiter ────────────────────────────────────────────────────────────

// makeLimitHandler returns a httprate option that responds with a Problem Details
// 429, a Retry-After header, and increments the rate-limit-hit counter.
// context is a label for the metrics counter ("global" or "login").
func makeLimitHandler(window time.Duration, context string) httprate.Option {
	retryAfter := strconv.Itoa(max(1, int(window.Seconds())))
	return httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
		metrics.RateLimitHits.WithLabelValues(context).Inc()
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusTooManyRequests)
		body := map[string]any{
			"type":   "https://teammanager.example/errors/too-many-requests",
			"title":  "Too Many Requests",
			"status": http.StatusTooManyRequests,
			"detail": "rate limit exceeded; please slow down",
		}
		_ = json.NewEncoder(w).Encode(body)
	})
}

// RateLimit returns middleware that limits each client IP to requestsPerSecond
// per second using a sliding-window counter.
//
// Keying by IP (rather than a single global counter) means one noisy client
// cannot exhaust the request budget for everyone, and per-instance throughput
// is not artificially capped by aggregate traffic. Behind a proxy/load balancer
// this relies on X-Forwarded-For / X-Real-IP being set correctly.
func RateLimit(requestsPerSecond int) func(http.Handler) http.Handler {
	return httprate.NewRateLimiter(
		requestsPerSecond,
		time.Second,
		makeLimitHandler(time.Second, "global"),
		httprate.WithKeyFuncs(httprate.KeyByRealIP),
	).Handler
}

// PerIPRateLimit returns middleware that limits each unique remote IP to
// requestsPerPeriod within period. Intended for sensitive endpoints such as
// login where brute-force protection is critical.
func PerIPRateLimit(requestsPerPeriod int, period time.Duration) func(http.Handler) http.Handler {
	return httprate.NewRateLimiter(
		requestsPerPeriod,
		period,
		makeLimitHandler(period, "login"),
		httprate.WithKeyFuncs(httprate.KeyByRealIP),
	).Handler
}

// ─── Body Size Limiter ───────────────────────────────────────────────────────

// BodyLimit wraps each request body with an io.LimitedReader capped at maxBytes.
// Requests whose bodies exceed the limit result in a 413 Request Entity Too Large.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── CORS ────────────────────────────────────────────────────────────────────

// CORS returns middleware that handles Cross-Origin Resource Sharing.
// Only origins present in allowedOrigins are permitted; unknown origins receive
// no CORS headers (effectively blocked by the browser).
// Preflight (OPTIONS) requests are answered immediately with 204 No Content.
// Credentials (cookies, Authorization) are allowed for matching origins.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")

				if r.Method == http.MethodOptions {
					// Preflight
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}

				w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ─── CSRF (Origin check) ───────────────────────────────────────────────────────

// csrfSafeMethods are methods that cannot mutate state and are exempt from the
// Origin check.
var csrfSafeMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// CSRFOriginCheck provides defense-in-depth against CSRF for cookie-based auth.
// Because the session is carried by a cookie the browser attaches automatically,
// SameSite=Lax is the primary defense; this middleware adds a second layer: for
// state-changing methods it rejects requests whose Origin header is present but
// not in the whitelist. A missing Origin is allowed so non-browser API clients
// and same-origin requests that omit it keep working — a forged cross-site
// browser request always carries a (disallowed) Origin and is blocked.
func CSRFOriginCheck(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, safe := csrfSafeMethods[r.Method]; !safe {
				if origin := r.Header.Get("Origin"); origin != "" {
					if _, ok := allowed[origin]; !ok {
						writeProblem(w, http.StatusForbidden, "cross-origin request blocked")
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── Recoverer ───────────────────────────────────────────────────────────────

// Recoverer returns middleware that catches panics, logs them with a stack
// trace using the supplied slog.Logger, and writes a 500 Internal Server Error
// RFC 9457 Problem Details response.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					logger.ErrorContext(
						ctx, "panic recovered",
						slog.Any("panic", rec),
						slog.String("stack", string(stack)),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("request_id", GetRequestID(ctx)),
					)

					// Report to Sentry (no-op when not initialized) and mark the
					// active trace span as errored (no-op when tracing is off).
					sentry.CurrentHub().Recover(rec)
					span := trace.SpanFromContext(ctx)
					span.RecordError(fmt.Errorf("%w: %v", errPanicRecovered, rec))
					span.SetStatus(codes.Error, "panic recovered")

					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					body := map[string]any{
						"type":   "https://teammanager.example/errors/internal-server-error",
						"title":  "Internal Server Error",
						"status": http.StatusInternalServerError,
						"detail": "an unexpected error occurred",
					}
					_ = json.NewEncoder(w).Encode(body)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
