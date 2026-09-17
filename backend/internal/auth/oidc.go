package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Errors surfaced by the OIDC login flow. They exist as distinct values
// because each maps to a different message on the login screen -- see
// Handler.OidcCallback's loginErrorCode.
var (
	// ErrOIDCDisabled is returned when an OIDC endpoint is reached on a
	// deployment that has no provider configured.
	ErrOIDCDisabled = errors.New("auth: no OIDC provider configured")

	// ErrOIDCState is returned when the callback's state does not match the
	// state cookie, the cookie is missing, or it has expired. Deliberately one
	// error for all three: from the caller's side they are the same "start
	// over" situation, and distinguishing them tells an attacker which half of
	// the check they got past.
	ErrOIDCState = errors.New("auth: OIDC state mismatch")

	// ErrOIDCEmailUnverified is returned when the provider did not assert a
	// verified email address. Linking accounts by email address is only safe if
	// the provider actually vouches for the address, so this is fatal to the
	// login rather than a downgrade to "create an unverified account".
	ErrOIDCEmailUnverified = errors.New("auth: OIDC provider did not assert a verified email address")

	// ErrOIDCAccountDeleted is returned when the verified address still belongs
	// to a soft-deleted account.
	ErrOIDCAccountDeleted = errors.New("auth: account for this address was deleted")

	// ErrOIDCNoIDToken is returned when the provider's token response carried
	// no id_token, which means it is not actually speaking OIDC to us.
	ErrOIDCNoIDToken = errors.New("auth: token response carried no id_token")
)

// OIDCConfig is the resolved provider configuration. Scopes is the final list
// sent in the authorization request (base scopes plus any extra scopes),
// already merged by the config layer.
type OIDCConfig struct {
	Issuer           string
	ClientID         string
	ClientSecret     string
	RedirectURL      string
	Scopes           []string
	ProviderID       string
	ProviderName     string
	ProviderSubtitle string
	ProviderIcon     string
	// PostLoginURL is where the callback sends the browser once it is done,
	// success or failure -- this deployment's frontend origin.
	PostLoginURL string
}

// OIDCClaims is the subset of the ID token this application acts on.
type OIDCClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// OIDCClient wraps provider discovery, the authorization request and ID-token
// verification for a single configured provider.
//
// Discovery is lazy and memoized rather than done once at construction: the
// identity provider is a separate system that may be down or not yet reachable
// when this process starts, and refusing to boot in that case would take
// password login -- the very fallback that situation calls for -- down with it.
type OIDCClient struct {
	cfg        OIDCConfig
	logger     *slog.Logger
	httpClient *http.Client

	mu       sync.Mutex
	oauthCfg *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewOIDCClient builds a client for cfg. It performs no network I/O; call
// Warm to attempt discovery eagerly.
func NewOIDCClient(cfg OIDCConfig, logger *slog.Logger) *OIDCClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &OIDCClient{
		cfg:    cfg,
		logger: logger,
		// A bounded client rather than http.DefaultClient: discovery, JWKS and
		// the token exchange all sit in the request path of a user's login, and
		// an unresponsive provider must fail rather than pile up goroutines.
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Config returns the provider configuration (display metadata included).
func (c *OIDCClient) Config() OIDCConfig { return c.cfg }

// Warm attempts discovery once, logging rather than returning a failure. Used
// at startup so the first real login doesn't pay the discovery round trip, and
// so a misconfigured issuer shows up in the logs immediately instead of at the
// first login attempt.
func (c *OIDCClient) Warm(ctx context.Context) {
	if _, _, err := c.resolve(ctx); err != nil {
		c.logger.Warn("OIDC discovery failed; will retry on first login",
			slog.String("issuer", c.cfg.Issuer), slog.String("error", err.Error()))
	}
}

// resolve returns the oauth2 config and ID-token verifier, running discovery
// on first use and caching the result.
func (c *OIDCClient) resolve(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.oauthCfg != nil && c.verifier != nil {
		return c.oauthCfg, c.verifier, nil
	}

	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, c.httpClient), c.cfg.Issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.OIDCClient: discovery for %q: %w", c.cfg.Issuer, err)
	}

	c.oauthCfg = &oauth2.Config{
		ClientID:     c.cfg.ClientID,
		ClientSecret: c.cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.cfg.RedirectURL,
		Scopes:       c.cfg.Scopes,
	}
	c.verifier = provider.Verifier(&oidc.Config{ClientID: c.cfg.ClientID})
	return c.oauthCfg, c.verifier, nil
}

