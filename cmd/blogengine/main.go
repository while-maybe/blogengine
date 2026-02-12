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
	"runtime"
	"syscall"
	"time"

	"blogengine/internal/config"
	"blogengine/internal/content"
	"blogengine/internal/handlers"
	"blogengine/internal/middleware"
	"blogengine/internal/router"
	"blogengine/internal/storage"
	"blogengine/internal/storage/sqlite"
	"blogengine/internal/telemetry"

	"github.com/gofrs/uuid/v5"
)

type App struct {
	Server    *http.Server
	Logger    *slog.Logger
	Config    *config.Config
	Posts     content.PostService
	Media     content.MediaService
	DB        storage.Store
	Telemetry *telemetry.Telemetry
}

func NewApp(ctx context.Context, cfg *config.Config, logger *slog.Logger, posts content.PostService, media content.MediaService, handler http.Handler, db storage.Store, tel *telemetry.Telemetry) (*App, error) {

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  cfg.HTTP.Timeouts.Read,
		WriteTimeout: cfg.HTTP.Timeouts.Write,
		IdleTimeout:  cfg.HTTP.Timeouts.Idle,
	}

	return &App{
		Server:    server,
		Logger:    logger,
		Config:    cfg,
		Posts:     posts,
		Media:     media,
		DB:        db,
		Telemetry: tel,
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

	// reuse shutdownCtx here to give telemetry some time to finish
	if err := a.Telemetry.Shutdown(shutdownCtx); err != nil {
		a.Logger.Error("telemetry shutdown failed", "err", err)
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

	// TODO hardcoded version
	tel, err := telemetry.Init(rootCtx, cfg.App.Name, "1.0.0", cfg.App.Environment, cfg.Metrics.OtelEndpoint, cfg.Metrics.EnableTelemetry, logger)
	if err != nil {
		logger.Error("failed to init telemetry", "err", err)
		os.Exit(1)
	}

	metrics, err := telemetry.NewMetrics(tel.Meter)
	if err != nil {
		logger.Error("failed to create metrics", "err", err)
		os.Exit(1)
	}

	storageProvider := storage.NewLocalStorage(cfg.App.SourcesDir)

	ns := uuid.Must(uuid.FromString(cfg.App.AssetNamespace)) // has already been validated in config
	assetManager := content.NewAssetManager(storageProvider, ns)

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

	if len(repo.Data) == 0 {
		logger.Warn("no posts found", "path", cfg.App.SourcesDir, "pattern", wantedFiles)
	} else {
		logger.Info("posts loaded", "count", len(repo.Data))
		metrics.RecordPostsLoaded(rootCtx, len(repo.Data))
	}

	renderer := content.NewMarkDownRenderer(assetManager)

	db, err := sqlite.NewStore(cfg.DB.Path)
	if err != nil {
		logger.Error("failed to create database", "path", cfg.DB.Path, "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(cfg.DB.MigrationsPath); err != nil {
		logger.Error("db migration failed", "path", cfg.DB.MigrationsPath, "err", err)
		os.Exit(1)
	}

	logger.Info("database migrated successfully")

	// session manager
	sessionLifetime := 24 * time.Hour
	session := middleware.NewSessionManager(sessionLifetime, cfg.App.Environment == "prod", db.RawDB())

	limiter := middleware.NewIPRateLimiter(rootCtx, cfg.Limiter.RPS, cfg.Limiter.Burst, cfg.Proxy.Trusted)

	authRPS := 1
	authBurst := 3
	authLimiter := middleware.NewIPRateLimiter(rootCtx, authRPS, authBurst, cfg.Proxy.Trusted)

	geo := middleware.NewGeoStats(rootCtx)

	needsInvite := cfg.Auth.InviteCode != ""

	blogHandler := handlers.NewBlogHandler(repo, db, renderer, cfg.App.Name, needsInvite, cfg.Auth.SessionSecret, logger, geo, tel.Tracer, metrics, session)

	// cheap cheap vps?
	numProcs := max(1, runtime.GOMAXPROCS(0)-1)
	imgProcessor, err := content.NewProcessor(rootCtx, cfg.App.SourcesDir, numProcs, logger)
	if err != nil {
		logger.Error("failed to start image processor", "err", err)
		os.Exit(1)
	}

	assetHandler := &handlers.AssetHandler{Assets: assetManager, Processor: imgProcessor}

	csrf := middleware.NewCSRF(cfg.App.Environment == "prod")
	csp := middleware.NewCSP(cfg.App.Environment == "prod")

	routerDeps := router.RouterDependencies{
		Cfg:               cfg,
		Logger:            logger,
		BlogHandler:       blogHandler,
		AssetHandler:      assetHandler,
		Limiter:           limiter,
		AuthLimiter:       authLimiter,
		GeoStats:          geo,
		Tracer:            tel.Tracer,
		Metrics:           metrics,
		PrometheusHandler: tel.PrometheusHandler,
		Session:           session,
		CSRF:              csrf,
		CSP:               csp,
	}

	router := router.NewRouter(routerDeps)

	// initialise
	app, err := NewApp(rootCtx, cfg, logger, repo, assetManager, router, db, tel)
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
