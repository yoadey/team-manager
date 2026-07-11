package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

type slowJobArgs struct{}

func (slowJobArgs) Kind() string { return "slow_test_job" }

// slowJobWorker blocks in Work until its context is cancelled, standing in
// for a job still mid-run when a graceful shutdown starts (like
// RetentionWorker, whose own Timeout() budget is 150s).
type slowJobWorker struct {
	river.WorkerDefaults[slowJobArgs]
	started chan struct{}
}

func (w *slowJobWorker) Work(ctx context.Context, _ *river.Job[slowJobArgs]) error {
	close(w.started)
	<-ctx.Done()
	return ctx.Err()
}

// Regression test for cmd/server/main.go's graceful-shutdown race: without
// river.Config.SoftStopTimeout configured, a timed-out river.Client.Stop call
// does NOT cancel a still-running job -- Stop just returns an error early
// while the job keeps executing (and holding a pool connection) for up to
// its own Timeout() budget, racing pool.Close() right after it. Wiring
// jobs.SoftStopTimeout into river.Config (the same way jobs.NewClient does)
// must make Stop() return in bounded time -- roughly SoftStopTimeout -- even
// while a job is still stuck mid-run, by auto-cancelling that job's context
// once the soft-stop window elapses.
func TestSoftStopTimeout_BoundsStopDuration_EvenWithStillRunningJob(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	require.NoError(t, jobs.MigrateRiver(ctx, pool))

	worker := &slowJobWorker{started: make(chan struct{})}
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:          map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers:         workers,
		SoftStopTimeout: jobs.SoftStopTimeout,
	})
	require.NoError(t, err)
	require.NoError(t, rc.Start(ctx))

	_, err = rc.Insert(ctx, slowJobArgs{}, nil)
	require.NoError(t, err)

	select {
	case <-worker.started:
	case <-time.After(10 * time.Second):
		t.Fatal("slow job never started")
	}

	stopStart := time.Now()
	// Deliberately generous -- if SoftStopTimeout were NOT honored, Stop
	// would block for this entire budget waiting on the job that never
	// finishes on its own, instead of returning near jobs.SoftStopTimeout.
	stopCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = rc.Stop(stopCtx)
	elapsed := time.Since(stopStart)

	assert.NoError(t, err)
	assert.Less(t, elapsed, 15*time.Second,
		"Stop must return near jobs.SoftStopTimeout (%s) via automatic job cancellation, not block for the job's full run", jobs.SoftStopTimeout)
}
