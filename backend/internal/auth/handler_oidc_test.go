package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// ─── fake identity provider ──────────────────────────────────────────────────

// fakeIDP is a minimal OpenID Provider: discovery document, JWKS and a token
// endpoint that mints a signed ID token. Driving the real go-oidc client
// against it exercises signature, issuer, audience and nonce verification for
// real, which a stubbed-out client would quietly skip.
type fakeIDP struct {
	server    *httptest.Server
	key       *rsa.PrivateKey
	clientID  string
	tokenHits atomic.Int32

	// Knobs individual tests flip before driving the flow.
	subject       string
	email         string
	emailVerified any
	nonceOverride *string
	issuerForJWT  string
}

func newFakeIDP(t *testing.T) *fakeIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIDP{
		key:           key,
		clientID:      "team-manager-test",
		subject:       "sub-123",
		email:         "user@example.com",
		emailVerified: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                                idp.issuer(),
			"authorization_endpoint":                idp.issuer() + "/authorize",
			"token_endpoint":                        idp.issuer() + "/token",
			"jwks_uri":                              idp.issuer() + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		pub := key.Public().(*rsa.PublicKey)
		writeJSON(w, map[string]any{"keys": []map[string]any{{
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"kid": "test-key",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		idp.tokenHits.Add(1)
		// A real provider looks the nonce up in the session it created at the
		// authorization step; this fake carries it over in lastNonce instead.
		nonce, _ := lastNonce.Load().(string)
		if idp.nonceOverride != nil {
			nonce = *idp.nonceOverride
		}
		writeJSON(w, map[string]any{
			"access_token": "access-token",
			"token_type":   "Bearer",
			"id_token":     idp.signIDToken(t, nonce),
		})
	})

	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	return idp
}

func (f *fakeIDP) issuer() string {
	if f.server == nil {
		// Called while building the discovery document handler before the
		// server exists is impossible (handlers run after Start), but the
		// struct literal above references it, so keep this total.
		return ""
	}
	return f.server.URL
}

func (f *fakeIDP) signIDToken(t *testing.T, nonce string) string {
	t.Helper()
	iss := f.issuer()
	if f.issuerForJWT != "" {
		iss = f.issuerForJWT
	}
	claims := jwt.MapClaims{
		"iss":            iss,
		"aud":            f.clientID,
		"sub":            f.subject,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"iat":            time.Now().Unix(),
		"nonce":          nonce,
		"email":          f.email,
		"email_verified": f.emailVerified,
		"name":           "Test User",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	signed, err := token.SignedString(f.key)
	require.NoError(t, err)
	return signed
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// lastNonce carries the nonce from the authorization request to the fake token
// endpoint, which (unlike a real provider) has no session to look it up in.
var lastNonce atomic.Value

func init() { lastNonce.Store("") }

// ─── harness ─────────────────────────────────────────────────────────────────

type oidcHarness struct {
	handler    *auth.Handler
	repo       *regTestRepo
	idp        *fakeIDP
	stateCodec *auth.SessionCookieCodec
}

func newOIDCHarness(t *testing.T) *oidcHarness {
	t.Helper()
	idp := newFakeIDP(t)
	repo := newRegTestRepo()
	svc := newRegTestService(t, repo, time.Hour, true)

	stateCodec, err := auth.NewOIDCStateCodec([][]byte{make([]byte, 32)}, false)
	require.NoError(t, err)

	client := auth.NewOIDCClient(auth.OIDCConfig{
		Issuer:           idp.issuer(),
		ClientID:         idp.clientID,
		ClientSecret:     "secret",
		RedirectURL:      "https://app.example.com/api/v1/auth/oidc/callback",
		Scopes:           []string{"openid", "profile", "email"},
		ProviderID:       testOIDCProvider,
		ProviderName:     "Google",
		ProviderSubtitle: "Mit Google anmelden",
		ProviderIcon:     "/provider-icons/google.svg",
		PostLoginURL:     "https://app.example.com",
	}, slog.Default())

	h := auth.NewHandler(svc, slog.Default(), nil, nil)
	h.SetOIDC(client, stateCodec)
	return &oidcHarness{handler: h, repo: repo, idp: idp, stateCodec: stateCodec}
}

// start drives StartOidcLogin through the strict middleware so the state
// cookie is actually emitted, and returns the redirect target plus that cookie.
func (h *oidcHarness) start(t *testing.T, codec *auth.SessionCookieCodec, returnTo ...string) (location string, stateCookie *http.Cookie) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/oidc/start", http.NoBody)

	var params gen.StartOidcLoginParams
	if len(returnTo) > 0 {
		params.ReturnTo = &returnTo[0]
	}
	mw := codec.StrictMiddleware()
	wrapped := mw(func(ctx context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
		return h.handler.StartOidcLogin(ctx, gen.StartOidcLoginRequestObject{Params: params})
	}, "StartOidcLogin")

	resp, err := wrapped(req.Context(), rec, req, nil)
	require.NoError(t, err)

	switch typed := resp.(type) {
	case gen.StartOidcLogin302Response:
		require.NotNil(t, typed.Headers.Location)
		location = *typed.Headers.Location
	default:
		t.Fatalf("unexpected start response %T", resp)
	}

	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.OIDCStateCookieName {
			stateCookie = c
		}
	}
	return location, stateCookie
}

// callback drives OidcCallback through both the state middleware (which lifts
// the cookie into the context) and the strict middleware (which sets the
// session cookie), returning the redirect target and the response recorder.
func (h *oidcHarness) callback(t *testing.T, codec *auth.SessionCookieCodec, query string, cookies ...*http.Cookie) (string, *httptest.ResponseRecorder) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/oidc/callback?"+query, http.NoBody)
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}

	var location string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mw := codec.StrictMiddleware()
		wrapped := mw(func(ctx context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
			return h.handler.OidcCallback(ctx, gen.OidcCallbackRequestObject{Params: parseCallbackParams(r)})
		}, "OidcCallback")
		resp, err := wrapped(r.Context(), w, r, nil)
		require.NoError(t, err)
		typed, ok := resp.(gen.OidcCallback302Response)
		require.True(t, ok, "unexpected callback response %T", resp)
		require.NotNil(t, typed.Headers.Location)
		location = *typed.Headers.Location
	})

	h.handler.OIDCStateMiddleware(inner).ServeHTTP(rec, req)
	return location, rec
}

