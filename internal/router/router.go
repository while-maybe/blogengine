package router

import (
	"blogengine/internal/config"
	"blogengine/internal/handlers"
	"blogengine/internal/middleware"
	"log/slog"
	"net/http"
)

// RouterDependencies holds everything needed to register routes.
type RouterDependencies struct {
	Cfg          *config.Config
	Logger       *slog.Logger
	BlogHandler  *handlers.BlogHandler
	AssetHandler *handlers.AssetHandler
	Limiter      *middleware.IPRateLimiter
	GeoStats     *middleware.GeoStats
}

func NewRouter(deps RouterDependencies) http.Handler {
	// routing
	mux := http.NewServeMux()

	// static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.Handle("GET /assets/", deps.AssetHandler)

	// routes
	mux.Handle("GET /", deps.BlogHandler.HandleIndex())
	mux.Handle("GET /post/", deps.BlogHandler.HandlePost())
	mux.Handle("GET /metrics", deps.BlogHandler.HandleMetrics())

	defaultMiddlewareStack := []middleware.Middleware{
		middleware.Recover(deps.Logger),
		deps.Limiter.Middleware(deps.Logger),
		middleware.Logger(deps.Logger),
		deps.GeoStats.Middleware(deps.Logger),
	}

	return middleware.Chain(mux, defaultMiddlewareStack...)

}
