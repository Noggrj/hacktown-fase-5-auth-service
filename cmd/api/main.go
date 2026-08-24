package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	authhttp "github.com/noggrj/fiapx-auth-service/internal/auth/delivery/http"
	"github.com/noggrj/fiapx-auth-service/internal/auth/gateway"
	"github.com/noggrj/fiapx-auth-service/internal/auth/usecase"

	"github.com/noggrj/fiapx-auth-service/internal/platform/config"
	"github.com/noggrj/fiapx-auth-service/internal/platform/db"
	"github.com/noggrj/fiapx-auth-service/internal/platform/health"
	"github.com/noggrj/fiapx-auth-service/internal/platform/jwt"
	"github.com/noggrj/fiapx-auth-service/internal/platform/logging"
	"github.com/noggrj/fiapx-auth-service/internal/platform/metrics"
)

var version = "dev"

func main() {
	// CI smoke test runs `./server --version` against a container with no
	// DB reachable — must exit immediately instead of falling through to
	// ListenAndServe, which blocks forever.
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	cfg := config.Load()
	log := logging.New(cfg.ServiceName)
	slog.SetDefault(log)

	log.Info("starting auth-service",
		slog.String("version", version),
		slog.String("env", cfg.Env),
		slog.String("port", cfg.Port),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.JWTSecret == "" {
		log.Error("JWT_SECRET is required")
		os.Exit(1)
	}
	issuer, err := jwt.NewIssuer(cfg.JWTSecret)
	if err != nil {
		log.Error("invalid jwt secret", slog.Any("error", err))
		os.Exit(1)
	}
	verifier, err := jwt.NewVerifier(cfg.JWTSecret)
	if err != nil {
		log.Error("invalid jwt secret", slog.Any("error", err))
		os.Exit(1)
	}

	// ── Postgres ────────────────────────────────────────────────
	pool, err := db.NewPool(ctx, cfg.DBURL)
	if err != nil {
		log.Warn("postgres unavailable at startup — /ready will be degraded", slog.Any("error", err))
	} else {
		log.Info("postgres connected")
		defer pool.Close()
	}

	var userRepo *gateway.PostgresUserRepository
	if pool != nil {
		userRepo = gateway.NewPostgresUserRepository(pool.Pool)
	}

	var (
		registerUC *usecase.RegisterUseCase
		loginUC    *usecase.LoginUseCase
	)
	if userRepo != nil {
		registerUC = usecase.NewRegister(userRepo, log)
		loginUC = usecase.NewLogin(userRepo, issuer, log)
	}

	// ── Health probes ──────────────────────────────────────────
	probes := map[string]health.Probe{
		"self": func() health.Check { return health.Check{Status: "healthy"} },
		"postgres": func() health.Check {
			if pool == nil {
				return health.Check{Status: "unhealthy", Detail: "pool not initialized"}
			}
			if err := pool.Healthy(ctx); err != nil {
				return health.Check{Status: "unhealthy", Detail: err.Error()}
			}
			return health.Check{Status: "healthy"}
		},
	}
	hh := health.New(version, probes)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer, metrics.Middleware)
	r.Get("/health", hh.Live)
	r.Get("/ready", hh.Ready)
	r.Handle("/metrics", metrics.Handler())

	if registerUC != nil && loginUC != nil {
		handler := authhttp.NewHandler(registerUC, loginUC, verifier, log)
		handler.Register(r)
	}

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	idle := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", slog.Any("error", err))
		}
		close(idle)
	}()
	log.Info("server listening", slog.String("port", cfg.Port))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}
	<-idle
	log.Info("shutdown complete")
}
