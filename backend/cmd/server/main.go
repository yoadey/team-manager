package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/config"
	"github.com/yoadey/team-manager/backend/internal/db"
	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/mailer"
	"github.com/yoadey/team-manager/backend/internal/members"
	"github.com/yoadey/team-manager/backend/internal/middleware"
	"github.com/yoadey/team-manager/backend/internal/news"
	"github.com/yoadey/team-manager/backend/internal/notifications"
	"github.com/yoadey/team-manager/backend/internal/observability"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/polls"
	"github.com/yoadey/team-manager/backend/internal/roles"
	"github.com/yoadey/team-manager/backend/internal/server"
	"github.com/yoadey/team-manager/backend/internal/stats"
	"github.com/yoadey/team-manager/backend/internal/storage"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// Build metadata; version is overridable via -ldflags "-X main.version=...".
var (
	version     = "dev"
	serviceName = "team-manager-backend"
)

// requestTimeout bounds how long chimiddleware.Timeout lets a request's
// context stay alive before it's considered timed out.
//
// httpWriteTimeout must exceed it with real margin, not equal it: chi's
// Timeout doesn't preemptively abort a running handler -- it derives a
// context deadline, calls next.ServeHTTP synchronously, and only in a defer
// AFTER that returns checks ctx.Err() and writes 504. net/http's
// WriteTimeout is reset when the request's headers are read, so with both
// set to the same value they'd expire within microseconds of each other on
// a legitimately slow request (DB contention, a long sequential chain like
// GetFinanceOverview's queries) -- chi's deferred 504 write can lose that
// race against the connection's write deadline already having elapsed,
// silently dropping the well-formed RFC 9457 error body in favor of a bare
// connection reset. The margin gives that deferred write room to actually
// flush.
const (
	requestTimeout   = 30 * time.Second
	httpWriteTimeout = requestTimeout + 10*time.Second
)

// initObservability wires optional OTel tracing and Sentry error tracking and
// returns a single cleanup that shuts both down. Both are no-ops when their
// env (OTEL_EXPORTER_OTLP_ENDPOINT / SENTRY_DSN) is unset.
func initObservability(ctx context.Context, cfg *config.Config) (func(context.Context), error) {
	tracedServiceName := serviceName
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		tracedServiceName = v
	}
	shutdownTracer, err := observability.InitTracer(ctx, tracedServiceName, version)
	if err != nil {
		return nil, fmt.Errorf("initObservability: %w", err)
	}
	flushSentry, err := observability.InitSentry(cfg.SentryDSN, os.Getenv("ENVIRONMENT"), version)
	if err != nil {
		return nil, fmt.Errorf("initObservability: %w", err)
	}
	return func(c context.Context) {
		if shutErr := shutdownTracer(c); shutErr != nil {
			slog.Error("tracer shutdown error", "err", shutErr)
		}
		flushSentry(2 * time.Second)
	}, nil
}

// failIfMetricsOpen exits with an error when COOKIE_SECURE=true but METRICS_TOKEN is
// unset. Exposing raw Prometheus metrics on an unauthenticated endpoint in production
// leaks internal operational data (user counts, login rates, DB pool state). Either set
// METRICS_TOKEN or restrict /metrics at the network layer and set METRICS_ALLOW_OPEN=true.
func failIfMetricsOpen(cfg *config.Config) {
	if cfg.MetricsToken == "" && cfg.CookieSecure {
		if os.Getenv("METRICS_ALLOW_OPEN") == "true" {
			slog.Warn("METRICS_TOKEN is not set; /metrics is unauthenticated (METRICS_ALLOW_OPEN=true overrides)")
			return
		}
		slog.Error("METRICS_TOKEN must be set when COOKIE_SECURE=true; set METRICS_TOKEN or METRICS_ALLOW_OPEN=true to allow open metrics")
		os.Exit(1)
	}
}

// warnIfPaginationKeyOpen logs (does not fail startup — impact is limited to
// clients being able to craft their own list-ordering cursor within data
// they can already read) when COOKIE_SECURE=true but PAGINATION_HMAC_KEY is
// unset, unlike JWT/cookie keys which are hard-required in that case.
func warnIfPaginationKeyOpen(cfg *config.Config) {
	if len(cfg.PaginationHMACKey) == 0 && cfg.CookieSecure {
		slog.Warn("PAGINATION_HMAC_KEY is not set; pagination cursors are unsigned in production")
	}
}

