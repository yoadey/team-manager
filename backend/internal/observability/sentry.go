package observability

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry error tracking when dsn is non-empty; otherwise
// it is a no-op (mirrors the frontend, where Sentry is disabled without a DSN).
// The returned flush function should be called on shutdown to deliver buffered
// events.
func InitSentry(dsn, environment, release string) (func(time.Duration), error) {
	noop := func(time.Duration) {}
	if dsn == "" {
		return noop, nil
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: environment,
		Release:     release,
	}); err != nil {
		return noop, fmt.Errorf("observability.InitSentry: %w", err)
	}
	return func(d time.Duration) { sentry.Flush(d) }, nil
}
