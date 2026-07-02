package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ErrDatabaseURLRequired is returned when DATABASE_URL is not set.
var ErrDatabaseURLRequired = errors.New("DATABASE_URL is required")

// ErrInvalidPositiveInt is returned when an integer env var is not a positive integer.
var ErrInvalidPositiveInt = errors.New("must be a positive integer")

// ErrInvalidCookieKey is returned when a cookie encryption key value cannot be
// decoded as a valid 32-byte hex- or base64-encoded value.
var ErrInvalidCookieKey = errors.New("cookie encryption key must be 32 bytes encoded as hex or base64")

// ErrInvalidPaginationHMACKey is returned when PAGINATION_HMAC_KEY is set but
// not a valid 32-byte hex- or base64-encoded value.
var ErrInvalidPaginationHMACKey = errors.New("PAGINATION_HMAC_KEY must be 32 bytes encoded as hex or base64")

// ErrInvalidDatabaseURL is returned when DATABASE_URL has an invalid format
// (wrong scheme or missing host).
var ErrInvalidDatabaseURL = errors.New("DATABASE_URL must use scheme 'postgres' or 'postgresql' and include a host")

// ErrCookieKeyRequired is returned when no cookie encryption key is configured
// while COOKIE_SECURE is true (production), where an ephemeral key is unsafe.
var ErrCookieKeyRequired = errors.New("COOKIE_ENCRYPTION_KEY (or COOKIE_ENCRYPTION_KEYS) is required when COOKIE_SECURE=true")

// ErrNoAllowedOrigins is returned when ALLOWED_ORIGINS is set but contains no
// usable (non-empty) origin after trimming, leaving nothing to build a public
// base URL or CORS allowlist from.
var ErrNoAllowedOrigins = errors.New("ALLOWED_ORIGINS must contain at least one non-empty origin")

// ErrJWTKeysRequired is returned when JWT_PRIVATE_KEY/JWT_PUBLIC_KEY are
// missing or only partially set while COOKIE_SECURE is true (production),
// where an ephemeral per-process key pair breaks sessions across restarts
// and multi-replica deployments.
var ErrJWTKeysRequired = errors.New("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY are both required when COOKIE_SECURE=true")

// ErrInvalidTrustedProxyCIDR is returned when TRUSTED_PROXY_CIDRS contains an
// entry that is not a valid CIDR (e.g. "10.0.0.0/8").
var ErrInvalidTrustedProxyCIDR = errors.New("TRUSTED_PROXY_CIDRS must be a comma-separated list of valid CIDRs")

// cookieKeySize is the AES-256 key length required for session cookie encryption.
const cookieKeySize = 32

// Config holds all runtime configuration for the server.
// CookieEncryptionKeys is an ordered list of AES-256 keys (newest first) for
// zero-downtime rotation: keys[0] encrypts; all keys are tried for decryption.
// Set via COOKIE_ENCRYPTION_KEYS (comma-separated) or COOKIE_ENCRYPTION_KEY (single).
type Config struct {
	Port                 string
	DatabaseURL          string
	JWTPrivateKey        string
	JWTPublicKey         string
	SessionTTL           time.Duration
	MigrationsDir        string
	AllowedOrigins       []string
	CookieEncryptionKeys [][]byte
	CookieSecure         bool
	CookieName           string
	PublicBaseURL        string
	MetricsToken         string
	SentryDSN            string
	// RateLimitRPS is the global per-IP request rate limit (requests per second).
	RateLimitRPS int
	// LoginRateLimitPerMin is the per-IP login attempt limit per minute.
	LoginRateLimitPerMin int
	// PaginationHMACKey is used to sign keyset pagination cursors (HMAC-SHA256)
	// so that clients cannot craft arbitrary cursor values. Optional: when nil,
	// cursors are plain base64 (dev mode). Set via PAGINATION_HMAC_KEY (32 bytes,
	// hex or base64).
	PaginationHMACKey []byte
	// RetentionNotificationDays is how many days to keep notification rows before
	// the daily retention job deletes them. Default: 90.
	RetentionNotificationDays int
	// RetentionSessionDays is how many days to keep session rows before the daily
	// retention job deletes them. Default: 30.
	RetentionSessionDays int
	// RetentionAuditLogDays is how many days to keep audit_log rows before the
	// daily retention job deletes them. Default: 365.
	RetentionAuditLogDays int
	// TrustedProxyCIDRs lists the CIDR ranges of reverse proxies/load balancers
	// allowed to set client-IP headers (X-Forwarded-For, X-Real-IP,
	// True-Client-IP) for rate limiting. Requests arriving directly from a peer
	// outside these ranges have their headers ignored, so a client cannot
	// bypass rate limiting by spoofing them. Empty by default (trust nothing —
	// rate limiting keys on the raw TCP peer address until explicitly
	// configured). Set via TRUSTED_PROXY_CIDRS (comma-separated CIDRs).
	TrustedProxyCIDRs []string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, ErrDatabaseURLRequired
	}
	if err := validateDatabaseURL(dbURL); err != nil {
		return nil, err
	}

	ttlHours, err := loadSessionTTLHours()
	if err != nil {
		return nil, err
	}

	origins, publicBaseURL, err := loadOriginsAndPublicBaseURL()
	if err != nil {
		return nil, err
	}

	cookieSecure, err := loadCookieSecure()
	if err != nil {
		return nil, err
	}

	cookieKeys, err := loadCookieEncryptionKeys(cookieSecure)
	if err != nil {
		return nil, err
	}

	jwtPrivateKey, jwtPublicKey, err := loadJWTKeys(cookieSecure)
	if err != nil {
		return nil, err
	}

	trustedProxyCIDRs, err := loadTrustedProxyCIDRs()
	if err != nil {
		return nil, err
	}

	rateLimitRPS, loginRateLimitPerMin, err := loadRateLimits()
	if err != nil {
		return nil, err
	}

	paginationHMACKey, err := loadOptionalBytesKey("PAGINATION_HMAC_KEY", cookieKeySize, ErrInvalidPaginationHMACKey)
	if err != nil {
		return nil, err
	}

	retentionNotificationDays, retentionSessionDays, retentionAuditLogDays, err := loadRetentionDays()
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:                      envOr("PORT", "8080"),
		DatabaseURL:               dbURL,
		JWTPrivateKey:             jwtPrivateKey,
		JWTPublicKey:              jwtPublicKey,
		SessionTTL:                time.Duration(ttlHours) * time.Hour,
		MigrationsDir:             envOr("MIGRATIONS_DIR", "internal/db/migrations"),
		AllowedOrigins:            origins,
		CookieEncryptionKeys:      cookieKeys,
		CookieSecure:              cookieSecure,
		CookieName:                os.Getenv("COOKIE_NAME"),
		PublicBaseURL:             publicBaseURL,
		MetricsToken:              os.Getenv("METRICS_TOKEN"),
		SentryDSN:                 os.Getenv("SENTRY_DSN"),
		RateLimitRPS:              rateLimitRPS,
		LoginRateLimitPerMin:      loginRateLimitPerMin,
		PaginationHMACKey:         paginationHMACKey,
		RetentionNotificationDays: retentionNotificationDays,
		RetentionSessionDays:      retentionSessionDays,
		RetentionAuditLogDays:     retentionAuditLogDays,
		TrustedProxyCIDRs:         trustedProxyCIDRs,
	}, nil
}

