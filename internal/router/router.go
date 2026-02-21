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
	Cfg          *config.Config
	Logger       *slog.Logger
	BlogHandler  *handlers.BlogHandler
	AssetHandler *handlers.AssetHandler
	Limiter      *middleware.IPRateLimiter
	AuthLimiter  *middleware.IPRateLimiter
	GeoStats     *middleware.GeoStats
	Tracer       trace.Tracer
	Metrics      *telemetry.Metrics
	Session      *middleware.Sessions
	CSRF         *middleware.CSRF
	CSP          *middleware.CSP
}

func NewRouter(deps RouterDependencies) http.Handler {
	// routing
	appMux := http.NewServeMux()

	// static files
	fs := http.FileServer(http.Dir("static"))
	appMux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	appMux.Handle("GET /assets/{key}", deps.AssetHandler)

	authDelay := 500 * time.Millisecond
	authStack := func(h http.Handler) http.Handler {
		h = middleware.SecureDelay(authDelay, deps.Metrics)(h)
		h = deps.AuthLimiter.Middleware(deps.Logger)(h)
		return h
	}

	// auth
	appMux.Handle("GET /register", deps.BlogHandler.HandleRegisterPage())
	appMux.Handle("POST /register", authStack(deps.BlogHandler.HandleRegister()))
	appMux.Handle("GET /login", deps.BlogHandler.HandleLoginPage())
	appMux.Handle("POST /login", authStack(deps.BlogHandler.HandleLogin()))
	appMux.Handle("POST /logout", authStack(deps.BlogHandler.HandleLogout()))
	appMux.Handle("POST /post/{id}/comment", authStack(deps.BlogHandler.HandleComment()))
	appMux.Handle("POST /post/{id}/comment/{commentID}/delete", authStack(deps.BlogHandler.HandleDeleteComment()))

	// routes
	appMux.Handle("GET /{$}", deps.BlogHandler.HandleIndex())
	appMux.Handle("GET /post/{id}", deps.BlogHandler.HandlePost())

	appMux.HandleFunc("/", deps.BlogHandler.NotFound)

	// static
	appMux.Handle("/about", deps.BlogHandler.HandleAbout())
	appMux.Handle("/privacy", deps.BlogHandler.HandlePrivacy())
	appMux.Handle("/terms", deps.BlogHandler.HandleTerms())
	appMux.Handle("/contact", deps.BlogHandler.HandleContact())

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
		deps.Session.Middleware(deps.Logger, deps.Tracer),
		deps.CSRF.Middleware(deps.Logger),
		middleware.Logger(deps.Logger), // Inner logger (shows simple text logs)
	)

	appHandler := middleware.Chain(appMux, middlewareStack...)

	rootMux := http.NewServeMux()

	rootMux.Handle("GET /metrics", deps.BlogHandler.HandleMetrics())

	// lightweight for docker keepalive
	rootMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	rootMux.Handle("/", appHandler)

	return rootMux
}
