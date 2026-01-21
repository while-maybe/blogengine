package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"blogengine/content"
	"blogengine/handlers"
)

type App struct {
	Repo   *content.Repository
	Server *http.Server
	Logger *slog.Logger
	stderr io.Writer
}

// TODO investigate Go library Goldmark

func NewApp(blogTitle string, sourcesDir string, logger *slog.Logger, writer io.Writer) (*App, error) {

	repo := content.NewPosts(blogTitle)

	wantedFiles := "*.md"
	files, err := filepath.Glob(filepath.Join(sourcesDir, wantedFiles))
	if err != nil {
		return nil, fmt.Errorf("failed to scan sources: %w", err)
	}
	repo.LoadLazyMetaFromDisk(files)

	h := handlers.NewBlogHandler(repo, blogTitle)

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
		WriteTimeout: 10 * time.Second,
	}

	return &App{
		Repo:   repo,
		Server: server,
		Logger: logger,
		stderr: writer,
	}, nil
}

func (a *App) Run() error {
	a.Logger.Info("server starting", "addr", a.Server.Addr)
	return a.Server.ListenAndServe()
}

func main() {
	stderr := os.Stderr
	const blogTitle = "Strange coding blog"

	logHandler := slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(logHandler).With("app", blogTitle)

	app, err := NewApp(blogTitle, "./sources", logger, stderr)
	if err != nil {
		logger.Error("server initialise", "err", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		logger.Error("server crashed", "err", err)
		os.Exit(1)
	}
}