// gomemlimitHeadroomFactor scales GOMEMLIMIT down before handing it to the
// Go runtime's soft memory limit, leaving headroom below the container's
// hard cgroup memory limit. GOMEMLIMIT only accounts for memory the Go
// runtime tracks (heap + goroutine stacks), not other RSS contributors
// (mmap'd regions, cgo allocations, thread stacks) -- setting it equal to
// the hard limit means a load spike that grows non-heap RSS even slightly
// can still trip the kubelet's OOMKill before the GC pacer's soft-limit
// response gets a chance to react, undermining the exact protection
// GOMEMLIMIT exists to provide (see deployment.yaml's env block comment).
const gomemlimitHeadroomFactor = 0.9

// applyMemoryLimitHeadroom re-applies GOMEMLIMIT (set by the Helm chart from
// the downward API as a raw byte count -- limits.memory with no divisor,
// see deployment.yaml) scaled down by gomemlimitHeadroomFactor. Kubernetes'
// resourceFieldRef has no built-in percentage option, so the Go runtime's
// automatic env-var handling alone can't add this margin; calling
// debug.SetMemoryLimit here overrides whatever it applied at process init.
// A no-op when GOMEMLIMIT is unset or unparseable (local dev, tests, or a
// future non-numeric GOMEMLIMIT format like "256MiB" that this intentionally
// doesn't attempt to parse -- the chart only ever sets the raw-byte form).
func applyMemoryLimitHeadroom() {
	raw := os.Getenv("GOMEMLIMIT")
	if raw == "" {
		return
	}
	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || limit <= 0 {
		return
	}
	debug.SetMemoryLimit(int64(float64(limit) * gomemlimitHeadroomFactor))
}

// metricsHandler exposes Prometheus metrics, requiring a bearer token when one
// is configured. Left open when token is empty so local dev and in-cluster
// scraping over a private network keep working.
func metricsHandler(token string) http.Handler {
	h := promhttp.Handler()
	if token != "" {
		return middleware.RequireBearerToken(token)(h)
	}
	return h
}

