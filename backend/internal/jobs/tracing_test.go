package jobs_test

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// Regression test: internal/jobs previously had no OTEL span coverage at
// all -- an HTTP request's trace ends the moment EnqueueNotification
// returns, so the actual asynchronous delivery (NotificationWorker.Work)
// and the daily RetentionWorker.Work were entirely invisible in tracing
// when OTEL_EXPORTER_OTLP_ENDPOINT is set, with no way to correlate a
// failure back to a trace. Not run with t.Parallel(): it swaps the global
// TracerProvider for its duration, which would race with sibling tests in
// this package that also exercise Work() through the same package-level
// tracer.
func TestRetentionWorker_Work_RecordsErrorSpanOnFailure(t *testing.T) {
	pool := testutil.NewTestDB(t)
	pool.Close() // every subsequent query now fails cleanly with a closed-pool error

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prev)

	worker := jobs.NewRetentionWorker(pool, 0, 0, 0, 0)
	err := worker.Work(context.Background(), &river.Job[jobs.RetentionArgs]{JobRow: &rivertype.JobRow{ID: 1}})
	require.Error(t, err)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "retention.work", spans[0].Name())
	assert.Equal(t, codes.Error, spans[0].Status().Code)
	require.NotEmpty(t, spans[0].Events(), "span must record the error as an event")
}
