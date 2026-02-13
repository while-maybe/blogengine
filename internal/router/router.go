package router

import (
	"blogengine/internal/config"
	"blogengine/internal/handlers"
	"blogengine/internal/middleware"
	"blogengine/internal/telemetry"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// RouterDependencies holds everything needed to register routes.
type RouterDependencies struct {
	Cfg               *config.Config
	Logger            *slog.Logger
	BlogHandler       *handlers.BlogHandler
	AssetHandler      *handlers.AssetHandler
	Limiter           *middleware.IPRateLimiter
	AuthLimiter       *middleware.IPRateLimiter
	GeoStats          *middleware.GeoStats
	Tracer            trace.Tracer
	Metrics           *telemetry.Metrics
	PrometheusHandler http.Handler
	Session           *middleware.Sessions
	CSRF              *middleware.CSRF
	CSP               *middleware.CSP
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
	mux.Handle("GET /assets/{key}", deps.AssetHandler)

	authDelay := 500 * time.Millisecond
	authStack := func(h http.Handler) http.Handler {
		h = middleware.SecureDelay(authDelay, deps.Metrics)(h)
		h = deps.AuthLimiter.Middleware(deps.Logger)(h)
		return h
	}

	// auth
	mux.Handle("GET /register", deps.BlogHandler.HandleRegisterPage())
	mux.Handle("POST /register", authStack(deps.BlogHandler.HandleRegister()))
	mux.Handle("GET /login", deps.BlogHandler.HandleLoginPage())
	mux.Handle("POST /login", authStack(deps.BlogHandler.HandleLogin()))
	mux.Handle("POST /logout", authStack(deps.BlogHandler.HandleLogout()))
	mux.Handle("POST /post/{id}/comment", authStack(deps.BlogHandler.HandleComment()))
	mux.Handle("POST /post/{id}/comment/{commentID}/delete", authStack(deps.BlogHandler.HandleDeleteComment()))

	// routes
	mux.Handle("GET /{$}", deps.BlogHandler.HandleIndex())
	mux.Handle("GET /post/{id}", deps.BlogHandler.HandlePost())
	mux.Handle("GET /metrics", deps.BlogHandler.HandleMetrics())

	mux.HandleFunc("/", deps.BlogHandler.NotFound)

	middlewareStack := []middleware.Middleware{
		middleware.Recover(deps.Logger),
	}

	if deps.Cfg.Metrics.EnableTelemetry {
		// order matters so don't append
		middlewareStack = append(middlewareStack, middleware.Observability(deps.Tracer, deps.Metrics, deps.Logger))
	}

	middlewareStack = append(middlewareStack,
		deps.CSP.Middleware(),
		deps.Limiter.Middleware(deps.Logger),
		deps.GeoStats.Middleware(deps.Logger),
		deps.Session.Middleware(deps.Logger),
		deps.CSRF.Middleware(deps.Logger),
		middleware.Logger(deps.Logger), // Inner logger (shows simple text logs)
	)

	return middleware.Chain(mux, middlewareStack...)
}
