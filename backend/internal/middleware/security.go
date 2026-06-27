package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// RequireBearerToken returns middleware that rejects any request whose
// Authorization header does not carry the exact bearer token. The comparison is
// constant-time to avoid leaking the token via timing. Intended to guard
// operational endpoints such as /metrics, which would otherwise expose internal
// telemetry to anyone who can reach the service.
func RequireBearerToken(token string) func(http.Handler) http.Handler {
	const prefix = "Bearer "
	want := []byte(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			got := []byte(strings.TrimPrefix(h, prefix))
			if !strings.HasPrefix(h, prefix) || subtle.ConstantTimeCompare(got, want) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders adds defensive HTTP response headers to every response.
// These headers protect against common browser-based attacks (clickjacking,
// MIME-sniffing, information leakage) and enforce HTTPS once deployed.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		// API responses are never navigable documents; default-src 'none' ensures
		// any accidental text/html response cannot load external resources, and
		// frame-ancestors 'none' prevents clickjacking via iframe embedding.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
