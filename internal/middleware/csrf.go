package middleware

import (
	"log/slog"
	"net/http"

	"github.com/justinas/nosurf"
)

type CSRF struct {
	isProd bool
}

func NewCSRF(isProd bool) *CSRF {
	return &CSRF{isProd: isProd}
}

func (c *CSRF) Middleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		csrfHandler := nosurf.New(next)

		csrfHandler.SetBaseCookie(http.Cookie{
			HttpOnly: true,
			Path:     "/",
			Secure:   c.isProd,
			SameSite: http.SameSiteLaxMode,
		})

		csrfHandler.SetFailureHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				logger.Warn("CSRF validation failed", "path", r.URL.Path, "ip", r.RemoteAddr)
				http.Error(w, "invalid CSRF token", http.StatusBadRequest)
			}))

		return csrfHandler
	}
}
