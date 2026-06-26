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
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("PORT", "")
	t.Setenv("SESSION_TTL_HOURS", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 720*time.Hour, cfg.SessionTTL)
	assert.Equal(t, []string{"http://localhost:5173"}, cfg.AllowedOrigins)
	// PublicBaseURL defaults to the first allowed origin.
	assert.Equal(t, "http://localhost:5173", cfg.PublicBaseURL)
}

func TestLoad_PublicBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("PUBLIC_BASE_URL", "https://app.example.com/")

	cfg, err := config.Load()
	require.NoError(t, err)
	// Trailing slash is trimmed so links don't get a double slash.
	assert.Equal(t, "https://app.example.com", cfg.PublicBaseURL)
}

func TestLoad_PublicBaseURLDefaultsToAllowedOrigin(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("ALLOWED_ORIGINS", "https://example.com")
	t.Setenv("PUBLIC_BASE_URL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.PublicBaseURL)
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("PORT", "9090")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
}

func TestLoad_CustomSessionTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("SESSION_TTL_HOURS", "48")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 48*time.Hour, cfg.SessionTTL)
}

func TestLoad_InvalidSessionTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("SESSION_TTL_HOURS", "not-a-number")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoad_CustomAllowedOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("ALLOWED_ORIGINS", "https://example.com")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"https://example.com"}, cfg.AllowedOrigins)
}

func TestLoad_MigrationsDir(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	dir := t.TempDir()
	t.Setenv("MIGRATIONS_DIR", dir)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, dir, cfg.MigrationsDir)
}

func TestLoad_JWTKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("JWT_PRIVATE_KEY", "private-key-pem")
	t.Setenv("JWT_PUBLIC_KEY", "public-key-pem")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "private-key-pem", cfg.JWTPrivateKey)
	assert.Equal(t, "public-key-pem", cfg.JWTPublicKey)
}

func TestLoad_AllowedOriginsMultiple(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key
	t.Setenv("ALLOWED_ORIGINS", "https://a.com,https://b.com")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, cfg.AllowedOrigins, 2)
	assert.Equal(t, []string{"https://a.com", "https://b.com"}, cfg.AllowedOrigins)
}

func TestLoad_EnvOr(t *testing.T) {
	os.Unsetenv("MIGRATIONS_DIR") //nolint:errcheck // Unsetenv never fails in tests
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false") // dev mode: allow ephemeral cookie key

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "internal/db/migrations", cfg.MigrationsDir)
}

func TestLoad_CookieKeyRequiredWhenSecure(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "")

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrCookieKeyRequired)
}

func TestLoad_CookieKeyEphemeralInDev(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, cfg.CookieEncryptionKey, 32)
	assert.False(t, cfg.CookieSecure)
}

func TestLoad_CookieKeyFromHex(t *testing.T) {
	// 32 bytes encoded as 64 hex chars.
	const hexKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", hexKey)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, cfg.CookieEncryptionKey, 32)
	assert.True(t, cfg.CookieSecure)
}

func TestLoad_CookieKeyFromBase64(t *testing.T) {
	// 32 zero bytes, standard base64.
	const b64Key = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", b64Key)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, cfg.CookieEncryptionKey, 32)
}

func TestLoad_CookieKeyInvalidLength(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "deadbeef") // 4 bytes, too short

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrInvalidCookieKey)
}
