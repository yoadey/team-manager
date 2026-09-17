package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

// DefaultSessionCookieName is the cookie name used when none is configured.
const DefaultSessionCookieName = "tv_session"

// OIDCStateCookieName is the name of the short-lived cookie that carries the
// OIDC state/nonce/PKCE verifier between the start endpoint and the callback.
const OIDCStateCookieName = "tv_oidc"

// OIDCStateCookiePath scopes the state cookie to the two OIDC endpoints, so it
// isn't sent along with every other request for the ten minutes it lives.
// It mirrors the base path cmd/server/main.go mounts the generated routes
// under ("/api/v1") plus the OIDC route prefix.
const OIDCStateCookiePath = "/api/v1/auth/oidc"

// OIDCStateTTL bounds how long a started login may take to come back. Long
// enough for a user to actually authenticate at the provider (including a
// password manager detour), short enough that an abandoned attempt doesn't
// linger.
const OIDCStateTTL = 10 * time.Minute

// ErrInvalidCookie is returned when a session cookie cannot be decoded or
// authenticated (tampered, truncated, or encrypted with a different key).
var ErrInvalidCookie = errors.New("auth: invalid session cookie")

// ErrNoKeys is returned when NewSessionCookieCodec is called with an empty
// key slice.
var ErrNoKeys = errors.New("auth.NewSessionCookieCodec: at least one key is required")

// SessionCookieCodec encrypts/decrypts the session JWT into an opaque,
// authenticated cookie value using AES-256-GCM and manages the Set-Cookie /
// clear-cookie headers.
//
// Multiple keys are supported for zero-downtime rotation: gcms[0] is always
// used for encryption; all keys are tried for decryption so that cookies
// encrypted with an older key remain valid after a rotation.
type SessionCookieCodec struct {
	gcms   []cipher.AEAD // gcms[0] is the active key; older keys follow for decryption only
	secure bool
	ttl    time.Duration
	name   string
	path   string
}

// NewSessionCookieCodec builds a codec from one or more 32-byte keys. keys[0]
// is the active encryption key; subsequent keys are only used for decryption,
// enabling zero-downtime rotation. At least one key is required.
//
// secure controls the cookie's Secure attribute; ttl its Max-Age. An empty
// name falls back to DefaultSessionCookieName.
func NewSessionCookieCodec(keys [][]byte, secure bool, ttl time.Duration, name string) (*SessionCookieCodec, error) {
	if name == "" {
		name = DefaultSessionCookieName
	}
	return newCookieCodec(keys, secure, ttl, name, "/")
}

// NewOIDCStateCodec builds the codec for the short-lived OIDC state cookie. It
// deliberately reuses the session cookie's keys (COOKIE_ENCRYPTION_KEYS): the
// state cookie protects a login that is in flight, so it needs exactly the same
// confidentiality and rotation story, and giving it a second key to configure
// would be one more thing to get wrong for no gain.
func NewOIDCStateCodec(keys [][]byte, secure bool) (*SessionCookieCodec, error) {
	return newCookieCodec(keys, secure, OIDCStateTTL, OIDCStateCookieName, OIDCStateCookiePath)
}

func newCookieCodec(keys [][]byte, secure bool, ttl time.Duration, name, path string) (*SessionCookieCodec, error) {
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}
	gcms := make([]cipher.AEAD, len(keys))
	for i, key := range keys {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("auth.newCookieCodec: key[%d]: %w", i, err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("auth.newCookieCodec: key[%d]: %w", i, err)
		}
		gcms[i] = gcm
	}
	return &SessionCookieCodec{gcms: gcms, secure: secure, ttl: ttl, name: name, path: path}, nil
}

// Name returns the cookie name.
func (c *SessionCookieCodec) Name() string { return c.name }

// Encrypt seals the JWT with AES-256-GCM using the active (first) key and
// returns base64url(nonce||ciphertext).
func (c *SessionCookieCodec) Encrypt(jwt string) (string, error) {
	gcm := c.gcms[0]
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("auth.SessionCookieCodec.Encrypt: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(jwt), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt, trying each key in order. Any decoding or
// authentication failure with all keys yields ErrInvalidCookie.
func (c *SessionCookieCodec) Decrypt(value string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", ErrInvalidCookie
	}
	for _, gcm := range c.gcms {
		ns := gcm.NonceSize()
		if len(raw) < ns {
			continue
		}
		nonce, ciphertext := raw[:ns], raw[ns:]
		plain, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			continue
		}
		return string(plain), nil
	}
	return "", ErrInvalidCookie
}

