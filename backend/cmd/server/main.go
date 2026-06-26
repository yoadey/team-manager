package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yoadey/team-manager/backend/internal/absences"
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
	"github.com/yoadey/team-manager/backend/internal/polls"
	"github.com/yoadey/team-manager/backend/internal/roles"
	"github.com/yoadey/team-manager/backend/internal/server"
	"github.com/yoadey/team-manager/backend/internal/stats"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// ─── Config ──────────────────────────────────────────────────────────────

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
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

	// ─── River job queue ──────────────────────────────────────────────────────

	jobsClient, riverClient, err := jobs.NewClient(pool)
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

	// ─── Auth ────────────────────────────────────────────────────────────────

	authRepo := auth.NewRepository(pool)
	authSvc, err := auth.NewService(authRepo, cfg.JWTPrivateKey, cfg.JWTPublicKey, cfg.SessionTTL)
	if err != nil {
		slog.Error("auth service init failed", "err", err)
		os.Exit(1) //nolint:gocritic
	}
	cookieCodec, err := auth.NewSessionCookieCodec(cfg.CookieEncryptionKey, cfg.CookieSecure, cfg.SessionTTL, cfg.CookieName)
	if err != nil {
		slog.Error("cookie codec init failed", "err", err)
		os.Exit(1)
	}
	authHandler := auth.NewHandler(authSvc, logger, cookieCodec)

	// ─── Teams ───────────────────────────────────────────────────────────────

	teamsRepo := teams.NewRepository(pool)
	teamsSvc := teams.NewService(teamsRepo, cfg.PublicBaseURL)
	teamsHandler := teams.NewHandler(teamsSvc, logger)

	// ─── Members ─────────────────────────────────────────────────────────────

	membersRepo := members.NewRepository(pool)
	membersSvc := members.NewService(membersRepo)
	membersHandler := members.NewHandler(membersSvc, logger)

	// ─── Roles ───────────────────────────────────────────────────────────────

	rolesRepo := roles.NewRepository(pool)
	rolesSvc := roles.NewService(rolesRepo)
	rolesHandler := roles.NewHandler(rolesSvc, logger)

	// ─── Events ──────────────────────────────────────────────────────────────

	eventsRepo := events.NewRepository(pool)
	eventsSvc := events.NewService(eventsRepo, jobsClient)
	eventsHandler := events.NewHandler(eventsSvc, logger)

	// ─── Absences ────────────────────────────────────────────────────────────

	absencesRepo := absences.NewRepository(pool)
	absencesSvc := absences.NewService(absencesRepo)
	absencesHandler := absences.NewHandler(absencesSvc, logger)

	// ─── News ────────────────────────────────────────────────────────────────

	newsRepo := news.NewRepository(pool)
	newsSvc := news.NewService(newsRepo, jobsClient)
	newsHandler := news.NewHandler(newsSvc, logger)

	// ─── Polls ───────────────────────────────────────────────────────────────

	pollsRepo := polls.NewRepository(pool)
	pollsSvc := polls.NewService(pollsRepo, jobsClient)
	pollsHandler := polls.NewHandler(pollsSvc, logger)

	// ─── Notifications ────────────────────────────────────────────────────────

	notifRepo := notifications.NewRepository(pool)
	notifSvc := notifications.NewService(notifRepo)
	notifHandler := notifications.NewHandler(notifSvc, logger)

	// ─── Finances ─────────────────────────────────────────────────────────────

	financesRepo := finances.NewRepository(pool)
	financesSvc := finances.NewService(financesRepo)
	financesHandler := finances.NewHandler(financesSvc, logger)

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
	r.Use(middleware.RateLimit(100))
	r.Use(middleware.BodyLimit(4 << 20)) // 4 MB default body limit

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
	r.Handle("/metrics", promhttp.Handler())

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
		r.With(middleware.PerIPRateLimit(5, time.Minute)).Post("/auth/login", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.Login(w, req)
		})
		r.Get("/auth/providers", func(w http.ResponseWriter, req *http.Request) {
			strictSrv.ListProviders(w, req)
		})
	})

	// ─── HTTP server ─────────────────────────────────────────────────────────

	httpSrv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

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
}
