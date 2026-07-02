// Package jobs defines River background workers for the team-manager backend.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// ─── Job args ────────────────────────────────────────────────────────────────

// NotificationArgs carries the data required to persist a single notification row.
type NotificationArgs struct {
	TeamID     uuid.UUID  `json:"team_id"`
	Type       string     `json:"type"` // "news" | "event_created" | "event_cancelled" | "poll"
	ActorID    uuid.UUID  `json:"actor_id"`
	EventID    *uuid.UUID `json:"event_id,omitempty"`
	EventTitle *string    `json:"event_title,omitempty"`
	EventDate  *time.Time `json:"event_date,omitempty"`
	Title      *string    `json:"title,omitempty"`
	Note       *string    `json:"note,omitempty"`
}

// Kind implements river.JobArgs — must be unique per worker type.
func (NotificationArgs) Kind() string { return "notification" }

// ─── Worker ───────────────────────────────────────────────────────────────────

// NotificationWorker inserts a notification row into Postgres.
// River provides at-least-once delivery (not exactly-once): the same job can
// run more than once if the process crashes after the INSERT commits but
// before River records the job as complete. The insert keys on job.ID (via
// the unique river_job_id column) with ON CONFLICT DO NOTHING to make a retry
// a no-op instead of creating a duplicate notification row.
type NotificationWorker struct {
	river.WorkerDefaults[NotificationArgs]
	pool *pgxpool.Pool
}

// NewNotificationWorker constructs a NotificationWorker backed by pool.
func NewNotificationWorker(pool *pgxpool.Pool) *NotificationWorker {
	return &NotificationWorker{pool: pool}
}

// Work is called by River for each job. It inserts a row into the notifications table.
func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[NotificationArgs]) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := job.Args
	_, err := w.pool.Exec(ctx, `
		INSERT INTO notifications
		    (team_id, type, actor_id, event_id, event_title, event_date, title, note, river_job_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (river_job_id) WHERE river_job_id IS NOT NULL DO NOTHING
	`, a.TeamID, a.Type, a.ActorID, a.EventID, a.EventTitle, a.EventDate, a.Title, a.Note, job.ID)
	if err != nil {
		return fmt.Errorf("jobs.NotificationWorker: insert notification: %w", err)
	}
	return nil
}

// ─── Client ───────────────────────────────────────────────────────────────────

// Client wraps river.Client[pgx.Tx] for enqueuing notification jobs from services.
type Client struct {
	rc *river.Client[pgx.Tx]
}

// NewClient creates a River client backed by the given pool.
// retentionWorker is optional: pass nil to skip registering the retention job.
// Call Start() on the returned river.Client separately if running workers
// in the same process.
func NewClient(pool *pgxpool.Pool, retentionWorker *RetentionWorker) (client *Client, riverClient *river.Client[pgx.Tx], err error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, NewNotificationWorker(pool))

	var periodicJobs []*river.PeriodicJob
	if retentionWorker != nil {
		river.AddWorker(workers, retentionWorker)
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(24*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return RetentionArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		))
	}

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
		Logger:       slog.Default(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("jobs.NewClient: %w", err)
	}
	return &Client{rc: rc}, rc, nil
}

// EnqueueNotification enqueues a notification job for delivery by the worker.
// The call is non-blocking; River persists the job to Postgres.
func (c *Client) EnqueueNotification(ctx context.Context, args NotificationArgs) error {
	_, err := c.rc.Insert(ctx, args, nil)
	if err != nil {
		return fmt.Errorf("jobs.Client.EnqueueNotification: %w", err)
	}
	return nil
}

// MigrateRiver runs River's built-in schema migrations against the given pool.
// It must be called once during startup, after the application schema migrations.
func MigrateRiver(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("jobs.MigrateRiver: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("jobs.MigrateRiver: %w", err)
	}
	return nil
}