// Set writes the encrypted cookie onto the response.
func (c *SessionCookieCodec) Set(w http.ResponseWriter, jwt string) error {
	value, err := c.Encrypt(jwt)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    value,
		Path:     c.path,
		MaxAge:   int(c.ttl.Seconds()),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// Clear overwrites the cookie with an expired, empty value.
func (c *SessionCookieCodec) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    "",
		Path:     c.path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// sessionTokenContextKey is the context key under which StrictMiddleware
// installs a *string "holder" that Login/VerifyEmail/ResetPassword fill in
// via SetSessionToken. The signed session JWT deliberately never appears in
// those handlers' JSON response bodies (only in the httpOnly cookie) -- the
// holder is how the token still reaches applyCookie after f returns.
type sessionTokenContextKey struct{}

// SetSessionToken records the session JWT that StrictMiddleware should set as
// the httpOnly session cookie for the current request. Call this from a
// strict handler that establishes a new session (Login, VerifyEmail,
// ResetPassword) instead of returning the token in the JSON response body --
// returning it in the body would defeat the point of an httpOnly cookie. A
// no-op if called outside a request wrapped by StrictMiddleware.
func SetSessionToken(ctx context.Context, token string) {
	if holder, ok := ctx.Value(sessionTokenContextKey{}).(*string); ok {
		*holder = token
	}
}

// extraCookiesContextKey is the context key under which StrictMiddleware
// installs a *[]*http.Cookie that handlers append to via AddResponseCookie.
// Same motivation as sessionTokenContextKey: a strict handler receives no
// http.ResponseWriter, so anything it wants in a header has to travel out
// through the context.
type extraCookiesContextKey struct{}

// AddResponseCookie records a cookie StrictMiddleware should emit alongside
// the response, whatever that response turns out to be. Used by the OIDC
// handlers for the short-lived state cookie, which -- unlike the session
// cookie -- is set and cleared on redirects rather than on a specific JSON
// result, so applyCookie's operation/response switch is the wrong place for
// it. A no-op outside a request wrapped by StrictMiddleware.
func AddResponseCookie(ctx context.Context, cookie *http.Cookie) {
	if holder, ok := ctx.Value(extraCookiesContextKey{}).(*[]*http.Cookie); ok {
		*holder = append(*holder, cookie)
	}
}

// StateCookie builds the Set-Cookie carrying an encrypted OIDC state payload.
func (c *SessionCookieCodec) StateCookie(payload string) (*http.Cookie, error) {
	value, err := c.Encrypt(payload)
	if err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name:     c.name,
		Value:    value,
		Path:     c.path,
		MaxAge:   int(c.ttl.Seconds()),
		HttpOnly: true,
		Secure:   c.secure,
		// Lax, not Strict: the browser arrives back from the identity
		// provider through a top-level cross-site GET navigation. Strict
		// would withhold the cookie on exactly that request and break every
		// login.
		SameSite: http.SameSiteLaxMode,
	}, nil
}

// ClearedStateCookie builds the Set-Cookie that expires the state cookie once
// a login attempt has been consumed (successfully or not), so a replayed
// callback finds nothing to validate against.
func (c *SessionCookieCodec) ClearedStateCookie() *http.Cookie {
	return &http.Cookie{
		Name:     c.name,
		Value:    "",
		Path:     c.path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// StrictMiddleware returns a generated-strict-handler middleware that sets the
// session cookie on a successful Login/VerifyEmail/ResetPassword/OidcCallback
// (via the token the handler passes to SetSessionToken) and clears it on
// Logout / account erasure. It also emits any cookie a handler registered via
// AddResponseCookie. It runs before the response is visited, so the Set-Cookie
// headers are emitted with the body.
func (c *SessionCookieCodec) StrictMiddleware() gen.StrictMiddlewareFunc {
	return func(f gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			var token string
			var extra []*http.Cookie
			ctx = context.WithValue(ctx, sessionTokenContextKey{}, &token)
			ctx = context.WithValue(ctx, extraCookiesContextKey{}, &extra)
			resp, err := f(ctx, w, r, request)
			if err != nil {
				return resp, err
			}
			for _, cookie := range extra {
				// The codec is the single choke point every extra cookie
				// passes through, so it stamps the transport-security
				// attributes here rather than trusting each handler to have
				// got them right. Secure tracks COOKIE_SECURE, exactly like
				// the session cookie (Set/Clear above).
				cookie.Secure = c.secure
				cookie.HttpOnly = true
				http.SetCookie(w, cookie)
			}
			if cookieErr := c.applyCookie(w, operationID, resp, token); cookieErr != nil {
				return resp, cookieErr
			}
			return resp, nil
		}
	}
}

// applyCookie sets the session cookie after a successful Login, VerifyEmail,
// ResetPassword or OidcCallback (using the token the handler recorded via
// SetSessionToken) and clears it after a successful Logout or account
// erasure, based on the operation result.
func (c *SessionCookieCodec) applyCookie(w http.ResponseWriter, operationID string, resp any, token string) error {
	switch operationID {
	case "Login":
		if _, ok := resp.(gen.Login200JSONResponse); ok {
			return c.Set(w, token)
		}
	case "VerifyEmail":
		if _, ok := resp.(gen.VerifyEmail200JSONResponse); ok {
			return c.Set(w, token)
		}
	case "ResetPassword":
		if _, ok := resp.(gen.ResetPassword200JSONResponse); ok {
			return c.Set(w, token)
		}
	case "OidcCallback":
		// The callback redirects on both success and failure, so unlike the
		// cases above the response type alone doesn't say whether a session
		// was established -- a token recorded via SetSessionToken does.
		if _, ok := resp.(gen.OidcCallback302Response); ok && token != "" {
			return c.Set(w, token)
		}
	case "Logout":
		if _, ok := resp.(gen.Logout204Response); ok {
			c.Clear(w)
		}
	case "DeleteCurrentUser":
		if _, ok := resp.(gen.DeleteCurrentUser204Response); ok {
			c.Clear(w)
		}
	}
	return nil
}
