package db

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// slowQueryThreshold is the minimum query duration that triggers a warning log.
const slowQueryThreshold = time.Second

type queryTraceKey struct{}

type queryTraceData struct {
	start time.Time
	sql   string
}

// SlowQueryTracer implements pgx.QueryTracer and logs any query that exceeds
// slowQueryThreshold. It is attached to the pgxpool connection config so that
// every query in the application is automatically monitored without requiring
// changes to individual repositories.
type SlowQueryTracer struct {
	logger *slog.Logger
}

// NewSlowQueryTracer creates a SlowQueryTracer that logs to logger.
func NewSlowQueryTracer(logger *slog.Logger) *SlowQueryTracer {
	return &SlowQueryTracer{logger: logger}
}

func (t *SlowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryTraceKey{}, &queryTraceData{start: time.Now(), sql: data.SQL})
}

func (t *SlowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	d, ok := ctx.Value(queryTraceKey{}).(*queryTraceData)
	if !ok {
		return
	}
	duration := time.Since(d.start)
	if duration >= slowQueryThreshold {
		t.logger.WarnContext(ctx, "slow query detected",
			slog.Duration("duration", duration),
			slog.String("sql", d.sql),
		)
	}
}
