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
	Type       string     `json:"type"`  // "news" | "event_created" | "event_cancelled" | "poll"
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
// River provides at-least-once delivery; the INSERT is idempotent via River's
// job-state tracking.
type NotificationWorker struct {
	river.WorkerDefaults[NotificationArgs]
	pool *pgxpool.Pool
}

// Work is called by River for each job. It inserts a row into the notifications table.
func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[NotificationArgs]) error {
	a := job.Args
	_, err := w.pool.Exec(ctx, `
		INSERT INTO notifications
		    (team_id, type, actor_id, event_id, event_title, event_date, title, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, a.TeamID, a.Type, a.ActorID, a.EventID, a.EventTitle, a.EventDate, a.Title, a.Note)
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
// Call Start() on the returned river.Client separately if running workers
// in the same process.
func NewClient(pool *pgxpool.Pool) (*Client, *river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &NotificationWorker{pool: pool})

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		Logger:  slog.Default(),
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
