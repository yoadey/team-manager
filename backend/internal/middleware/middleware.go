// Package middleware provides composable HTTP middleware for the team-manager API.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/yoadey/team-manager/backend/internal/apierror"
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
//
// The log line is emitted from a defer so a panicking handler still produces
// a "request" record (with status forced to 500) — Recoverer wraps this
// middleware, so without the defer a panic would unwind straight past the
// logging call below. The panic is re-raised afterward so Recoverer still
// handles the response and its own "panic recovered" log line as before.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			start := time.Now()
			rw := newResponseWriter(w)

			defer func() {
				status := rw.status
				rec := recover()
				if rec != nil {
					status = http.StatusInternalServerError
				}
				attrs := []any{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", GetRequestID(ctx)),
				}
				// Correlate logs with traces when a span is active.
				if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
					attrs = append(attrs,
						slog.String("trace_id", sc.TraceID().String()),
						slog.String("span_id", sc.SpanID().String()),
					)
				}
				logger.InfoContext(ctx, "request", attrs...)
				if rec != nil {
					panic(rec)
				}
			}()

			next.ServeHTTP(rw, r)
		})
	}
}

// ─── Rate Limiter ────────────────────────────────────────────────────────────

// makeLimitHandler returns a httprate option that responds with a Problem Details
// 429, a Retry-After header, and increments the rate-limit-hit counter.
// limitCtx is a label for the metrics counter ("global" or "login").
func makeLimitHandler(window time.Duration, limitCtx string) httprate.Option {
	retryAfter := strconv.Itoa(max(1, int(window.Seconds())))
	return httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
		metrics.RateLimitHits.WithLabelValues(limitCtx).Inc()
		w.Header().Set("Retry-After", retryAfter)
		apierror.New(http.StatusTooManyRequests, "Too Many Requests", "rate limit exceeded; please slow down").Render(w)
	})
}

// ParseTrustedProxies parses CIDR strings (already validated by
// config.Load) into *net.IPNet values for use by trustedProxyKeyFunc.
func ParseTrustedProxies(cidrs []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipNet, err := net.ParseCIDR(c)
		if err != nil {
			return nil, fmt.Errorf("middleware.ParseTrustedProxies: %q: %w", c, err)
		}
		nets = append(nets, ipNet)
	}
	return nets, nil
}

// trustedProxyKeyFunc returns an httprate.KeyFunc that only honors the
// client-supplied X-Forwarded-For header when the immediate TCP peer
// address falls within a trusted CIDR. For any other peer, the header is
// ignored and the raw connection address is used instead — this prevents a
// client from bypassing rate limiting (including login brute-force
// protection) by spoofing the header, while still supporting the common
// reverse-proxy/load-balancer deployment when its address range is
// explicitly configured.
//
// This deliberately does not use httprate.KeyByRealIP, which also honors
// True-Client-IP and X-Real-IP verbatim and takes X-Forwarded-For's
// left-most (client-supplied) hop. Both single-value headers can be set
// directly by the client with no guarantee that every reverse
// proxy/ingress/LB overwrites rather than passes them through, and the
// left-most X-Forwarded-For hop is, by definition, whatever the client
// itself claimed. See realForwardedIP for the hop-walking this uses
// instead.
func trustedProxyKeyFunc(trusted []*net.IPNet) httprate.KeyFunc {
	return func(r *http.Request) (string, error) {
		if isTrustedPeer(r.RemoteAddr, trusted) {
			return realForwardedIP(r, trusted), nil
		}
		return httprate.KeyByIP(r)
	}
}

