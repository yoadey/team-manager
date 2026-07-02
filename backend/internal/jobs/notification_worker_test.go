package jobs_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// TestNotificationWorker_InsertsNotificationRow verifies that Work() persists
// a notification row with the fields carried on the job args.
func TestNotificationWorker_InsertsNotificationRow(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Notify Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor', 'actor@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	title := "Neues Training"
	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		Args: jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "news",
			ActorID: actorID,
			Title:   &title,
		},
	}

	require.NoError(t, worker.Work(ctx, job))

	var count int
	var gotTitle *string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*), MAX(title) FROM notifications WHERE team_id = $1`, teamID,
	).Scan(&count, &gotTitle))
	assert.Equal(t, 1, count)
	require.NotNil(t, gotTitle)
	assert.Equal(t, title, *gotTitle)
}

// TestNotificationWorker_Work_IsIdempotentOnRetry is a regression test for a
// worker whose doc comment claimed the insert was idempotent via River's
// job-state tracking, when in fact a bare INSERT with no unique key would
// create a duplicate row if River retried the job (its at-least-once
// delivery guarantee) after a crash between commit and job-completion ack.
func TestNotificationWorker_Work_IsIdempotentOnRetry(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Idempotent Notify Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor2', 'actor2@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	title := "Retried Notification"
	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 424242},
		Args: jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "news",
			ActorID: actorID,
			Title:   &title,
		},
	}

	// Simulate River retrying the same job after a crash: Work() runs twice
	// with the same job.ID.
	require.NoError(t, worker.Work(ctx, job))
	require.NoError(t, worker.Work(ctx, job))

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE team_id = $1`, teamID,
	).Scan(&count))
	assert.Equal(t, 1, count, "a retried job with the same ID must not create a duplicate notification row")
}
