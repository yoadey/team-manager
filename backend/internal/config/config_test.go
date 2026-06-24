package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/config"
)

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("PORT", "")
	t.Setenv("SESSION_TTL_HOURS", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 720*time.Hour, cfg.SessionTTL)
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.AllowedOrigins)
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("PORT", "9090")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
}

func TestLoad_CustomSessionTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("SESSION_TTL_HOURS", "48")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 48*time.Hour, cfg.SessionTTL)
}

func TestLoad_InvalidSessionTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("SESSION_TTL_HOURS", "not-a-number")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoad_CustomAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("ALLOWED_ORIGINS", "https://example.com")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com"}, cfg.AllowedOrigins)
}

func TestLoad_MigrationsDir(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	dir := t.TempDir()
	t.Setenv("MIGRATIONS_DIR", dir)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, dir, cfg.MigrationsDir)
}

func TestLoad_JWTKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("JWT_PRIVATE_KEY", "private-key-pem")
	t.Setenv("JWT_PUBLIC_KEY", "public-key-pem")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "private-key-pem", cfg.JWTPrivateKey)
	assert.Equal(t, "public-key-pem", cfg.JWTPublicKey)
}

func TestLoad_AllowedOriginsMultiple(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("ALLOWED_ORIGINS", "https://a.com,https://b.com")

	cfg, err := config.Load()
	require.NoError(t, err)
	// Single env value (no splitting by comma yet — the config treats it as one origin)
	assert.Len(t, cfg.AllowedOrigins, 1)
}

func TestLoad_EnvOr(t *testing.T) {
	os.Unsetenv("MIGRATIONS_DIR") //nolint:errcheck // Unsetenv never fails in tests
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "internal/db/migrations", cfg.MigrationsDir)
}
