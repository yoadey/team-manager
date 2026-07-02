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
	assert.Len(t, cfg.CookieEncryptionKeys[0], 32)
	assert.False(t, cfg.CookieSecure)
}

func TestLoad_CookieKeyFromHex(t *testing.T) {
	// 32 bytes encoded as 64 hex chars.
	const hexKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", hexKey)
	t.Setenv("JWT_PRIVATE_KEY", "private-key-pem")
	t.Setenv("JWT_PUBLIC_KEY", "public-key-pem")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, cfg.CookieEncryptionKeys[0], 32)
	assert.True(t, cfg.CookieSecure)
}

func TestLoad_CookieKeyFromBase64(t *testing.T) {
	// 32 zero bytes, standard base64.
	const b64Key = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", b64Key)
	t.Setenv("JWT_PRIVATE_KEY", "private-key-pem")
	t.Setenv("JWT_PUBLIC_KEY", "public-key-pem")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, cfg.CookieEncryptionKeys[0], 32)
}

func TestLoad_CookieKeyInvalidLength(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "deadbeef") // 4 bytes, too short

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrInvalidCookieKey)
}

func TestLoad_CookieEncryptionKeysPlural(t *testing.T) {
	// Two valid 32-byte keys as hex — newest key first.
	const key0 = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	const key1 = "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100"
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "")
	t.Setenv("COOKIE_ENCRYPTION_KEYS", key0+","+key1)
	t.Setenv("JWT_PRIVATE_KEY", "private-key-pem")
	t.Setenv("JWT_PUBLIC_KEY", "public-key-pem")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Len(t, cfg.CookieEncryptionKeys, 2)
	assert.Len(t, cfg.CookieEncryptionKeys[0], 32)
	assert.Len(t, cfg.CookieEncryptionKeys[1], 32)
}

func TestLoad_CookieEncryptionKeysPluralInvalidEntry(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_ENCRYPTION_KEYS", "notakey")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "")

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrInvalidCookieKey)
}

func TestLoad_AllowedOriginsAllEmpty(t *testing.T) {
	// Regression test: ALLOWED_ORIGINS set to only commas/whitespace used to
	// panic with "index out of range [0] with length 0" inside Load() (before
	// the HTTP server/Recoverer middleware exist) instead of returning a
	// clean, actionable config error.
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ALLOWED_ORIGINS", ", ,")

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrNoAllowedOrigins)
}

func TestLoad_JWTKeysRequiredWhenSecure(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrJWTKeysRequired)
}

func TestLoad_JWTKeysPartialRequiredWhenSecure(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_ENCRYPTION_KEY", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	t.Setenv("JWT_PRIVATE_KEY", "private-key-pem")
	t.Setenv("JWT_PUBLIC_KEY", "")

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrJWTKeysRequired)
}

func TestLoad_JWTKeysEphemeralInDev(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.JWTPrivateKey)
	assert.Empty(t, cfg.JWTPublicKey)
}

func TestLoad_TrustedProxyCIDRsDefaultEmpty(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXY_CIDRS", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.TrustedProxyCIDRs)
}

func TestLoad_TrustedProxyCIDRsParsed(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 172.16.0.0/12")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.0/8", "172.16.0.0/12"}, cfg.TrustedProxyCIDRs)
}

func TestLoad_TrustedProxyCIDRsInvalid(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr")

	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrInvalidTrustedProxyCIDR)
}

func TestLoad_DatabaseURLInvalidScheme(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://user:pass@localhost/db")
	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrInvalidDatabaseURL)
}

func TestLoad_DatabaseURLMissingHost(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres:///db")
	_, err := config.Load()
	require.ErrorIs(t, err, config.ErrInvalidDatabaseURL)
}
