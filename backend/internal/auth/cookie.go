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

// ErrInvalidCookie is returned when a session cookie cannot be decoded or
// authenticated (tampered, truncated, or encrypted with a different key).
var ErrInvalidCookie = errors.New("auth: invalid session cookie")

// SessionCookieCodec encrypts/decrypts the session JWT into an opaque,
// authenticated cookie value using AES-256-GCM and manages the Set-Cookie /
// clear-cookie headers.
type SessionCookieCodec struct {
	gcm    cipher.AEAD
	secure bool
	ttl    time.Duration
	name   string
}

// NewSessionCookieCodec builds a codec from a 32-byte key. secure controls the
// cookie's Secure attribute, ttl its Max-Age. An empty name falls back to
// DefaultSessionCookieName.
func NewSessionCookieCodec(key []byte, secure bool, ttl time.Duration, name string) (*SessionCookieCodec, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("auth.NewSessionCookieCodec: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("auth.NewSessionCookieCodec: %w", err)
	}
	if name == "" {
		name = DefaultSessionCookieName
	}
	return &SessionCookieCodec{gcm: gcm, secure: secure, ttl: ttl, name: name}, nil
}

// Name returns the cookie name.
func (c *SessionCookieCodec) Name() string { return c.name }

// Encrypt seals the JWT with AES-256-GCM and returns base64url(nonce||ciphertext).
func (c *SessionCookieCodec) Encrypt(jwt string) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("auth.SessionCookieCodec.Encrypt: %w", err)
	}
	sealed := c.gcm.Seal(nonce, nonce, []byte(jwt), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Any decoding or authentication failure yields
// ErrInvalidCookie.
func (c *SessionCookieCodec) Decrypt(value string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", ErrInvalidCookie
	}
	ns := c.gcm.NonceSize()
	if len(raw) < ns {
		return "", ErrInvalidCookie
	}
	nonce, ciphertext := raw[:ns], raw[ns:]
	plain, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidCookie
	}
	return string(plain), nil
}

// Set writes the encrypted session cookie onto the response.
func (c *SessionCookieCodec) Set(w http.ResponseWriter, jwt string) error {
	value, err := c.Encrypt(jwt)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(c.ttl.Seconds()),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// Clear overwrites the session cookie with an expired, empty value.
func (c *SessionCookieCodec) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// StrictMiddleware returns a generated-strict-handler middleware that sets the
// session cookie on a successful Login and clears it on Logout. It runs before
// the response is visited, so the Set-Cookie header is emitted with the body.
func (c *SessionCookieCodec) StrictMiddleware() gen.StrictMiddlewareFunc {
	return func(f gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			resp, err := f(ctx, w, r, request)
			if err != nil {
				return resp, err
			}
			switch operationID {
			case "Login":
				if login, ok := resp.(gen.Login200JSONResponse); ok {
					if setErr := c.Set(w, login.Token); setErr != nil {
						return resp, setErr
					}
				}
			case "Logout":
				if _, ok := resp.(gen.Logout204Response); ok {
					c.Clear(w)
				}
			}
			return resp, nil
		}
	}
}
