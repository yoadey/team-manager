package auth_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

func newCodec(t *testing.T) *auth.SessionCookieCodec {
	t.Helper()
	codec, err := auth.NewSessionCookieCodec([][]byte{make([]byte, 32)}, false, time.Hour, "")
	require.NoError(t, err)
	return codec
}

func TestSessionCookieCodec_EncryptDecryptRoundtrip(t *testing.T) {
	t.Parallel()

	codec := newCodec(t)
	const jwt = "header.payload.signature"

	value, err := codec.Encrypt(jwt)
	require.NoError(t, err)
	assert.NotContains(t, value, jwt, "cookie value must be opaque")

	got, err := codec.Decrypt(value)
	require.NoError(t, err)
	assert.Equal(t, jwt, got)
}

func TestSessionCookieCodec_DecryptRejectsTampered(t *testing.T) {
	t.Parallel()

	codec := newCodec(t)
	value, err := codec.Encrypt("the.jwt.token")
	require.NoError(t, err)

	// Corrupt a byte of the ciphertext so GCM authentication fails.
	raw, err := base64.RawURLEncoding.DecodeString(value)
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	_, err = codec.Decrypt(tampered)
	require.ErrorIs(t, err, auth.ErrInvalidCookie)
}

func TestSessionCookieCodec_DecryptRejectsGarbage(t *testing.T) {
	t.Parallel()

	codec := newCodec(t)
	_, err := codec.Decrypt("not-valid-base64-or-cipher!!!")
	require.ErrorIs(t, err, auth.ErrInvalidCookie)
}

func TestSessionCookieCodec_DecryptRejectsOtherKey(t *testing.T) {
	t.Parallel()

	codec := newCodec(t)
	value, err := codec.Encrypt("the.jwt.token")
	require.NoError(t, err)

	other, err := auth.NewSessionCookieCodec([][]byte{append(make([]byte, 31), 0x01)}, false, time.Hour, "")
	require.NoError(t, err)

	_, err = other.Decrypt(value)
	require.ErrorIs(t, err, auth.ErrInvalidCookie)
}

func TestSessionCookieCodec_DecryptWithRotatedKey(t *testing.T) {
	t.Parallel()

	oldKey := make([]byte, 32)
	newKey := append(make([]byte, 31), 0x01)

	// Cookie encrypted with the old key.
	oldCodec, err := auth.NewSessionCookieCodec([][]byte{oldKey}, false, time.Hour, "")
	require.NoError(t, err)
	value, err := oldCodec.Encrypt("the.jwt.token")
	require.NoError(t, err)

	// After rotation: new key first, old key still present for decryption.
	rotated, err := auth.NewSessionCookieCodec([][]byte{newKey, oldKey}, false, time.Hour, "")
	require.NoError(t, err)

	got, err := rotated.Decrypt(value)
	require.NoError(t, err)
	assert.Equal(t, "the.jwt.token", got, "old-key cookie must decrypt after rotation")

	// New encryptions use the new key and cannot be decrypted by old-only codec.
	newValue, err := rotated.Encrypt("new.jwt.token")
	require.NoError(t, err)

	_, err = oldCodec.Decrypt(newValue)
	require.ErrorIs(t, err, auth.ErrInvalidCookie, "new-key cookie must not decrypt with old key only")
}

func TestStrictMiddleware_SetsCookieOnLogin(t *testing.T) {
	t.Parallel()

	codec := newCodec(t)
	handler := codec.StrictMiddleware()(
		func(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
			return gen.Login200JSONResponse{Token: "signed.jwt.value"}, nil
		},
		"Login",
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", http.NoBody)
	_, err := handler(context.Background(), w, req, gen.LoginRequestObject{})
	require.NoError(t, err)

	cookie := findCookie(w.Result().Cookies(), codec.Name())
	require.NotNil(t, cookie, "session cookie must be set on login")
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Positive(t, cookie.MaxAge)

	jwt, err := codec.Decrypt(cookie.Value)
	require.NoError(t, err)
	assert.Equal(t, "signed.jwt.value", jwt)
}

func TestStrictMiddleware_ClearsCookieOnLogout(t *testing.T) {
	t.Parallel()

	codec := newCodec(t)
	handler := codec.StrictMiddleware()(
		func(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
			return gen.Logout204Response{}, nil
		},
		"Logout",
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout", http.NoBody)
	_, err := handler(context.Background(), w, req, gen.LogoutRequestObject{})
	require.NoError(t, err)

	cookie := findCookie(w.Result().Cookies(), codec.Name())
	require.NotNil(t, cookie, "session cookie must be overwritten on logout")
	assert.Empty(t, cookie.Value)
	assert.Negative(t, cookie.MaxAge)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
