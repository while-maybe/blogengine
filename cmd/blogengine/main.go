package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"blogengine/internal/content"
	"blogengine/internal/handlers"
)

type App struct {
	Repo   *content.Repository
	Server *http.Server
	Logger *slog.Logger
}

func NewApp(blogTitle string, sourcesDir string, logger *slog.Logger, writer io.Writer) (*App, error) {

	repo, err := content.NewRepository(blogTitle)
	if err != nil {
		return nil, fmt.Errorf("could not create repository: %w", err)
	}

	wantedFiles := "*.md"
	files, err := filepath.Glob(filepath.Join(sourcesDir, wantedFiles))
	if err != nil {
		return nil, fmt.Errorf("failed to scan data sources: %w", err)
	}
	repo.LoadLazyMetaFromDisk(files)

	h := handlers.NewBlogHandler(repo, blogTitle, logger)

	mux := http.NewServeMux()

	// static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	// routes
	mux.Handle("GET /", h.HandleIndex())
	mux.Handle("GET /post/", h.HandlePost())
	mux.Handle("GET /metrics", h.HandleMetrics())

	server := &http.Server{
		Addr:         ":3000",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		Repo:   repo,
		Server: server,
		Logger: logger,
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
	srvCleanupDuration := 10 * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), srvCleanupDuration)
	defer cancel()

	a.Logger.Info("draining connections...")
	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	a.Logger.Info("server stopped")
	return nil
}

func main() {
	stderr := os.Stderr
	const blogTitle = "Strange coding blog"

	logHandler := slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(logHandler).With("app", blogTitle)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// initialise
	app, err := NewApp(blogTitle, "./sources", logger, stderr)
	if err != nil {
		logger.Error("server initialise", "err", err)
		os.Exit(1)
	}

	// run the app with context
	if err := app.Run(rootCtx); err != nil {
		logger.Error("server crashed", "err", err)
		os.Exit(1)
	}
}
