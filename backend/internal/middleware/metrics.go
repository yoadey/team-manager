package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "teammanager",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests by method, path pattern, and status.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "teammanager",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds by method and path pattern.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	httpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "teammanager",
		Name:      "http_requests_in_flight",
		Help:      "Number of HTTP requests currently being served.",
	})
)

// Metrics returns middleware that records Prometheus metrics for each request:
// request count (by method, path, status), latency histogram, and in-flight gauge.
//
// Recording happens in a defer (rather than immediately after next.ServeHTTP
// returns) so a panicking handler still decrements the in-flight gauge and
// records a request/duration sample — Recoverer wraps this middleware, so
// without the defer a panic would unwind straight past the bookkeeping below,
// permanently leaking the in-flight gauge and undercounting error rates. The
// panic is re-raised afterward so Recoverer still handles the response and
// logging as before.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := newResponseWriter(w)
		httpRequestsInFlight.Inc()
		start := time.Now()

		defer func() {
			httpRequestsInFlight.Dec()
			status := rw.status
			rec := recover()
			if rec != nil {
				status = http.StatusInternalServerError
			}
			httpRequestsTotal.WithLabelValues(r.Method, routeLabel(r), strconv.Itoa(status)).Inc()
			httpRequestDuration.WithLabelValues(r.Method, routeLabel(r)).Observe(time.Since(start).Seconds())
			if rec != nil {
				panic(rec)
			}
		}()

		next.ServeHTTP(rw, r)
	})
}

// routeLabel returns the matched chi route pattern (e.g.
// "/api/v1/teams/{teamId}/events/{id}") rather than the raw request path, so
// dynamic UUID path segments never become distinct label values. Using
// r.URL.Path directly would give any caller — even unauthenticated, since
// this middleware runs before auth — unbounded control over the label
// cardinality of an in-process Prometheus metric, a memory-growth vector.
// Falls back to a fixed placeholder for paths that matched no route (e.g.
// 404s), since RoutePattern() is empty in that case.
func routeLabel(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}
