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
	"time"
)

// ErrDatabaseURLRequired is returned when DATABASE_URL is not set.
var ErrDatabaseURLRequired = errors.New("DATABASE_URL is required")

// ErrInvalidCookieKey is returned when COOKIE_ENCRYPTION_KEY is set but not a
// valid 32-byte hex- or base64-encoded value.
var ErrInvalidCookieKey = errors.New("COOKIE_ENCRYPTION_KEY must be 32 bytes encoded as hex or base64")

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
		origins = []string{v}
	}

	cookieKey, err := loadCookieEncryptionKey()
	if err != nil {
		return nil, err
	}

	cookieSecure := true
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("COOKIE_SECURE: %w", err)
		}
		cookieSecure = b
	}

	return &Config{
		Port:                port,
		DatabaseURL:         dbURL,
		JWTPrivateKey:       os.Getenv("JWT_PRIVATE_KEY"),
		JWTPublicKey:        os.Getenv("JWT_PUBLIC_KEY"),
		SessionTTL:          time.Duration(ttlHours) * time.Hour,
		MigrationsDir:       envOr("MIGRATIONS_DIR", "internal/db/migrations"),
		AllowedOrigins:      origins,
		CookieEncryptionKey: cookieKey,
		CookieSecure:        cookieSecure,
		CookieName:          os.Getenv("COOKIE_NAME"),
	}, nil
}

// loadCookieEncryptionKey reads COOKIE_ENCRYPTION_KEY (hex or base64, 32 bytes).
// When unset, a random ephemeral key is generated for dev use, with a warning
// (sessions will not survive a restart in that case).
func loadCookieEncryptionKey() ([]byte, error) {
	raw := os.Getenv("COOKIE_ENCRYPTION_KEY")
	if raw == "" {
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