// loadSessionTTLHours reads SESSION_TTL_HOURS, defaulting to 720 (30 days).
// Rejects zero/negative values, which would otherwise produce sessions that
// expire before they're created and a non-positive cookie MaxAge.
func loadSessionTTLHours() (int, error) {
	n, err := parseInt(os.Getenv("SESSION_TTL_HOURS"), 720)
	if err != nil {
		return 0, fmt.Errorf("SESSION_TTL_HOURS: %w", err)
	}
	return n, nil
}

// loadOriginsAndPublicBaseURL reads ALLOWED_ORIGINS and derives PublicBaseURL,
// defaulting the latter to the first allowed origin (trailing slash trimmed)
// so a correctly configured deployment produces working shareable links
// (e.g. team invites) out of the box.
func loadOriginsAndPublicBaseURL() (origins []string, publicBaseURL string, err error) {
	origins = loadAllowedOrigins()
	if len(origins) == 0 {
		return nil, "", ErrNoAllowedOrigins
	}
	publicBaseURL = os.Getenv("PUBLIC_BASE_URL")
	if publicBaseURL == "" {
		publicBaseURL = origins[0]
	}
	return origins, strings.TrimRight(publicBaseURL, "/"), nil
}

// loadCookieSecure reads COOKIE_SECURE, defaulting to true (production-safe).
func loadCookieSecure() (bool, error) {
	v := os.Getenv("COOKIE_SECURE")
	if v == "" {
		return true, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	return b, nil
}

// loadJWTKeys reads JWT_PRIVATE_KEY/JWT_PUBLIC_KEY, failing loudly if either
// is missing while cookieSecure is true (production) — see ErrJWTKeysRequired.
func loadJWTKeys(cookieSecure bool) (privateKey, publicKey string, err error) {
	privateKey = os.Getenv("JWT_PRIVATE_KEY")
	publicKey = os.Getenv("JWT_PUBLIC_KEY")
	if cookieSecure && (privateKey == "" || publicKey == "") {
		return "", "", ErrJWTKeysRequired
	}
	return privateKey, publicKey, nil
}

// loadRateLimits reads RATE_LIMIT_RPS and LOGIN_RATE_LIMIT_PER_MIN.
func loadRateLimits() (rps, loginPerMin int, err error) {
	rps, err = parseInt(os.Getenv("RATE_LIMIT_RPS"), 100)
	if err != nil {
		return 0, 0, fmt.Errorf("RATE_LIMIT_RPS: %w", err)
	}
	loginPerMin, err = parseInt(os.Getenv("LOGIN_RATE_LIMIT_PER_MIN"), 5)
	if err != nil {
		return 0, 0, fmt.Errorf("LOGIN_RATE_LIMIT_PER_MIN: %w", err)
	}
	return rps, loginPerMin, nil
}

// loadRetentionDays reads the RETENTION_*_DAYS trio governing how long
// notifications, sessions, and audit log rows are kept before the daily
// retention job deletes them.
func loadRetentionDays() (notifications, sessions, auditLog int, err error) {
	notifications, err = parseInt(os.Getenv("RETENTION_NOTIFICATIONS_DAYS"), 90)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("RETENTION_NOTIFICATIONS_DAYS: %w", err)
	}
	sessions, err = parseInt(os.Getenv("RETENTION_SESSIONS_DAYS"), 30)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("RETENTION_SESSIONS_DAYS: %w", err)
	}
	auditLog, err = parseInt(os.Getenv("RETENTION_AUDIT_LOG_DAYS"), 365)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("RETENTION_AUDIT_LOG_DAYS: %w", err)
	}
	return notifications, sessions, auditLog, nil
}

