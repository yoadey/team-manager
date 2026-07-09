package jobs_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// Regression test: unlike db.RunMigrations (which passes goose a
// WithSessionLocker so concurrent Up() calls across processes serialize on
// a Postgres advisory lock), jobs.MigrateRiver used to call River's own
// migrator with no equivalent locking -- several replicas migrating
// concurrently against a fresh schema (the Helm chart's per-pod migrate
// initContainer running on multiple pods during a rolling update, HPA
// scale-out, or a fresh multi-replica install) could race on the same DDL
// and fail with a duplicate-object error, since River's migration tables
// are created without IF NOT EXISTS/ON CONFLICT. This fires many concurrent
// MigrateRiver calls against the same fresh (no River schema yet) database
// and asserts every one succeeds.
func TestMigrateRiver_ConcurrentCallsAllSucceed(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	const concurrency = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			errs[idx] = jobs.MigrateRiver(ctx, pool)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "concurrent MigrateRiver call %d must not fail", i)
	}
}
