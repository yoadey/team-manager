package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// ErrVAPIDKeysRequired is returned by NewWebPusher when either VAPID key is empty.
var ErrVAPIDKeysRequired = errors.New("push.NewWebPusher: VAPIDPublicKey and VAPIDPrivateKey are both required")

// ErrPushServiceStatus is wrapped into the error WebPusher.Send returns when
// the push service responds with a non-2xx status other than 404/410 (which
// map to ErrGone instead).
var ErrPushServiceStatus = errors.New("push: push service returned a non-2xx status")

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
// or 410 response is mapped to ErrGone so callers know to delete the
// subscription; any other non-2xx status or transport error is returned
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
		return fmt.Errorf("push.WebPusher.Send: %w: status %d", ErrPushServiceStatus, resp.StatusCode)
	}
	return nil
}