// loadTrustedProxyCIDRs parses TRUSTED_PROXY_CIDRS into a slice of trimmed,
// non-empty CIDR strings, validating that each one parses. Empty/unset
// (default) means no proxy is trusted.
func loadTrustedProxyCIDRs() ([]string, error) {
	v := os.Getenv("TRUSTED_PROXY_CIDRS")
	if v == "" {
		return nil, nil
	}
	parts := strings.Split(v, ",")
	cidrs := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(trimmed); err != nil {
			return nil, fmt.Errorf("%w: %q", ErrInvalidTrustedProxyCIDR, trimmed)
		}
		cidrs = append(cidrs, trimmed)
	}
	return cidrs, nil
}

// validateDatabaseURL checks that dsn uses the postgres/postgresql scheme and
// includes a non-empty host, so DSN typos are caught at startup rather than
// producing a cryptic connection error later.
func validateDatabaseURL(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("DATABASE_URL: %w: %w", ErrInvalidDatabaseURL, err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return ErrInvalidDatabaseURL
	}
	if u.Host == "" {
		return ErrInvalidDatabaseURL
	}
	return nil
}

// loadAllowedOrigins parses ALLOWED_ORIGINS into a slice of trimmed, non-empty
// origin strings, falling back to localhost:5173 in development.
func loadAllowedOrigins() []string {
	v := os.Getenv("ALLOWED_ORIGINS")
	if v == "" {
		return []string{"http://localhost:5173"}
	}
	parts := strings.Split(v, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

// loadCookieEncryptionKeys reads cookie encryption keys from environment variables.
// It checks COOKIE_ENCRYPTION_KEYS (plural, comma-separated, newest key first) and
// falls back to COOKIE_ENCRYPTION_KEY (singular, backward-compatible). In production
// (secure=true) at least one key is required; in dev an ephemeral key is generated.
func loadCookieEncryptionKeys(secure bool) ([][]byte, error) {
	if raw := os.Getenv("COOKIE_ENCRYPTION_KEYS"); raw != "" {
		parts := strings.Split(raw, ",")
		keys := make([][]byte, 0, len(parts))
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key, err := decodeKey32(part)
			if err != nil {
				return nil, fmt.Errorf("COOKIE_ENCRYPTION_KEYS[%d]: %w", i, ErrInvalidCookieKey)
			}
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			return keys, nil
		}
	}
	if raw := os.Getenv("COOKIE_ENCRYPTION_KEY"); raw != "" {
		key, err := decodeKey32(raw)
		if err != nil {
			return nil, ErrInvalidCookieKey
		}
		return [][]byte{key}, nil
	}
	if secure {
		return nil, ErrCookieKeyRequired
	}
	key := make([]byte, cookieKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate dev cookie key: %w", err)
	}
	slog.Warn("COOKIE_ENCRYPTION_KEY not set; generated an ephemeral dev key — sessions will not survive a restart")
	return [][]byte{key}, nil
}

// decodeKey32 parses a hex- or base64-encoded 32-byte AES key.
func decodeKey32(raw string) ([]byte, error) {
	if key, err := hex.DecodeString(raw); err == nil && len(key) == cookieKeySize {
		return key, nil
	}
	if key, err := base64.StdEncoding.DecodeString(raw); err == nil && len(key) == cookieKeySize {
		return key, nil
	}
	return nil, ErrInvalidCookieKey
}

// loadOptionalBytesKey reads an optional environment variable that must be a
// hex- or base64-encoded byte slice of exactly wantLen bytes when set.
// Returns nil (no error) when the variable is unset or empty.
func loadOptionalBytesKey(envVar string, wantLen int, invalidErr error) ([]byte, error) {
	raw := os.Getenv(envVar)
	if raw == "" {
		return nil, nil
	}
	if key, err := hex.DecodeString(raw); err == nil && len(key) == wantLen {
		return key, nil
	}
	if key, err := base64.StdEncoding.DecodeString(raw); err == nil && len(key) == wantLen {
		return key, nil
	}
	return nil, invalidErr
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseInt parses a decimal integer from s, returning defaultVal when s is empty.
func parseInt(s string, defaultVal int) (int, error) {
	if s == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse integer: %w", err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("got %d: %w", n, ErrInvalidPositiveInt)
	}
	return n, nil
}
