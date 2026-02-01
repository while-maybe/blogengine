package router

import (
	"blogengine/internal/config"
	"blogengine/internal/handlers"
	"blogengine/internal/middleware"
	"blogengine/internal/telemetry"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

// RouterDependencies holds everything needed to register routes.
type RouterDependencies struct {
	Cfg               *config.Config
	Logger            *slog.Logger
	BlogHandler       *handlers.BlogHandler
	AssetHandler      *handlers.AssetHandler
	Limiter           *middleware.IPRateLimiter
	GeoStats          *middleware.GeoStats
	Tracer            trace.Tracer
	Metrics           *telemetry.Metrics
	PrometheusHandler http.Handler
}

func NewRouter(deps RouterDependencies) http.Handler {
	// routing
	mux := http.NewServeMux()

	if deps.PrometheusHandler != nil {
		mux.Handle("GET /metrics/prometheus", deps.PrometheusHandler)
	}

	// static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.Handle("GET /assets/", deps.AssetHandler)

	// routes
	mux.Handle("GET /", deps.BlogHandler.HandleIndex())
	mux.Handle("GET /post/", deps.BlogHandler.HandlePost())
	mux.Handle("GET /metrics", deps.BlogHandler.HandleMetrics())

	middlewareStack := []middleware.Middleware{
		middleware.Recover(deps.Logger),
	}

	if deps.Cfg.Metrics.EnableTelemetry {
		// order matters so don't append
		middlewareStack = append(middlewareStack, middleware.Observability(deps.Tracer, deps.Metrics, deps.Logger))
	}

	middlewareStack = append(middlewareStack,
		deps.Limiter.Middleware(deps.Logger),
		deps.GeoStats.Middleware(deps.Logger),
		middleware.Logger(deps.Logger), // Inner logger (shows simple text logs)
	)

	return middleware.Chain(mux, middlewareStack...)
}
