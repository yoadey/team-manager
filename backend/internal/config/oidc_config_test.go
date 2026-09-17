package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/config"
)

// baseEnv sets the minimum a config.Load needs, so each test below only has to
// describe its own OIDC_* inputs.
func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
}

func TestLoad_OIDCDisabledByDefault(t *testing.T) {
	baseEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.False(t, cfg.OIDC.Enabled)
	assert.Empty(t, cfg.OIDC.Issuer)
}

func TestLoad_OIDCEnabled(t *testing.T) {
	baseEnv(t)
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://sso.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client")
	t.Setenv("OIDC_CLIENT_SECRET", "secret")
	t.Setenv("OIDC_PROVIDER_ID", "google")
	t.Setenv("OIDC_PROVIDER_NAME", "Google")
	t.Setenv("OIDC_PROVIDER_ICON", "/provider-icons/google.svg")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.True(t, cfg.OIDC.Enabled)
	assert.Equal(t, "https://sso.example.com", cfg.OIDC.Issuer)
	assert.Equal(t, "google", cfg.OIDC.ProviderID)
	assert.Equal(t, []string{"openid", "profile", "email"}, cfg.OIDC.Scopes)
	// Derived from PUBLIC_BASE_URL (itself derived from ALLOWED_ORIGINS) so
	// the common deployment needs no explicit redirect configuration.
	assert.Equal(t, "https://app.example.com/api/v1/auth/oidc/callback", cfg.OIDC.RedirectURL)
}

func TestLoad_OIDCExplicitRedirectURLWins(t *testing.T) {
	baseEnv(t)
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://sso.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client")
	t.Setenv("OIDC_CLIENT_SECRET", "secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://other.example.com/cb")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "https://other.example.com/cb", cfg.OIDC.RedirectURL)
}

// The extra scopes are the mechanism a deployment uses to pass a
// provider-specific hint (e.g. a ZITADEL IdP selection scope) without the
// application knowing that provider.
func TestLoad_OIDCExtraScopesAreAppendedAndDeduplicated(t *testing.T) {
	baseEnv(t)
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://sso.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client")
	t.Setenv("OIDC_CLIENT_SECRET", "secret")
	t.Setenv("OIDC_SCOPES", "openid profile email")
	t.Setenv("OIDC_EXTRA_SCOPES", "email urn:zitadel:iam:org:idp:id:12345")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"openid", "profile", "email", "urn:zitadel:iam:org:idp:id:12345"},
		cfg.OIDC.Scopes)
}

func TestLoad_OIDCMissingCredentialsFailsStartup(t *testing.T) {
	for _, missing := range []string{"OIDC_ISSUER", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET"} {
		t.Run("without_"+missing, func(t *testing.T) {
			baseEnv(t)
			t.Setenv("OIDC_ENABLED", "true")
			t.Setenv("OIDC_ISSUER", "https://sso.example.com")
			t.Setenv("OIDC_CLIENT_ID", "client")
			t.Setenv("OIDC_CLIENT_SECRET", "secret")
			t.Setenv(missing, "")

			_, err := config.Load()
			require.ErrorIs(t, err, config.ErrOIDCConfigRequired)
		})
	}
}

// The icon ends up as an <img src> on the unauthenticated login screen, so the
// scheme allow-list is enforced at startup rather than sanitized later.
func TestLoad_OIDCIconValidation(t *testing.T) {
	cases := []struct {
		icon    string
		wantErr bool
	}{
		{"", false},
		{"/provider-icons/google.svg", false},
		{"https://cdn.example.com/google.svg", false},
		{"javascript:alert(1)", true},
		{"data:image/svg+xml;base64,PHN2Zy8+", true},
		{"//evil.example.com/x.svg", true},
		{"http://cdn.example.com/google.svg", true},
		// A browser reads a backslash in a URL as a path separator and drops
		// tab/CR/LF before resolving it, so each of these reaches the network
		// as the protocol-relative "//evil.example.com" the case above
		// rejects.
		{"/\\evil.example.com/x.svg", true},
		{"/\t/evil.example.com/x.svg", true},
		{"/\n/evil.example.com/x.svg", true},
		{"https://cdn.example.com\\@evil.example.com/x.svg", true},
	}
	for _, tc := range cases {
		t.Run(tc.icon, func(t *testing.T) {
			baseEnv(t)
			t.Setenv("OIDC_ENABLED", "true")
			t.Setenv("OIDC_ISSUER", "https://sso.example.com")
			t.Setenv("OIDC_CLIENT_ID", "client")
			t.Setenv("OIDC_CLIENT_SECRET", "secret")
			t.Setenv("OIDC_PROVIDER_ICON", tc.icon)

			cfg, err := config.Load()
			if tc.wantErr {
				require.ErrorIs(t, err, config.ErrOIDCIconInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.icon, cfg.OIDC.ProviderIcon)
		})
	}
}

// openid is what makes the request an OpenID Connect request; without it the
// provider returns no ID token and every login fails deep inside the token
// exchange. An operator who overrides OIDC_SCOPES must not be able to drop it.
func TestLoad_OIDCAlwaysRequestsTheOpenIDScope(t *testing.T) {
	baseEnv(t)
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://sso.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client")
	t.Setenv("OIDC_CLIENT_SECRET", "secret")
	t.Setenv("OIDC_SCOPES", "profile email")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, []string{"openid", "profile", "email"}, cfg.OIDC.Scopes)
}

// ...and it must not be requested twice when the operator does include it.
func TestLoad_OIDCOpenIDScopeIsNotDuplicated(t *testing.T) {
	baseEnv(t)
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER", "https://sso.example.com")
	t.Setenv("OIDC_CLIENT_ID", "client")
	t.Setenv("OIDC_CLIENT_SECRET", "secret")
	t.Setenv("OIDC_SCOPES", "email openid profile")
	t.Setenv("OIDC_EXTRA_SCOPES", "openid urn:zitadel:iam:org:idp:id:42")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, []string{"openid", "email", "profile", "urn:zitadel:iam:org:idp:id:42"}, cfg.OIDC.Scopes)
}
