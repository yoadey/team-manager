// Package middleware provides composable HTTP middleware for the team-manager API.
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/httprate"
	"github.com/google/uuid"
)

// ─── Request ID ──────────────────────────────────────────────────────────────

type contextKey string

const requestIDKey contextKey = "request_id"

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

			logger.InfoContext(r.Context(), "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", GetRequestID(r.Context())),
			)
		})
	}
}

// ─── Rate Limiter ────────────────────────────────────────────────────────────

// RateLimit returns middleware that limits requests to requestsPerSecond per
// second using a sliding-window counter. Clients that exceed the limit receive
// a 429 Too Many Requests response with an RFC 9457 Problem Details body.
func RateLimit(requestsPerSecond int) func(http.Handler) http.Handler {
	limiter := httprate.NewRateLimiter(
		requestsPerSecond,
		time.Second,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusTooManyRequests)
			body := map[string]any{
				"type":   "https://teammanager.example/errors/too-many-requests",
				"title":  "Too Many Requests",
				"status": http.StatusTooManyRequests,
				"detail": "rate limit exceeded; please slow down",
			}
			_ = json.NewEncoder(w).Encode(body)
		}),
	)
	return limiter.Handler
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

// ─── Recoverer ───────────────────────────────────────────────────────────────

// Recoverer returns middleware that catches panics, logs them with a stack
// trace using the supplied slog.Logger, and writes a 500 Internal Server Error
// RFC 9457 Problem Details response.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					logger.ErrorContext(r.Context(), "panic recovered",
						slog.Any("panic", rec),
						slog.String("stack", string(stack)),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.String("request_id", GetRequestID(r.Context())),
					)

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
