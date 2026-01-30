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
	"blogengine/internal/router"
	"blogengine/internal/storage"
)

type App struct {
	Server *http.Server
	Logger *slog.Logger
	Config *config.Config
	Posts  content.PostService
	Media  content.MediaService
}

func NewApp(ctx context.Context, cfg *config.Config, logger *slog.Logger, posts content.PostService, media content.MediaService, handler http.Handler) (*App, error) {

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.Timeouts.Read,
		WriteTimeout: cfg.HTTP.Timeouts.Write,
		IdleTimeout:  cfg.HTTP.Timeouts.Idle,
	}

	return &App{
		Server: server,
		Logger: logger,
		Config: cfg,
		Posts:  posts,
		Media:  media,
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
	cfg := config.LoadWithDefaults()
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}

	stderr := os.Stderr
	logHandler := slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: cfg.Logger.Level})
	logger := slog.New(logHandler).With("app", cfg.App.Name)

	// Add PID
	logger.Info("application starting", "pid", os.Getpid())
	logger.Info("configuration loaded",
		"name", cfg.App.Name,
		"sources", cfg.App.SourcesDir,
		"env", cfg.App.Environment,
		"port", cfg.HTTP.Port,
		"rate_limit_rps", cfg.Limiter.RPS,
		"trusted_proxy", cfg.Proxy.Trusted,
	)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	storageProvider := storage.NewLocalStorage(cfg.App.SourcesDir)
	assetManager := content.NewAssetManager(storageProvider)

	repo, err := content.NewLocalRepository(cfg.App.Name)
	if err != nil {
		logger.Error("could not create repository", "err", err)
		os.Exit(1)
	}

	wantedFiles := "*.md"
	files, err := filepath.Glob(filepath.Join(cfg.App.SourcesDir, wantedFiles))
	if err != nil {
		logger.Error("failed to scan data sources", "err", err)
		os.Exit(1)
	}
	repo.LoadLazyMetaFromDisk(files)

	renderer := content.NewMarkDownRenderer(assetManager)

	limiter := middleware.NewIPRateLimiter(rootCtx, cfg.Limiter.RPS, cfg.Limiter.Burst, cfg.Proxy.Trusted)
	geo := middleware.NewGeoStats(rootCtx)

	blogHandler := handlers.NewBlogHandler(repo, renderer, cfg.App.Name, logger, geo)
	assetHandler := &handlers.AssetHandler{Assets: assetManager}

	router := router.NewRouter(router.RouterDependencies{
		Cfg:          cfg,
		Logger:       logger,
		BlogHandler:  blogHandler,
		AssetHandler: assetHandler,
		Limiter:      limiter,
		GeoStats:     geo,
	})

	// initialise
	app, err := NewApp(rootCtx, cfg, logger, repo, assetManager, router)
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
