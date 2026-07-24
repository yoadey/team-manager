// Package push abstracts outbound Web Push delivery away from the jobs and
// HTTP-handler packages, mirroring how internal/mailer abstracts outbound
// email: a small interface, a real implementation (VAPID + webpush-go), and
// an in-memory/logging fake for dev and tests.
package push

import (
	"context"
	"errors"
)

// ErrGone is returned by Pusher.Send when the push service reports (via a
// 404 or 410 HTTP status) that the subscription's endpoint will never accept
// another push. Callers must delete the corresponding push_subscriptions row
// when they see this error -- retrying it is pointless.
var ErrGone = errors.New("push: subscription is gone (404/410)")

// Subscription is the browser-issued Web Push subscription needed to
// address a single device.
type Subscription struct {
	Endpoint string
	P256dh   string
	AuthKey  string
}

// Payload is the small JSON-serializable notification body delivered to the
// browser's push event listener.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// URL is the in-app route to focus/open when the notification is
	// clicked (relative, e.g. "/"); optional.
	URL string `json:"url,omitempty"`
}

// Pusher sends a single Web Push notification to one subscription.
type Pusher interface {
	Send(ctx context.Context, sub Subscription, payload Payload) error
}
