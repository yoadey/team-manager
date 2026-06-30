package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// ErrDatabaseURLRequired is returned when DATABASE_URL is not set.
var ErrDatabaseURLRequired = errors.New("DATABASE_URL is required")

// ErrInvalidPositiveInt is returned when an integer env var is not a positive integer.
var ErrInvalidPositiveInt = errors.New("must be a positive integer")

// ErrInvalidCookieKey is returned when COOKIE_ENCRYPTION_KEY is set but not a
// valid 32-byte hex- or base64-encoded value.
var ErrInvalidCookieKey = errors.New("COOKIE_ENCRYPTION_KEY must be 32 bytes encoded as hex or base64")

// ErrInvalidPaginationHMACKey is returned when PAGINATION_HMAC_KEY is set but
// not a valid 32-byte hex- or base64-encoded value.
var ErrInvalidPaginationHMACKey = errors.New("PAGINATION_HMAC_KEY must be 32 bytes encoded as hex or base64")

// ErrCookieKeyRequired is returned when COOKIE_ENCRYPTION_KEY is unset while
// COOKIE_SECURE is true (production), where an ephemeral key is unsafe.
var ErrCookieKeyRequired = errors.New("COOKIE_ENCRYPTION_KEY is required when COOKIE_SECURE=true")

// cookieKeySize is the AES-256 key length required for session cookie encryption.
const cookieKeySize = 32

type Config struct {
	Port                string
	DatabaseURL         string
	JWTPrivateKey       string
	JWTPublicKey        string
	SessionTTL          time.Duration
	MigrationsDir       string
	AllowedOrigins      []string
	CookieEncryptionKey []byte
	CookieSecure        bool
	CookieName          string
	PublicBaseURL       string
	MetricsToken        string
	SentryDSN           string
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
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, ErrDatabaseURLRequired
	}

	ttlHours := 720 // 30 days default
	if v := os.Getenv("SESSION_TTL_HOURS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("SESSION_TTL_HOURS: %w", err)
		}
		ttlHours = n
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	origins := []string{"http://localhost:5173"}
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		parts := strings.Split(v, ",")
		origins = make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}

	// Public base URL of the user-facing frontend, used to build shareable links
	// (e.g. team invite links). Defaults to the first allowed origin so a
	// correctly configured deployment produces working links out of the box.
	publicBaseURL := strings.TrimRight(envOr("PUBLIC_BASE_URL", origins[0]), "/")

	cookieSecure := true
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("COOKIE_SECURE: %w", err)
		}
		cookieSecure = b
	}

	cookieKey, err := loadCookieEncryptionKey(cookieSecure)
	if err != nil {
		return nil, err
	}

	rateLimitRPS, err := parseInt(os.Getenv("RATE_LIMIT_RPS"), 100)
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_RPS: %w", err)
	}
	loginRateLimitPerMin, err := parseInt(os.Getenv("LOGIN_RATE_LIMIT_PER_MIN"), 5)
	if err != nil {
		return nil, fmt.Errorf("LOGIN_RATE_LIMIT_PER_MIN: %w", err)
	}

	paginationHMACKey, err := loadOptionalBytesKey("PAGINATION_HMAC_KEY", cookieKeySize, ErrInvalidPaginationHMACKey)
	if err != nil {
		return nil, err
	}

	retentionNotificationDays, err := parseInt(os.Getenv("RETENTION_NOTIFICATIONS_DAYS"), 90)
	if err != nil {
		return nil, fmt.Errorf("RETENTION_NOTIFICATIONS_DAYS: %w", err)
	}
	retentionSessionDays, err := parseInt(os.Getenv("RETENTION_SESSIONS_DAYS"), 30)
	if err != nil {
		return nil, fmt.Errorf("RETENTION_SESSIONS_DAYS: %w", err)
	}

	return &Config{
		Port:                 port,
		DatabaseURL:          dbURL,
		JWTPrivateKey:        os.Getenv("JWT_PRIVATE_KEY"),
		JWTPublicKey:         os.Getenv("JWT_PUBLIC_KEY"),
		SessionTTL:           time.Duration(ttlHours) * time.Hour,
		MigrationsDir:        envOr("MIGRATIONS_DIR", "internal/db/migrations"),
		AllowedOrigins:       origins,
		CookieEncryptionKey:  cookieKey,
		CookieSecure:         cookieSecure,
		CookieName:           os.Getenv("COOKIE_NAME"),
		PublicBaseURL:        publicBaseURL,
		MetricsToken:         os.Getenv("METRICS_TOKEN"),
		SentryDSN:            os.Getenv("SENTRY_DSN"),
		RateLimitRPS:              rateLimitRPS,
		LoginRateLimitPerMin:      loginRateLimitPerMin,
		PaginationHMACKey:         paginationHMACKey,
		RetentionNotificationDays: retentionNotificationDays,
		RetentionSessionDays:      retentionSessionDays,
	}, nil
}

// loadCookieEncryptionKey reads COOKIE_ENCRYPTION_KEY (hex or base64, 32 bytes).
// When unset in a secure (production) configuration it is a hard error — an
// ephemeral key would silently invalidate every session on restart and break
// horizontal scaling (each instance would generate a different key). When unset
// with COOKIE_SECURE=false (local dev) a random ephemeral key is generated with
// a warning instead.
func loadCookieEncryptionKey(secure bool) ([]byte, error) {
	raw := os.Getenv("COOKIE_ENCRYPTION_KEY")
	if raw == "" {
		if secure {
			return nil, ErrCookieKeyRequired
		}
		key := make([]byte, cookieKeySize)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate dev cookie key: %w", err)
		}
		slog.Warn("COOKIE_ENCRYPTION_KEY not set; generated an ephemeral dev key — sessions will not survive a restart")
		return key, nil
	}
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
