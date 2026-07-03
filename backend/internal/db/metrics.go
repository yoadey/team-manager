package db

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// statter is the subset of *pgxpool.Pool used by poolStatsCollector, so tests
// can supply a fake without a live database connection.
type statter interface {
	Stat() *pgxpool.Stat
}

// poolStatsCollector exports pgxpool connection-pool statistics as Prometheus
// gauges/counters on each scrape. There is no equivalent for the standard
// library's database/sql collector when using pgxpool directly, so this is
// the only source of pool-health metrics (used by the
// DatabaseConnectionPoolExhausted alert).
type poolStatsCollector struct {
	pool statter

	acquiredConns         *prometheus.Desc
	idleConns             *prometheus.Desc
	maxConns              *prometheus.Desc
	totalConns            *prometheus.Desc
	acquireCount          *prometheus.Desc
	emptyAcquireCount     *prometheus.Desc
	acquireDurationSecond *prometheus.Desc
}

// NewPoolStatsCollector creates a prometheus.Collector that reports pool
// stats for the given pgxpool.Pool. Register it with the metrics registry
// once at startup (e.g. prometheus.MustRegister(db.NewPoolStatsCollector(pool))).
func NewPoolStatsCollector(pool *pgxpool.Pool) prometheus.Collector {
	return &poolStatsCollector{
		pool: pool,
		acquiredConns: prometheus.NewDesc(
			"teammanager_pgxpool_acquired_conns", "Number of currently acquired connections in the pool.", nil, nil),
		idleConns: prometheus.NewDesc(
			"teammanager_pgxpool_idle_conns", "Number of currently idle connections in the pool.", nil, nil),
		maxConns: prometheus.NewDesc(
			"teammanager_pgxpool_max_conns", "Maximum size of the pool.", nil, nil),
		totalConns: prometheus.NewDesc(
			"teammanager_pgxpool_total_conns", "Total number of connections currently open (idle + acquired + constructing).", nil, nil),
		acquireCount: prometheus.NewDesc(
			"teammanager_pgxpool_acquire_count_total", "Cumulative number of successful connection acquisitions.", nil, nil),
		emptyAcquireCount: prometheus.NewDesc(
			"teammanager_pgxpool_empty_acquire_count_total", "Cumulative number of acquisitions that had to wait for a connection (pool was empty).", nil, nil),
		acquireDurationSecond: prometheus.NewDesc(
			"teammanager_pgxpool_acquire_duration_seconds_total", "Cumulative time spent waiting for a connection to be acquired.", nil, nil),
	}
}

func (c *poolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.maxConns
	ch <- c.totalConns
	ch <- c.acquireCount
	ch <- c.emptyAcquireCount
	ch <- c.acquireDurationSecond
}

func (c *poolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(s.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquireCount, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.acquireDurationSecond, prometheus.CounterValue, s.AcquireDuration().Seconds())
}