// realForwardedIP returns the rate-limit key for a request whose immediate
// peer is a trusted proxy. It walks X-Forwarded-For from right (closest hop)
// to left, skipping any entry that itself falls within a trusted CIDR --
// each proxy in a trusted chain may append its own hop -- and returns the
// first (right-most) entry that isn't trusted, i.e. the client the trusted
// proxy chain itself observed. Unlike taking the left-most entry, this
// cannot be spoofed by a client-supplied value: everything the client wrote
// into the header is to the left of, and therefore skipped in favor of, the
// first hop actually appended by a trusted proxy. If every entry is trusted
// (or the header is absent/unparseable), it falls back to the immediate
// peer address.
//
// Uses r.Header.Values, not Header.Get: HTTP permits X-Forwarded-For to
// appear as several separate header lines rather than one comma-joined
// value, and per RFC 7230 §3.2.2 those are semantically equivalent to a
// single header with all values joined in appearance order -- Header.Get
// returns only the first such line, silently discarding any hop a trusted
// proxy appended as its own line rather than joining it onto the existing
// value. Missing that hop would mean walking only the client's own,
// unclaimed first line as if it were the complete chain, reopening the
// exact spoofable-header gap this function exists to close.
func realForwardedIP(r *http.Request, trusted []*net.IPNet) string {
	if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
		var hops []string
		for _, v := range values {
			hops = append(hops, strings.Split(v, ",")...)
		}
		for i := len(hops) - 1; i >= 0; i-- {
			host := strings.TrimSpace(hops[i])
			ip := net.ParseIP(host)
			if ip == nil {
				continue
			}
			trustedHop := false
			for _, n := range trusted {
				if n.Contains(ip) {
					trustedHop = true
					break
				}
			}
			if !trustedHop {
				return canonicalizeIP(host)
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return canonicalizeIP(host)
}

// canonicalizeIP mirrors httprate's unexported canonicalizeIP: IPv4
// addresses are returned as-is, IPv6 addresses are masked to their /64 so
// clients can't evade rate limiting by cycling through addresses within
// their own /64, and non-IP strings pass through unchanged.
func canonicalizeIP(ip string) string {
	isIPv6 := false
	for i := 0; !isIPv6 && i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return ip
		case ':':
			isIPv6 = true
		}
	}
	if !isIPv6 {
		return ip
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}

// isTrustedPeer reports whether remoteAddr's host portion falls within any
// of the trusted CIDR ranges. No ranges configured means nothing is trusted.
func isTrustedPeer(remoteAddr string, trusted []*net.IPNet) bool {
	if len(trusted) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// RateLimit returns middleware that limits each client IP to requestsPerSecond
// per second using a sliding-window counter.
//
// Keying by IP (rather than a single global counter) means one noisy client
// cannot exhaust the request budget for everyone, and per-instance throughput
// is not artificially capped by aggregate traffic. trustedProxies restricts
// which peers' X-Forwarded-For/X-Real-IP headers are honored — see
// trustedProxyKeyFunc.
func RateLimit(requestsPerSecond int, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return httprate.NewRateLimiter(
		requestsPerSecond,
		time.Second,
		makeLimitHandler(time.Second, "global"),
		httprate.WithKeyFuncs(trustedProxyKeyFunc(trustedProxies)),
	).Handler
}

// PerIPRateLimit returns middleware that limits each unique remote IP to
// requestsPerPeriod within period. Intended for sensitive endpoints such as
// login where brute-force protection is critical. See RateLimit for the
// trustedProxies semantics.
func PerIPRateLimit(requestsPerPeriod int, period time.Duration, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return httprate.NewRateLimiter(
		requestsPerPeriod,
		period,
		makeLimitHandler(period, "login"),
		httprate.WithKeyFuncs(trustedProxyKeyFunc(trustedProxies)),
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
// SameSite=Lax is the primary defense; this middleware adds a second layer for
// state-changing methods:
//
//  1. If Sec-Fetch-Site is "cross-site", the browser itself has classified the
//     request as cross-site — block it unconditionally. Modern browsers always
//     send this metadata header, so it catches a forged request even in the
//     edge case where a (disallowed) Origin header is somehow absent.
//  2. Otherwise, if an Origin header is present, reject it unless whitelisted.
//
// A request with neither a disallowed Origin nor a cross-site Sec-Fetch-Site is
// allowed, so non-browser API clients and same-origin requests keep working.
func CSRFOriginCheck(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, safe := csrfSafeMethods[r.Method]; !safe {
				if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
					writeProblem(w, http.StatusForbidden, "cross-origin request blocked")
					return
				}
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

// ─── API Version ─────────────────────────────────────────────────────────────

// APIVersion returns middleware that sets the API-Version response header to
// the supplied version string on every response.
//
// When the environment variable API_DEPRECATION_DATE is non-empty, it is also
// emitted as both a Deprecation and a Sunset header so that Enterprise clients
// can programmatically detect and act on pending API lifecycle changes.
//
// Example:
//
//	r.Use(middleware.APIVersion("v1"))
func APIVersion(version string) func(http.Handler) http.Handler {
	deprecationDate := os.Getenv("API_DEPRECATION_DATE")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("API-Version", version)
			if deprecationDate != "" {
				w.Header().Set("Deprecation", `date="`+deprecationDate+`"`)
				w.Header().Set("Sunset", deprecationDate)
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

					apierror.Internal("an unexpected error occurred").Render(w)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
