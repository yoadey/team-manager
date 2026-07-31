package push_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/push"
)

// validSubscription generates a syntactically valid P256dh/auth keypair --
// SendNotificationWithContext performs real ECDH math on these before it
// ever reaches the HTTP layer, so a placeholder string would fail before
// the test server sees the request.
func validSubscription(t *testing.T, endpoint string) push.Subscription {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	p256dh := base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())

	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)

	return push.Subscription{
		Endpoint: endpoint,
		P256dh:   p256dh,
		AuthKey:  base64.RawURLEncoding.EncodeToString(auth),
	}
}

func validVAPIDConfig(t *testing.T) push.VAPIDConfig {
	t.Helper()
	priv, pub, err := webpush.GenerateVAPIDKeys()
	require.NoError(t, err)
	return push.VAPIDConfig{PublicKey: pub, PrivateKey: priv, Subject: "mailto:ops@example.com"}
}

func TestNewWebPusher_RequiresVAPIDKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  push.VAPIDConfig
	}{
		{"both empty", push.VAPIDConfig{}},
		{"missing private key", push.VAPIDConfig{PublicKey: "pub", Subject: "mailto:ops@example.com"}},
		{"missing public key", push.VAPIDConfig{PrivateKey: "priv", Subject: "mailto:ops@example.com"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := push.NewWebPusher(c.cfg)
			require.Error(t, err)
			assert.ErrorIs(t, err, push.ErrVAPIDKeysRequired)
		})
	}
}

func TestNewWebPusher_ValidConfig(t *testing.T) {
	t.Parallel()

	p, err := push.NewWebPusher(push.VAPIDConfig{
		PublicKey:  "pub",
		PrivateKey: "priv",
		Subject:    "mailto:ops@example.com",
	})
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestWebPusher_Send_NonGoneStatusIncludesResponseBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"errno":109,"message":"Invalid VAPID token"}`))
	}))
	t.Cleanup(server.Close)

	p, err := push.NewWebPusher(validVAPIDConfig(t))
	require.NoError(t, err)

	err = p.Send(context.Background(), validSubscription(t, server.URL), push.Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.ErrorIs(t, err, push.ErrPushServiceStatus)
	assert.Contains(t, err.Error(), "status 401")
	assert.Contains(t, err.Error(), "Invalid VAPID token")
}

func TestWebPusher_Send_TruncatesLargeResponseBody(t *testing.T) {
	t.Parallel()

	oversized := strings.Repeat("x", 4096)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(oversized))
	}))
	t.Cleanup(server.Close)

	p, err := push.NewWebPusher(validVAPIDConfig(t))
	require.NoError(t, err)

	err = p.Send(context.Background(), validSubscription(t, server.URL), push.Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.Less(t, len(err.Error()), len(oversized))
}

func TestWebPusher_Send_MozillaVAPIDKeyMismatchMapsToErrGone(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"errno":109,"error":"Unauthorized","message":"VAPID public key mismatch","more_info":"http://autopush.readthedocs.io/en/latest/http.html#error-codes"}`))
	}))
	t.Cleanup(server.Close)

	p, err := push.NewWebPusher(validVAPIDConfig(t))
	require.NoError(t, err)

	err = p.Send(context.Background(), validSubscription(t, server.URL), push.Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.ErrorIs(t, err, push.ErrGone, "a VAPID key mismatch means this subscription can never be delivered to again")
}

func TestWebPusher_Send_FCMVAPIDCredentialMismatchMapsToErrGone(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("the VAPID credentials in the authorization header do not correspond to the credentials used to create the subscriptions."))
	}))
	t.Cleanup(server.Close)

	p, err := push.NewWebPusher(validVAPIDConfig(t))
	require.NoError(t, err)

	err = p.Send(context.Background(), validSubscription(t, server.URL), push.Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.ErrorIs(t, err, push.ErrGone)
}

func TestWebPusher_Send_PayloadTooLargeMapsToErrPayloadTooLarge(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte(`{"code":413,"errno":104,"error":"Payload Too Large","message":"This message is intended for a constrained device and is limited in size. Converted buffer is too long by 1441 bytes"}`))
	}))
	t.Cleanup(server.Close)

	p, err := push.NewWebPusher(validVAPIDConfig(t))
	require.NoError(t, err)

	err = p.Send(context.Background(), validSubscription(t, server.URL), push.Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.ErrorIs(t, err, push.ErrPayloadTooLarge, "a 413 means this payload can never be delivered, retrying it is pointless")
	assert.Contains(t, err.Error(), "constrained device")
}

func TestWebPusher_Send_GoneStatusHasNoBodyLookup(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	t.Cleanup(server.Close)

	p, err := push.NewWebPusher(validVAPIDConfig(t))
	require.NoError(t, err)

	err = p.Send(context.Background(), validSubscription(t, server.URL), push.Payload{Title: "t", Body: "b"})
	require.Error(t, err)
	assert.ErrorIs(t, err, push.ErrGone)
}
