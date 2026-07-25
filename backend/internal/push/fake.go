package push

import (
	"context"
	"log/slog"
	"sync"
)

// FakePusher is an in-memory Pusher for dev and tests: it logs the payload
// instead of sending a real push, and records the last send so tests can
// assert on it without a real push service.
type FakePusher struct {
	logger *slog.Logger

	mu           sync.Mutex
	lastEndpoint string
	lastPayload  Payload
	sentCount    int
}

// NewFakePusher creates a FakePusher that logs via logger.
func NewFakePusher(logger *slog.Logger) *FakePusher {
	if logger == nil {
		logger = slog.Default()
	}
	return &FakePusher{logger: logger}
}

// Send logs payload instead of sending it, and records it for test assertions.
func (f *FakePusher) Send(ctx context.Context, sub Subscription, payload Payload) error {
	f.logger.InfoContext(ctx, "push: notification (VAPID not configured, logging instead)",
		"endpoint", sub.Endpoint, "title", payload.Title)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastEndpoint = sub.Endpoint
	f.lastPayload = payload
	f.sentCount++
	return nil
}

// LastSent returns the endpoint and payload of the most recently sent
// notification. Test helper.
func (f *FakePusher) LastSent() (endpoint string, payload Payload) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastEndpoint, f.lastPayload
}

// SentCount returns the total number of notifications sent. Test helper.
func (f *FakePusher) SentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sentCount
}