func parseCallbackParams(r *http.Request) gen.OidcCallbackParams {
	q := r.URL.Query()
	optional := func(key string) *string {
		if !q.Has(key) {
			return nil
		}
		v := q.Get(key)
		return &v
	}
	return gen.OidcCallbackParams{
		Code:             optional("code"),
		State:            optional("state"),
		Error:            optional("error"),
		ErrorDescription: optional("error_description"),
	}
}

func sessionCodecForTest(t *testing.T) *auth.SessionCookieCodec {
	t.Helper()
	codec, err := auth.NewSessionCookieCodec([][]byte{make([]byte, 32)}, false, time.Hour, "")
	require.NoError(t, err)
	return codec
}

// runFlow performs a full start → callback round trip and returns the final
// redirect plus the callback's recorder.
func (h *oidcHarness) runFlow(t *testing.T, returnTo ...string) (string, *httptest.ResponseRecorder) {
	t.Helper()
	codec := sessionCodecForTest(t)
	location, stateCookie := h.start(t, codec, returnTo...)
	require.NotNil(t, stateCookie)

	parsed, err := url.Parse(location)
	require.NoError(t, err)
	lastNonce.Store(parsed.Query().Get("nonce"))

	return h.callback(t, codec,
		"code=auth-code&state="+url.QueryEscape(parsed.Query().Get("state")), stateCookie)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestStartOidcLogin_RedirectsWithPKCEAndNoPrompt(t *testing.T) {
	h := newOIDCHarness(t)
	location, stateCookie := h.start(t, sessionCodecForTest(t))

	parsed, err := url.Parse(location)
	require.NoError(t, err)
	q := parsed.Query()

	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.NotEmpty(t, q.Get("code_challenge"))
	assert.NotEmpty(t, q.Get("state"))
	assert.NotEmpty(t, q.Get("nonce"))
	assert.Equal(t, "openid profile email", q.Get("scope"))
	// A prompt would make a broker render its own login UI instead of
	// forwarding to the upstream provider -- see the design note.
	assert.Empty(t, q.Get("prompt"), "no prompt parameter may be sent")

	require.NotNil(t, stateCookie)
	assert.True(t, stateCookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, stateCookie.SameSite)
	assert.Equal(t, auth.OIDCStateCookiePath, stateCookie.Path)
	assert.NotContains(t, stateCookie.Value, q.Get("state"), "the cookie must be encrypted, not plaintext state")
}

func TestStartOidcLogin_404WhenDisabled(t *testing.T) {
	h := auth.NewHandler(&mockAuthService{}, slog.Default(), nil, nil)

	resp, err := h.StartOidcLogin(context.Background(), gen.StartOidcLoginRequestObject{})
	require.NoError(t, err)
	assert.IsType(t, gen.StartOidcLogin404ApplicationProblemPlusJSONResponse{}, resp)
}

func TestOidcCallback_SuccessEstablishesSession(t *testing.T) {
	h := newOIDCHarness(t)
	location, rec := h.runFlow(t)

	assert.Equal(t, "https://app.example.com/", location)
	assert.NotContains(t, location, "login_error")

	var session, state *http.Cookie
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case auth.DefaultSessionCookieName:
			session = c
		case auth.OIDCStateCookieName:
			state = c
		}
	}
	require.NotNil(t, session, "a successful callback must set the session cookie")
	assert.True(t, session.HttpOnly)
	assert.NotEmpty(t, session.Value)
	require.NotNil(t, state)
	assert.Equal(t, -1, state.MaxAge, "the spent state cookie must be expired")

	user, err := h.repo.FindUserByEmail(context.Background(), "user@example.com")
	require.NoError(t, err)
	assert.Empty(t, user.PasswordHash)
	assert.NotNil(t, user.EmailVerifiedAt)
}

