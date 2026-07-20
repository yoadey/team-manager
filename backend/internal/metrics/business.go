// Package metrics exposes domain-level Prometheus counters that complement the
// HTTP-layer metrics in internal/middleware/metrics.go. HTTP metrics answer
// "how many requests?" at the infrastructure level; these counters answer
// "how many logins succeeded?" at the business level, enabling product and
// security dashboards independent of route naming conventions.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// LoginAttempts counts password login attempts split by outcome.
	// outcome label values: "success", "failure".
	LoginAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "login_attempts_total",
		Help:      "Total password login attempts by outcome (success / failure).",
	}, []string{"outcome"})

	// RegisterAttempts counts self-registration attempts split by outcome.
	// outcome label values: "success" (a generic 202, regardless of which of
	// Register's three enumeration-safe branches was taken), "disabled"
	// (SELF_REGISTRATION_ENABLED=false), "invalid" (validation failure).
	RegisterAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "register_attempts_total",
		Help:      "Total self-registration attempts by outcome.",
	}, []string{"outcome"})

	// RateLimitHits counts requests rejected by the rate limiter.
	// context label values: "global" (per-IP general limit), "login" (login endpoint).
	RateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "rate_limit_hits_total",
		Help:      "Total requests rejected by the rate limiter by context.",
	}, []string{"context"})

	// TeamEvents counts domain events by operation and module.
	// operation label values: "create", "update", "delete".
	// module label values: "team", "member", "event", "news", "poll", "finance".
	TeamEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "domain_events_total",
		Help:      "Total domain-level CRUD events by module and operation.",
	}, []string{"module", "operation"})

	// SlowQueries counts database queries whose duration exceeded the
	// slow-query threshold (see internal/db.SlowQueryTracer). Backs the
	// SlowQueriesDetected alert.
	SlowQueries = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "slow_queries_total",
		Help:      "Total number of database queries exceeding the slow-query threshold.",
	})

	// RetentionJobLastSuccessTimestamp is the Unix timestamp of the retention
	// job's (internal/jobs.RetentionWorker) last successful run. A silent
	// repeated failure (e.g. a permissions or lock issue) would otherwise let
	// notifications/sessions/audit_log grow unbounded with no signal until the
	// DB runs out of disk — this backs an alert on staleness (job runs daily).
	RetentionJobLastSuccessTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "teammanager",
		Name:      "retention_job_last_success_timestamp_seconds",
		Help:      "Unix timestamp of the retention job's last successful run.",
	})

	// RetentionJobRowsDeleted counts rows deleted by the retention job, by table.
	RetentionJobRowsDeleted = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "retention_job_rows_deleted_total",
		Help:      "Total rows deleted by the retention job, by table.",
	}, []string{"table"})

	// RetentionJobFailures counts failed retention job runs, by table (the
	// table whose delete failed and aborted the run).
	RetentionJobFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "retention_job_failures_total",
		Help:      "Total retention job run failures, by table whose delete failed.",
	}, []string{"table"})

	// NotificationEnqueueFailures counts failures to enqueue a notification
	// job (internal/jobs.Client.EnqueueNotification). Callers treat this as
	// best-effort and only log a warning so a transient enqueue failure
	// doesn't fail the request that triggered it (e.g. creating a poll/news
	// item) -- without a counter, that failure mode was otherwise invisible
	// to dashboards/alerts, unlike every other domain mutation which is
	// tracked via TeamEvents.
	NotificationEnqueueFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "notification_enqueue_failures_total",
		Help:      "Total failures to enqueue a notification job.",
	})

	// NotificationJobFailures counts internal/jobs.NotificationWorker.Work
	// failures (the INSERT into the notifications table). River retries with
	// backoff and eventually discards the job after max attempts; without
	// this counter (mirroring RetentionJobFailures for the same "silent
	// background failure" risk), a persistent failure here would leave users
	// silently not receiving notifications with no operational signal.
	NotificationJobFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "notification_job_failures_total",
		Help:      "Total internal/jobs.NotificationWorker.Work failures.",
	})
)
