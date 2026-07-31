package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// maxPushErrorBodyBytes bounds how much of a non-2xx push service response
// body is read into an error message -- push services return small JSON
// diagnostics on auth failures (e.g. a VAPID key mismatch), and this caps
// how much of that a misbehaving service could bloat a log line with.
const maxPushErrorBodyBytes = 2048

// ErrVAPIDKeysRequired is returned by NewWebPusher when either VAPID key is empty.
var ErrVAPIDKeysRequired = errors.New("push.NewWebPusher: VAPIDPublicKey and VAPIDPrivateKey are both required")

// ErrPushServiceStatus is wrapped into the error WebPusher.Send returns when
// the push service responds with a non-2xx status other than 404/410 (which
// map to ErrGone instead).
var ErrPushServiceStatus = errors.New("push: push service returned a non-2xx status")

// vapidKeyMismatchSignatures are case-insensitive substrings a push
// service's 401/403 response body uses specifically when a subscription
// was created against a VAPID public key this server no longer signs
// with -- as opposed to a generic auth rejection (e.g. a malformed
// VAPID_SUBJECT) that affects every subscription equally and can be
// recovered by a config fix alone. A subscription in this state can never
// be delivered to again short of the browser re-subscribing, so it is
// treated like ErrGone rather than retried forever.
var vapidKeyMismatchSignatures = []string{
	"vapid public key mismatch",                   // Mozilla autopush, errno 109
	"credentials used to create the subscription", // FCM (also matches plural "subscriptions")
}

// isVAPIDKeyMismatch reports whether body names a known VAPID
// key-mismatch signature.
func isVAPIDKeyMismatch(body string) bool {
	lower := strings.ToLower(body)
	for _, sig := range vapidKeyMismatchSignatures {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// VAPIDConfig holds the settings needed to authenticate this server to
// browser push services via VAPID (RFC 8292).
type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
	// Subject identifies the sender to the push service, e.g.
	// "mailto:ops@example.com" -- required by the VAPID spec.
	Subject string
}

// WebPusher sends real Web Push notifications using VAPID authentication.
type WebPusher struct {
	cfg VAPIDConfig
}

// NewWebPusher validates cfg and returns a WebPusher.
func NewWebPusher(cfg VAPIDConfig) (*WebPusher, error) {
	if cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return nil, ErrVAPIDKeysRequired
	}
	return &WebPusher{cfg: cfg}, nil
}

// webpushTimeout bounds a single push send so a slow/unresponsive push
// service can't stall the delivery worker indefinitely.
const webpushTimeout = 10 * time.Second

// Send delivers payload to sub via the browser vendor's push service. A 404
// or 410 response, or a 401/403 whose body names a known VAPID
// key-mismatch signature (see isVAPIDKeyMismatch), is mapped to ErrGone so
// callers know to delete the subscription. A 413 is mapped to
// ErrPayloadTooLarge so callers know not to retry without deleting the
// subscription. Any other non-2xx status or transport error is returned
// as-is for River's built-in retry/backoff to handle.
func (p *WebPusher) Send(ctx context.Context, sub Subscription, payload Payload) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("push.WebPusher.Send: marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, webpushTimeout)
	defer cancel()

	resp, err := webpush.SendNotificationWithContext(ctx, message, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.AuthKey,
		},
	}, &webpush.Options{
		Subscriber:      p.cfg.Subject,
		VAPIDPublicKey:  p.cfg.PublicKey,
		VAPIDPrivateKey: p.cfg.PrivateKey,
		TTL:             int((24 * time.Hour).Seconds()),
	})
	if err != nil {
		return fmt.Errorf("push.WebPusher.Send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return ErrGone
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxPushErrorBodyBytes))
		snippet := strings.TrimSpace(strings.ReplaceAll(string(body), "\n", " "))
		if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && isVAPIDKeyMismatch(snippet) {
			return ErrGone
		}
		if resp.StatusCode == http.StatusRequestEntityTooLarge {
			if snippet == "" {
				return fmt.Errorf("push.WebPusher.Send: %w", ErrPayloadTooLarge)
			}
			return fmt.Errorf("push.WebPusher.Send: %w: %s", ErrPayloadTooLarge, snippet)
		}
		if snippet == "" {
			return fmt.Errorf("push.WebPusher.Send: %w: status %d", ErrPushServiceStatus, resp.StatusCode)
		}
		return fmt.Errorf("push.WebPusher.Send: %w: status %d: %s", ErrPushServiceStatus, resp.StatusCode, snippet)
	}
	return nil
}