// The whole point of checking state first: an unsolicited callback must never
// make this server call out to the provider.
func TestOidcCallback_StateMismatchDoesNotReachTheProvider(t *testing.T) {
	h := newOIDCHarness(t)
	codec := sessionCodecForTest(t)
	_, stateCookie := h.start(t, codec)
	require.NotNil(t, stateCookie)

	before := h.idp.tokenHits.Load()
	location, rec := h.callback(t, codec, "code=auth-code&state=not-the-right-state", stateCookie)

	assert.Equal(t, before, h.idp.tokenHits.Load(), "no token exchange may happen")
	assert.Contains(t, location, "login_error=oidc_failed")
	assert.Nil(t, sessionCookie(rec), "no session may be established")
}

func TestOidcCallback_MissingStateCookieDoesNotReachTheProvider(t *testing.T) {
	h := newOIDCHarness(t)
	codec := sessionCodecForTest(t)
	location, _ := h.start(t, codec)
	parsed, err := url.Parse(location)
	require.NoError(t, err)

	before := h.idp.tokenHits.Load()
	redirect, rec := h.callback(t, codec, "code=auth-code&state="+url.QueryEscape(parsed.Query().Get("state")))

	assert.Equal(t, before, h.idp.tokenHits.Load())
	assert.Contains(t, redirect, "login_error=oidc_failed")
	assert.Nil(t, sessionCookie(rec))
}

func TestOidcCallback_ProviderErrorIsReportedWithoutExchange(t *testing.T) {
	h := newOIDCHarness(t)
	codec := sessionCodecForTest(t)
	_, stateCookie := h.start(t, codec)

	before := h.idp.tokenHits.Load()
	location, rec := h.callback(t, codec, "error=access_denied", stateCookie)

	assert.Equal(t, before, h.idp.tokenHits.Load())
	assert.Contains(t, location, "login_error=oidc_denied")
	assert.Nil(t, sessionCookie(rec))
}

func TestOidcCallback_MissingCodeIsRejected(t *testing.T) {
	h := newOIDCHarness(t)
	codec := sessionCodecForTest(t)
	location, stateCookie := h.start(t, codec)
	parsed, err := url.Parse(location)
	require.NoError(t, err)

	before := h.idp.tokenHits.Load()
	redirect, rec := h.callback(t, codec,
		"state="+url.QueryEscape(parsed.Query().Get("state")), stateCookie)

	assert.Equal(t, before, h.idp.tokenHits.Load())
	assert.Contains(t, redirect, "login_error=oidc_failed")
	assert.Nil(t, sessionCookie(rec))
}

func TestOidcCallback_UnverifiedEmailIsRejected(t *testing.T) {
	h := newOIDCHarness(t)
	h.idp.emailVerified = false

	location, rec := h.runFlow(t)

	assert.Contains(t, location, "login_error=oidc_email_unverified")
	assert.Nil(t, sessionCookie(rec))

	_, err := h.repo.FindUserByEmail(context.Background(), "user@example.com")
	assert.Error(t, err, "no account may be provisioned from an unverified assertion")
}

// A provider that reports email_verified as the string "true" is still
// asserting a verified address; one that reports "false" is not.
func TestOidcCallback_StringEmailVerifiedIsAccepted(t *testing.T) {
	h := newOIDCHarness(t)
	h.idp.emailVerified = "true"

	location, rec := h.runFlow(t)

	assert.NotContains(t, location, "login_error")
	assert.NotNil(t, sessionCookie(rec))
}

func TestOidcCallback_NonceMismatchIsRejected(t *testing.T) {
	h := newOIDCHarness(t)
	wrong := "not-the-nonce-we-sent"
	h.idp.nonceOverride = &wrong

	location, rec := h.runFlow(t)

	assert.Contains(t, location, "login_error=oidc_unavailable")
	assert.Nil(t, sessionCookie(rec))
}