// AuthCodeURL builds the authorization request for a fresh login attempt.
//
// No prompt parameter is sent, deliberately. Brokers commonly treat any prompt
// other than select_account as "show me your own login UI", which would defeat
// a deployment that configured an extra scope precisely to skip that UI and go
// straight to an upstream identity provider.
func (c *OIDCClient) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	oauthCfg, _, err := c.resolve(ctx)
	if err != nil {
		return "", err
	}
	return oauthCfg.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// Exchange trades the authorization code for tokens and verifies the returned
// ID token's signature, issuer, audience, expiry and nonce.
func (c *OIDCClient) Exchange(ctx context.Context, code, verifier, nonce string) (*OIDCClaims, error) {
	oauthCfg, idVerifier, err := c.resolve(ctx)
	if err != nil {
		return nil, err
	}

	ctx = oidc.ClientContext(ctx, c.httpClient)
	token, err := oauthCfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("auth.OIDCClient.Exchange: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("auth.OIDCClient.Exchange: %w", ErrOIDCNoIDToken)
	}

	idToken, err := idVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("auth.OIDCClient.Exchange: verify id_token: %w", err)
	}
	if idToken.Nonce != nonce {
		return nil, fmt.Errorf("auth.OIDCClient.Exchange: %w", ErrOIDCState)
	}

	var claims struct {
		Email         string   `json:"email"`
		EmailVerified flexBool `json:"email_verified"`
		Name          string   `json:"name"`
		GivenName     string   `json:"given_name"`
		FamilyName    string   `json:"family_name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("auth.OIDCClient.Exchange: decode claims: %w", err)
	}

	return &OIDCClaims{
		Subject:       idToken.Subject,
		Email:         strings.TrimSpace(claims.Email),
		EmailVerified: bool(claims.EmailVerified),
		Name:          displayName(claims.Name, claims.GivenName, claims.FamilyName, claims.Email),
	}, nil
}

// displayName picks the best available name for a newly provisioned account,
// falling back to the local part of the email address so an account is never
// created nameless (users.name is NOT NULL).
func displayName(name, given, family, email string) string {
	if n := strings.TrimSpace(name); n != "" {
		return n
	}
	if n := strings.TrimSpace(given + " " + family); n != "" {
		return n
	}
	if local, _, found := strings.Cut(email, "@"); found && local != "" {
		return local
	}
	return "Unbenannt"
}

// flexBool decodes a JSON boolean that some providers send as a string
// ("true"/"false") instead. Anything else decodes as false, which for
// email_verified is the safe direction.
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		*b = flexBool(asBool)
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*b = flexBool(asString == "true")
		return nil
	}
	*b = false
	return nil
}

// oidcState is the payload of the tv_oidc cookie: everything the callback
// needs to validate the response to a login it started itself.
type oidcState struct {
	State     string `json:"s"`
	Nonce     string `json:"n"`
	Verifier  string `json:"v"`
	IssuedAt  int64  `json:"t"`
	ReturnURL string `json:"r,omitempty"`
}

// newOIDCState mints the per-attempt secrets: an unguessable state (CSRF), a
// nonce (ID-token replay) and a PKCE verifier (authorization-code
// interception).
func newOIDCState() (*oidcState, error) {
	state, err := randomToken()
	if err != nil {
		return nil, err
	}
	nonce, err := randomToken()
	if err != nil {
		return nil, err
	}
	return &oidcState{
		State:    state,
		Nonce:    nonce,
		Verifier: oauth2.GenerateVerifier(),
		IssuedAt: time.Now().Unix(),
	}, nil
}

// encode serializes the state for the cookie.
func (s *oidcState) encode() (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("auth.oidcState.encode: %w", err)
	}
	return string(raw), nil
}

// decodeOIDCState parses a state cookie payload and enforces its age. The
// cookie's own Max-Age is a client-side hint only -- a client is free to keep
// sending an expired cookie -- so the deadline is re-checked here.
func decodeOIDCState(payload string) (*oidcState, error) {
	var s oidcState
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		return nil, ErrOIDCState
	}
	if s.State == "" || s.Nonce == "" || s.Verifier == "" {
		return nil, ErrOIDCState
	}
	if time.Since(time.Unix(s.IssuedAt, 0)) > OIDCStateTTL {
		return nil, ErrOIDCState
	}
	return &s, nil
}

// randomToken returns 32 bytes of URL-safe randomness.
func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth.randomToken: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
