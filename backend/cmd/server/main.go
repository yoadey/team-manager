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
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/config"
	"github.com/yoadey/team-manager/backend/internal/db"
	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/finances"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
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
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// Build metadata; version is overridable via -ldflags "-X main.version=...".
var (
	version     = "dev"
	serviceName = "team-manager-backend"
)

// initObservability wires optional OTel tracing and Sentry error tracking and
// returns a single cleanup that shuts both down. Both are no-ops when their
// env (OTEL_EXPORTER_OTLP_ENDPOINT / SENTRY_DSN) is unset.
func initObservability(ctx context.Context, cfg *config.Config) (func(context.Context), error) {
	shutdownTracer, err := observability.InitTracer(ctx, serviceName, version)
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

// warnIfMetricsOpen logs a startup warning when the /metrics endpoint is
// unauthenticated in a production configuration (COOKIE_SECURE=true).
// In production the endpoint should be protected via METRICS_TOKEN or
// restricted at the network layer (private subnet, mTLS).
func warnIfMetricsOpen(cfg *config.Config) {
	if cfg.MetricsToken == "" && cfg.CookieSecure {
		slog.Warn("METRICS_TOKEN is not set; /metrics is unauthenticated — restrict access at the network layer or set METRICS_TOKEN")
	}
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
	logger *slog.Logger,
	auditLogger *audit.Logger,
) (*auth.Handler, *auth.SessionCookieCodec, error) {
	repo := auth.NewRepository(pool)
	svc, err := auth.NewService(repo, cfg.JWTPrivateKey, cfg.JWTPublicKey, cfg.SessionTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("auth service: %w", err)
	}
	codec, err := auth.NewSessionCookieCodec(cfg.CookieEncryptionKeys, cfg.CookieSecure, cfg.SessionTTL, cfg.CookieName)
	if err != nil {
		return nil, nil, fmt.Errorf("cookie codec: %w", err)
	}
	return auth.NewHandler(svc, logger, codec, auditLogger), codec, nil
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// ─── Config ──────────────────────────────────────────────────────────────

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	warnIfMetricsOpen(cfg)

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

	retentionWorker := jobs.NewRetentionWorker(pool, cfg.RetentionNotificationDays, cfg.RetentionSessionDays)
	jobsClient, riverClient, err := jobs.NewClient(pool, retentionWorker)
	if err != nil {
		slog.Error("river client init failed", "err", err)
		os.Exit(1)
	}
	if err := riverClient.Start(ctx); err != nil {
		slog.Error("river worker start failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := riverClient.Stop(ctx); err != nil {
			slog.Error("river worker stop failed", "err", err)
		}
	}()

	// ─── Audit logger ────────────────────────────────────────────────────────
	// Writes to both structured log (stdout) and the audit_log DB table so that
	// records survive log rotation and can be queried for compliance review.
	auditLogger := audit.New(logger).WithDB(pool)

	// ─── Auth ────────────────────────────────────────────────────────────────

	authHandler, cookieCodec, err := initAuthComponents(pool, cfg, logger, auditLogger)
	if err != nil {
		slog.Error("auth init failed", "err", err)
		os.Exit(1) //nolint:gocritic // exitAfterDefer: river client stop defer is non-critical cleanup
	}

	// ─── Pagination ──────────────────────────────────────────────────────────

	// A single Paginator is shared across all services that produce/consume
	// keyset cursors. When PAGINATION_HMAC_KEY is set, cursors are HMAC-signed
	// so that clients cannot craft arbitrary cursor values.
	pager := pagination.New(cfg.PaginationHMACKey)

	// ─── Teams ───────────────────────────────────────────────────────────────

	teamsRepo := teams.NewRepository(pool)
	teamsSvc := teams.NewService(teamsRepo, cfg.PublicBaseURL)
	teamsHandler := teams.NewHandler(teamsSvc, logger)

	// ─── Members ─────────────────────────────────────────────────────────────

	membersRepo := members.NewRepository(pool)
	membersSvc := members.NewService(membersRepo, pager)
	membersHandler := members.NewHandler(membersSvc, logger, auditLogger)

	// ─── Roles ───────────────────────────────────────────────────────────────

	rolesRepo := roles.NewRepository(pool)
	rolesSvc := roles.NewService(rolesRepo)
	rolesHandler := roles.NewHandler(rolesSvc, logger, auditLogger)

	// ─── Events ──────────────────────────────────────────────────────────────

	eventsRepo := events.NewRepository(pool)
	eventsSvc := events.NewService(eventsRepo, jobsClient, pager)
	eventsHandler := events.NewHandler(eventsSvc, logger)

	// ─── Absences ────────────────────────────────────────────────────────────

	absencesRepo := absences.NewRepository(pool)
	absencesSvc := absences.NewService(absencesRepo, pager)
	absencesHandler := absences.NewHandler(absencesSvc, logger)

	// ─── News ────────────────────────────────────────────────────────────────

	newsRepo := news.NewRepository(pool)
	newsSvc := news.NewService(newsRepo, jobsClient, pager)
	newsHandler := news.NewHandler(newsSvc, logger)

	// ─── Polls ───────────────────────────────────────────────────────────────

	pollsRepo := polls.NewRepository(pool)
	pollsSvc := polls.NewService(pollsRepo, jobsClient, pager)
	pollsHandler := polls.NewHandler(pollsSvc, logger)

	// ─── Notifications ────────────────────────────────────────────────────────

	notifRepo := notifications.NewRepository(pool)
	notifSvc := notifications.NewService(notifRepo)
	notifHandler := notifications.NewHandler(notifSvc, logger)

	// ─── Finances ─────────────────────────────────────────────────────────────

	financesRepo := finances.NewRepository(pool)
	financesSvc := finances.NewService(financesRepo)
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
	strictSrv := gen.NewStrictHandler(srv, []gen.StrictMiddlewareFunc{cookieCodec.StrictMiddleware()})

	// ─── Router ──────────────────────────────────────────────────────────────

	r := chi.NewRouter()

	// Global middleware (applied to all routes, in order).
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Recoverer(logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Metrics)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.CSRFOriginCheck(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(cfg.RateLimitRPS))
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
		r.With(middleware.PerIPRateLimit(cfg.LoginRateLimitPerMin, time.Minute)).Post("/auth/login", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.Login(w, req)
		})
		r.Get("/auth/providers", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.ListProviders(w, req)
		})
	})

	// ─── HTTP server ─────────────────────────────────────────────────────────

	// Wrap the router so every request gets an OpenTelemetry server span and
	// incoming trace context is propagated (no-op when tracing is disabled).
	httpSrv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      otelhttp.NewHandler(r, "http.server"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go runHTTPServer(httpSrv)

	// ─── Graceful shutdown ───────────────────────────────────────────────────

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	shutdownObs(shutdownCtx)
}
