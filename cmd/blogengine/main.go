package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"blogengine/internal/config"
	"blogengine/internal/content"
	"blogengine/internal/handlers"
	"blogengine/internal/middleware"
)

type App struct {
	Repo   *content.Repository
	Server *http.Server
	Logger *slog.Logger
	Config *config.Config
}

func NewApp(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {

	repo, err := content.NewRepository(cfg.App.Name)
	if err != nil {
		return nil, fmt.Errorf("could not create repository: %w", err)
	}

	wantedFiles := "*.md"
	files, err := filepath.Glob(filepath.Join(cfg.App.SourcesDir, wantedFiles))
	if err != nil {
		return nil, fmt.Errorf("failed to scan data sources: %w", err)
	}
	repo.LoadLazyMetaFromDisk(files)

	limiter := middleware.NewIPRateLimiter(ctx, cfg.Limiter.RPS, cfg.Limiter.Burst, cfg.Proxy.Trusted)

	geo := middleware.NewGeoStats(ctx)

	h := handlers.NewBlogHandler(repo, cfg.App.Name, logger, geo)

	mux := http.NewServeMux()

	// static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	// routes
	mux.Handle("GET /", h.HandleIndex())
	mux.Handle("GET /post/", h.HandlePost())
	mux.Handle("GET /metrics", h.HandleMetrics())

	defaultMiddlewareStack := []middleware.Middleware{
		middleware.Recover(logger),
		limiter.Middleware(logger),
		middleware.Logger(logger),
		geo.Middleware(logger),
	}

	handleChain := middleware.Chain(mux, defaultMiddlewareStack...)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handleChain,
		ReadTimeout:  cfg.HTTP.Timeouts.Read,
		WriteTimeout: cfg.HTTP.Timeouts.Write,
		IdleTimeout:  cfg.HTTP.Timeouts.Idle,
	}

	return &App{
		Repo:   repo,
		Server: server,
		Logger: logger,
		Config: cfg,
	}, nil
}

func (a *App) Run(ctx context.Context) error {

	srvErrChan := make(chan error, 1)

	go func() {
		a.Logger.Info("server starting", "addr", a.Server.Addr)
		if err := a.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrChan <- err
		}
	}()

	select {
	case err := <-srvErrChan:
		return fmt.Errorf("server startup failed: %w", err)
	case <-ctx.Done():
		a.Logger.Info("shutdown signal received")
	}

	// attempt clean shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.Config.HTTP.Timeouts.Shutdown)
	defer cancel()

	a.Logger.Info("draining connections...")
	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		// graceful shutdown timed out
		if closeErr := a.Server.Close(); closeErr != nil {
			// both failed. Return combined error.
			return fmt.Errorf("graceful shutdown failed: %w", errors.Join(err, closeErr))
		}
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	a.Logger.Info("server stopped")
	return nil
}

func main() {
	stderr := os.Stderr

	cfg := config.LoadWithDefaults()
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}

	logHandler := slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: cfg.Logger.Level})
	logger := slog.New(logHandler).With("app", cfg.App.Name)

	// Add PID to this log line
	logger.Info("application starting", "pid", os.Getpid())
	logger.Info("configuration loaded",
		"name", cfg.App.Name,
		"sources", cfg.App.SourcesDir,
		"env", cfg.App.Environment,
		"port", cfg.HTTP.Port,
		"rate_limit_rps", cfg.Limiter.RPS,
		"trusted_proxy", cfg.Proxy.Trusted,
		// Do NOT log cfg.Proxy.Token!
	)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// initialise
	app, err := NewApp(rootCtx, cfg, logger)
	if err != nil {
		logger.Error("server initialise", "err", err)
		os.Exit(1)
	}

	// run the app with context
	if err := app.Run(rootCtx); err != nil {
		logger.Error("server crashed", "err", err)
		os.Exit(1)
	}

	logger.Info("application exited successfully")
	os.Exit(0)
}
