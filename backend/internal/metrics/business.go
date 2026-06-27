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
)
