package observability_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/observability"
)

func TestInitTracer_DisabledWhenNoEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	shutdown, err := observability.InitTracer(context.Background(), "svc", "test")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()), "no-op shutdown must succeed")
}

func TestInitSentry_DisabledWhenNoDSN(t *testing.T) {
	flush, err := observability.InitSentry("", "test", "v1")
	require.NoError(t, err)
	require.NotNil(t, flush)
	assert.NotPanics(t, func() { flush(time.Second) }, "no-op flush must not panic")
}
