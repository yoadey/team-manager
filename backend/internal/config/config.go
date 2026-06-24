package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ErrDatabaseURLRequired is returned when DATABASE_URL is not set.
var ErrDatabaseURLRequired = errors.New("DATABASE_URL is required")

type Config struct {
	Port           string
	DatabaseURL    string
	JWTPrivateKey  string
	JWTPublicKey   string
	SessionTTL     time.Duration
	MigrationsDir  string
	AllowedOrigins []string
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

	return &Config{
		Port:           port,
		DatabaseURL:    dbURL,
		JWTPrivateKey:  os.Getenv("JWT_PRIVATE_KEY"),
		JWTPublicKey:   os.Getenv("JWT_PUBLIC_KEY"),
		SessionTTL:     time.Duration(ttlHours) * time.Hour,
		MigrationsDir:  envOr("MIGRATIONS_DIR", "internal/db/migrations"),
		AllowedOrigins: origins,
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
