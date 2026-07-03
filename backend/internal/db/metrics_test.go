package db_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/db"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// TestPoolStatsCollector_ExportsExpectedMetrics is a regression test for the
// DatabaseConnectionPoolExhausted alert, which previously queried a metric
// name (pgxpool_acquire_duration_seconds) that was never registered anywhere
// in the codebase and so could never fire. Verifies the collector actually
// registers and reports pool stats under the names the alert now expects.
func TestPoolStatsCollector_ExportsExpectedMetrics(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	collector := db.NewPoolStatsCollector(pool)

	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err)

	names := make(map[string]bool, len(families))
	for _, f := range families {
		names[f.GetName()] = true
	}

	for _, want := range []string{
		"teammanager_pgxpool_acquired_conns",
		"teammanager_pgxpool_idle_conns",
		"teammanager_pgxpool_max_conns",
		"teammanager_pgxpool_total_conns",
		"teammanager_pgxpool_acquire_count_total",
		"teammanager_pgxpool_empty_acquire_count_total",
		"teammanager_pgxpool_acquire_duration_seconds_total",
	} {
		assert.True(t, names[want], "expected metric %q to be registered", want)
	}
}
