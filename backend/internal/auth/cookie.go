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
//
// Multiple keys are supported for zero-downtime rotation: gcms[0] is always
// used for encryption; all keys are tried for decryption so that cookies
// encrypted with an older key remain valid after a rotation.
type SessionCookieCodec struct {
	gcms   []cipher.AEAD // gcms[0] is the active key; older keys follow for decryption only
	secure bool
	ttl    time.Duration
	name   string
}

// NewSessionCookieCodec builds a codec from one or more 32-byte keys. keys[0]
// is the active encryption key; subsequent keys are only used for decryption,
// enabling zero-downtime rotation. At least one key is required.
//
// secure controls the cookie's Secure attribute; ttl its Max-Age. An empty
// name falls back to DefaultSessionCookieName.
func NewSessionCookieCodec(keys [][]byte, secure bool, ttl time.Duration, name string) (*SessionCookieCodec, error) {
	if len(keys) == 0 {
		return nil, errors.New("auth.NewSessionCookieCodec: at least one key is required")
	}
	gcms := make([]cipher.AEAD, len(keys))
	for i, key := range keys {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("auth.NewSessionCookieCodec: key[%d]: %w", i, err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("auth.NewSessionCookieCodec: key[%d]: %w", i, err)
		}
		gcms[i] = gcm
	}
	if name == "" {
		name = DefaultSessionCookieName
	}
	return &SessionCookieCodec{gcms: gcms, secure: secure, ttl: ttl, name: name}, nil
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
			if cookieErr := c.applyCookie(w, operationID, resp); cookieErr != nil {
				return resp, cookieErr
			}
			return resp, nil
		}
	}
}

// applyCookie sets the session cookie after a successful Login and clears it
// after a successful Logout or account erasure, based on the operation result.
func (c *SessionCookieCodec) applyCookie(w http.ResponseWriter, operationID string, resp any) error {
	switch operationID {
	case "Login":
		if login, ok := resp.(gen.Login200JSONResponse); ok {
			return c.Set(w, login.Token)
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