// runHTTPServer starts the HTTP server and blocks until it exits. Only
// non-close errors are logged — ErrServerClosed is the expected shutdown path.
func runHTTPServer(srv *http.Server) {
	slog.Info("server starting", "port", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// initAuthComponents constructs the auth handler and session codec from config.
// Extracted to keep main()'s cyclomatic complexity within acceptable bounds.
func initAuthComponents(
	pool *pgxpool.Pool,
	cfg *config.Config,
	objectStore storage.ObjectStore,
	mailSender mailer.Mailer,
	logger *slog.Logger,
	auditLogger *audit.Logger,
) (*auth.Handler, *auth.SessionCookieCodec, error) {
	if cfg.JWTPrivateKey == "" && cfg.JWTPublicKey == "" {
		slog.Warn("JWT_PRIVATE_KEY/JWT_PUBLIC_KEY not set; generating an ephemeral RSA key pair for this process — sessions will not survive a restart and won't verify across replicas")
	}
	repo := auth.NewRepository(pool)
	svc, err := auth.NewService(repo, objectStore, cfg.JWTPrivateKey, cfg.JWTPublicKey, cfg.SessionTTL, auth.RegistrationConfig{
		Mailer:                  mailSender,
		PublicBaseURL:           cfg.PublicBaseURL,
		EmailVerificationTTL:    cfg.EmailVerificationTTL,
		SelfRegistrationEnabled: cfg.SelfRegistrationEnabled,
	}, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("auth service: %w", err)
	}
	codec, err := auth.NewSessionCookieCodec(cfg.CookieEncryptionKeys, cfg.CookieSecure, cfg.SessionTTL, cfg.CookieName)
	if err != nil {
		return nil, nil, fmt.Errorf("cookie codec: %w", err)
	}
	return auth.NewHandler(svc, logger, codec, auditLogger), codec, nil
}

// initMailer constructs the Mailer used to send self-registration
// verification email. Falls back to an in-memory fake (logs the link) when
// SMTP_HOST is unset -- config.Load() already hard-requires it when
// COOKIE_SECURE=true, so this path is only reachable in dev/test, mirroring
// the JWT/cookie-key/object-store ephemeral fallbacks above.
func initMailer(cfg *config.Config, logger *slog.Logger) mailer.Mailer {
	if cfg.SMTPHost == "" {
		slog.Warn("SMTP_HOST not set; using a logging fake mailer — verification emails will only appear in the server log")
		return mailer.NewFakeMailer(logger)
	}
	m, err := mailer.NewSMTPMailer(mailer.SMTPConfig{
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		FromAddress: cfg.SMTPFromAddress,
	})
	if err != nil {
		slog.Error("mailer init failed", "err", err)
		os.Exit(1)
	}
	return m
}

// initObjectStore constructs the ObjectStore used for team/user image
// uploads. Falls back to an in-memory fake when S3_ENDPOINT is unset --
// config.Load() already hard-requires it when COOKIE_SECURE=true, so this
// path is only reachable in dev/test, mirroring the JWT/cookie-key ephemeral
// fallback above.
func initObjectStore(cfg *config.Config) storage.ObjectStore {
	if cfg.S3Endpoint == "" {
		slog.Warn("S3_ENDPOINT not set; using an in-memory fake object store — uploaded images will not persist across restarts and are not shared across replicas")
		return storage.NewFakeStore()
	}
	s3Store, err := storage.NewS3Store(storage.S3Config{
		Endpoint:        cfg.S3Endpoint,
		Region:          cfg.S3Region,
		Bucket:          cfg.S3Bucket,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretAccessKey,
		UsePathStyle:    cfg.S3UsePathStyle,
		PublicBaseURL:   cfg.S3PublicBaseURL,
	})
	if err != nil {
		slog.Error("object store init failed", "err", err)
		os.Exit(1)
	}
	return s3Store
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	// ─── Config ──────────────────────────────────────────────────────────────
	// Loaded before the logger so its level can be configured by LOG_LEVEL;
	// config.Load() never logs internally, only returns an error, and the
	// stdlib's own default slog logger (text-format to stderr) is usable
	// even before SetDefault runs, so this ordering doesn't lose the
	// failure message below.

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	// Registered here, before any slow startup work (DB connect/migrate,
	// River migration -- which holds a Postgres advisory lock for its
	// duration), rather than just before the blocking <-quit receive below.
	// Until signal.Notify runs, SIGTERM has its OS default disposition
	// (immediate termination) -- Go installs no handler for it automatically
	// -- so a rolling deployment or node drain sending SIGTERM while this
	// process is still mid-init would kill it outright, skipping the app's
	// own graceful-shutdown path (and its "shutting down" log line, Sentry/
	// OTel flush) entirely. The channel is buffered (size 1), so a signal
	// arriving during init is queued and simply picked up by <-quit as soon
	// as init finishes, instead of being lost.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	applyMemoryLimitHeadroom()
	failIfMetricsOpen(cfg)
	warnIfPaginationKeyOpen(cfg)

	// ─── Observability ───────────────────────────────────────────────────────
	// Tracing and error tracking are opt-in (OTEL_EXPORTER_OTLP_ENDPOINT /
	// SENTRY_DSN); both are no-ops when their env is unset.
	shutdownObs, err := initObservability(context.Background(), cfg)
	if err != nil {
		slog.Error("observability init failed", "err", err)
		os.Exit(1)
	}

	// ─── Database ─────────────────────────────────────────────────────────────

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	prometheus.MustRegister(db.NewPoolStatsCollector(pool))

	if err := db.RunMigrations(ctx, pool, cfg.MigrationsDir); err != nil {
		pool.Close()
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	if err := jobs.MigrateRiver(ctx, pool); err != nil {
		pool.Close()
		slog.Error("river migrations failed", "err", err)
		os.Exit(1)
	}

	if *migrateOnly {
		slog.Info("migrations complete, exiting")
		pool.Close()
		os.Exit(0)
	}

	// ─── River job queue ──────────────────────────────────────────────────────

	retentionWorker := jobs.NewRetentionWorker(pool, cfg.RetentionNotificationDays, cfg.RetentionSessionDays, cfg.RetentionAuditLogDays, cfg.RetentionUnverifiedAccountDays)
	jobsClient, riverClient, err := jobs.NewClient(pool, retentionWorker)
	if err != nil {
		slog.Error("river client init failed", "err", err)
		os.Exit(1)
	}
	if err := riverClient.Start(ctx); err != nil {
		slog.Error("river worker start failed", "err", err)
		os.Exit(1)
	}

	// ─── Audit logger ────────────────────────────────────────────────────────
	// Writes to both structured log (stdout) and the audit_log DB table so that
	// records survive log rotation and can be queried for compliance review.
	auditLogger := audit.New(logger).WithDB(pool)

	// ─── Object storage ──────────────────────────────────────────────────────
	// Shared by auth (user photos), teams (team photo/logo) and members (member
	// photo delivery).

	objectStore := initObjectStore(cfg)

	// ─── Auth ────────────────────────────────────────────────────────────────

	mailSender := initMailer(cfg, logger)
	authHandler, cookieCodec, err := initAuthComponents(pool, cfg, objectStore, mailSender, logger, auditLogger)
	if err != nil {
		slog.Error("auth init failed", "err", err)
		os.Exit(1)
	}

	// ─── Pagination ──────────────────────────────────────────────────────────

	// A single Paginator is shared across all services that produce/consume
	// keyset cursors. When PAGINATION_HMAC_KEY is set, cursors are HMAC-signed
	// so that clients cannot craft arbitrary cursor values.
	pager := pagination.New(cfg.PaginationHMACKey)

	// ─── Teams ───────────────────────────────────────────────────────────────

	teamsRepo := teams.NewRepository(pool)
	teamsSvc := teams.NewService(teamsRepo, objectStore, cfg.PublicBaseURL)
	teamsHandler := teams.NewHandler(teamsSvc, logger, auditLogger)

	// ─── Members ─────────────────────────────────────────────────────────────

	membersRepo := members.NewRepository(pool)
	membersSvc := members.NewService(membersRepo, objectStore, pager)
	membersHandler := members.NewHandler(membersSvc, logger, auditLogger)

	// ─── Roles ───────────────────────────────────────────────────────────────

	rolesRepo := roles.NewRepository(pool)
	rolesSvc := roles.NewService(rolesRepo)
	rolesHandler := roles.NewHandler(rolesSvc, logger, auditLogger)

	// ─── Events ──────────────────────────────────────────────────────────────

	eventsRepo := events.NewRepository(pool)
	eventsSvc := events.NewService(eventsRepo, jobsClient, pager, rolesRepo, membersRepo, logger)
	eventsHandler := events.NewHandler(eventsSvc, logger)

	// ─── Absences ────────────────────────────────────────────────────────────

	absencesRepo := absences.NewRepository(pool)
	absencesSvc := absences.NewService(absencesRepo, pager)
	absencesHandler := absences.NewHandler(absencesSvc, logger)

	// ─── News ────────────────────────────────────────────────────────────────

	newsRepo := news.NewRepository(pool)
	newsSvc := news.NewService(newsRepo, jobsClient, pager, logger)
	newsHandler := news.NewHandler(newsSvc, logger)

	// ─── Polls ───────────────────────────────────────────────────────────────

	pollsRepo := polls.NewRepository(pool)
	pollsSvc := polls.NewService(pollsRepo, jobsClient, pager, logger)
	pollsHandler := polls.NewHandler(pollsSvc, logger)

	// ─── Notifications ────────────────────────────────────────────────────────

	notifRepo := notifications.NewRepository(pool)
	notifSvc := notifications.NewService(notifRepo, membersRepo)
	notifHandler := notifications.NewHandler(notifSvc, logger)

	// ─── Finances ─────────────────────────────────────────────────────────────

	financesRepo := finances.NewRepository(pool)
	financesSvc := finances.NewService(financesRepo, pager, logger)
	financesHandler := finances.NewHandler(financesSvc, logger, auditLogger)

	// ─── Stats ───────────────────────────────────────────────────────────────

	statsRepo := stats.NewRepository(pool)
	statsSvc := stats.NewService(statsRepo)
	statsHandler := stats.NewHandler(statsSvc, logger)

	// ─── Aggregated server ───────────────────────────────────────────────────

	srv := server.New(
		authHandler,
		teamsHandler,
		membersHandler,
		rolesHandler,
		eventsHandler,
		absencesHandler,
		newsHandler,
		pollsHandler,
		notifHandler,
		financesHandler,
		statsHandler,
	)

	// Wrap the strict server in the generated strict handler adapter. The cookie
	// middleware sets the encrypted session cookie on Login and clears it on Logout.
	// Custom error handlers render *apierror.APIError as RFC 9457 Problem
	// Details instead of the library default, which would write err.Error()
	// as a plain-text 500 for every handler error (wrong status code, and a
	// potential leak of wrapped internal error details).
	strictSrv := gen.NewStrictHandlerWithOptions(
		srv,
		[]gen.StrictMiddlewareFunc{cookieCodec.StrictMiddleware()},
		gen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  apierror.RequestErrorHandler(logger),
			ResponseErrorHandlerFunc: apierror.ResponseErrorHandler(logger),
		},
	)

	// ─── Router ──────────────────────────────────────────────────────────────

	trustedProxies, err := middleware.ParseTrustedProxies(cfg.TrustedProxyCIDRs)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXY_CIDRS", "err", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	// Global middleware (applied to all routes, in order).
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Recoverer(logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Metrics)
	r.Use(chimiddleware.Timeout(requestTimeout))
	// RateLimit runs before CORS so that OPTIONS preflight requests are also
	// counted against the global limit — CORS answers OPTIONS itself and
	// never calls next, which would otherwise let preflight traffic bypass
	// every middleware registered after it.
	r.Use(middleware.RateLimit(cfg.RateLimitRPS, trustedProxies))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.CSRFOriginCheck(cfg.AllowedOrigins))
	r.Use(middleware.BodyLimit(4 << 20)) // 4 MB default body limit
	r.Use(middleware.APIVersion("v1"))

	// Internal endpoints (no auth, no external prefix).
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "readyz db ping failed", "err", err)
			http.Error(w, `{"status":"unavailable","db":"down"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Handle("/metrics", metricsHandler(cfg.MetricsToken))

	// API routes under /api/v1.
	r.Route("/api/v1", func(r chi.Router) {
		// Authenticated endpoints. The generated mux registers every route,
		// including /auth/login and /auth/providers; the public overrides below
		// are registered afterwards so they win (chi: last registration wins).
		r.Group(func(r chi.Router) {
			r.Use(authHandler.AuthMiddleware)
			// Team-scoped endpoints additionally require the caller to be a member.
			r.Use(middleware.RequireMembership(membersRepo))
			r.Use(middleware.RequirePermission(membersRepo))
			gen.HandlerFromMuxWithBaseURL(strictSrv, r, "")
		})

		// Public auth endpoints — no JWT required. Must be registered AFTER the
		// generated mux above to override its authenticated duplicates.
		// Per-IP brute-force protection: max 5 login attempts per minute.
		r.With(middleware.PerIPRateLimit(cfg.LoginRateLimitPerMin, time.Minute, trustedProxies)).Post("/auth/login", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.Login(w, req)
		})
		r.Get("/auth/providers", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.ListProviders(w, req)
		})
		// Self-registration and its verification endpoints are rate-limited the
		// same way as login -- each is a plausible target for volumetric abuse
		// (account-creation spam / verification-token brute-forcing / mail-bomb
		// via resend). verify-email itself is not separately rate-limited: its
		// token is a high-entropy, single-use secret, not a guessable value.
		r.With(middleware.PerIPRateLimit(cfg.RegisterRateLimitPerMin, time.Minute, trustedProxies)).Post("/auth/register", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.Register(w, req)
		})
		r.Post("/auth/verify-email", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.VerifyEmail(w, req)
		})
		r.With(middleware.PerIPRateLimit(cfg.ResendVerificationRateLimitPerMin, time.Minute, trustedProxies)).Post("/auth/resend-verification", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.ResendVerification(w, req)
		})
	})

	// ─── HTTP server ─────────────────────────────────────────────────────────

	// Wrap the router so every request gets an OpenTelemetry server span and
	// incoming trace context is propagated (no-op when tracing is disabled).
	httpSrv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelhttp.NewHandler(r, "http.server"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go runHTTPServer(httpSrv)

	// ─── Graceful shutdown ───────────────────────────────────────────────────
	// quit was registered near the top of main(), before any slow startup
	// work -- see that declaration's comment for why.

	<-quit

	slog.Info("shutting down server")
	// Each phase gets its own fresh timeout budget rather than sharing one
	// deadline: without jobs.SoftStopTimeout configured, river.Client.Stop
	// would return as soon as its own context is done WITHOUT cancelling a
	// still-running job (it just keeps executing, holding a pool connection,
	// for up to its own Timeout() budget -- RetentionWorker's is 150s), so a
	// slow HTTP drain eating into a shared deadline would silently skip job
	// draining before pool.Close() ever got a chance to matter.
	httpShutdownCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer httpCancel()
	if err := httpSrv.Shutdown(httpShutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	// Stop the job worker (draining in-flight jobs) before closing the pool it
	// depends on — must happen in this order, not as an unbounded defer
	// registered near Start, which would otherwise run after pool.Close().
	// jobs.SoftStopTimeout (configured on the river.Client in jobs.NewClient)
	// gives running jobs a chance to finish on their own, then automatically
	// cancels their contexts if that timeout elapses -- so Stop() reliably
	// returns within roughly jobs.SoftStopTimeout, not the much larger
	// worst-case job Timeout(). The margin here just covers Stop() actually
	// observing that cancellation and returning.
	riverStopCtx, riverCancel := context.WithTimeout(context.Background(), jobs.SoftStopTimeout+7*time.Second)
	defer riverCancel()
	if err := riverClient.Stop(riverStopCtx); err != nil {
		slog.Error("river worker stop failed", "err", err)
	}
	obsShutdownCtx, obsCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer obsCancel()
	shutdownObs(obsShutdownCtx)
	pool.Close()
}