func TestOidcCallback_WrongIssuerIsRejected(t *testing.T) {
	h := newOIDCHarness(t)
	h.idp.issuerForJWT = "https://attacker.example.com"

	location, rec := h.runFlow(t)

	assert.Contains(t, location, "login_error=oidc_unavailable")
	assert.Nil(t, sessionCookie(rec))
}

func TestOidcCallback_ReplayedStateCookieIsRejected(t *testing.T) {
	h := newOIDCHarness(t)
	codec := sessionCodecForTest(t)
	location, stateCookie := h.start(t, codec)
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	lastNonce.Store(parsed.Query().Get("nonce"))
	query := "code=auth-code&state=" + url.QueryEscape(parsed.Query().Get("state"))

	first, firstRec := h.callback(t, codec, query, stateCookie)
	require.NotContains(t, first, "login_error")
	require.NotNil(t, sessionCookie(firstRec))

	// The browser honors the expiry the first callback sent, so a replay
	// arrives without the cookie -- which is exactly the missing-state case.
	replay, replayRec := h.callback(t, codec, query)
	assert.Contains(t, replay, "login_error=oidc_failed")
	assert.Nil(t, sessionCookie(replayRec))
}

func TestListProviders_IncludesConfiguredOIDCProvider(t *testing.T) {
	h := newOIDCHarness(t)

	resp, err := h.handler.ListProviders(context.Background(), gen.ListProvidersRequestObject{})
	require.NoError(t, err)
	providers, ok := resp.(gen.ListProviders200JSONResponse)
	require.True(t, ok)
	require.Len(t, providers, 2)

	assert.Equal(t, testOIDCProvider, providers[0].Id)
	assert.Equal(t, "Google", providers[0].Name)
	assert.Equal(t, "Mit Google anmelden", providers[0].Sub)
	require.NotNil(t, providers[0].Icon)
	assert.Equal(t, "/provider-icons/google.svg", *providers[0].Icon)

	assert.Equal(t, "password", providers[1].Id, "password login stays available")
	assert.Nil(t, providers[1].Icon)
}

// The login screen draws Glyph as literal text inside a 34x34 badge, so an
// icon *name* like "login" renders as that word, clipped, rather than as a
// symbol. Every provider's fallback glyph has to be a single character.
func TestListProviders_GlyphIsASingleCharacter(t *testing.T) {
	h := newOIDCHarness(t)

	resp, err := h.handler.ListProviders(context.Background(), gen.ListProvidersRequestObject{})
	require.NoError(t, err)
	providers, ok := resp.(gen.ListProviders200JSONResponse)
	require.True(t, ok)

	for _, p := range providers {
		assert.Equal(t, 1, utf8.RuneCountInString(p.Glyph),
			"provider %q: glyph %q is not a single character", p.Id, p.Glyph)
	}
	assert.Equal(t, "G", providers[0].Glyph, "derived from the provider name")
	assert.Equal(t, "E", providers[1].Glyph)
}

func TestListProviders_PasswordOnlyWithoutOIDC(t *testing.T) {
	h := auth.NewHandler(&mockAuthService{}, slog.Default(), nil, nil)

	resp, err := h.ListProviders(context.Background(), gen.ListProvidersRequestObject{})
	require.NoError(t, err)
	providers, ok := resp.(gen.ListProviders200JSONResponse)
	require.True(t, ok)
	require.Len(t, providers, 1)
	assert.Equal(t, "password", providers[0].Id)
}

// sessionCookie returns the session cookie from a recorder, or nil.
func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.DefaultSessionCookieName && c.Value != "" && !strings.EqualFold(c.Value, "deleted") {
			return c
		}
	}
	return nil
}

// An invite link is redeemed by the frontend from the URL it loads on, so a
// login that starts at /join/... has to come back there -- otherwise "sign in
// with Google to join this team" logs the user in and joins nothing.
func TestOidcCallback_ReturnsToTheStartingPath(t *testing.T) {
	h := newOIDCHarness(t)

	location, _ := h.runFlow(t, "/join/team-1/abc123")

	assert.Equal(t, "https://app.example.com/join/team-1/abc123", location)
}

// The return path arrives as a query parameter on an unauthenticated endpoint,
// so it is an open-redirect vector until proven otherwise. Each of these
// resolves to an off-site destination in a browser.
func TestOidcCallback_RejectsAnOffSiteReturnPath(t *testing.T) {
	hostile := []string{
		"https://evil.example.com/",
		"//evil.example.com/",
		"/\\evil.example.com/",
		"/\t/evil.example.com/",
		"evil.example.com",
		"",
	}
	for _, returnTo := range hostile {
		t.Run(returnTo, func(t *testing.T) {
			h := newOIDCHarness(t)

			location, _ := h.runFlow(t, returnTo)

			assert.Equal(t, "https://app.example.com/", location,
				"a return path that is not root-relative must fall back to the app root")
		})
	}
}
