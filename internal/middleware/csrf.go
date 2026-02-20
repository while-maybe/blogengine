package middleware

import (
	"log/slog"
	"net/http"

	"github.com/justinas/nosurf"
)

type ErrorRenderer func(w http.ResponseWriter, r *http.Request, code int, title, message string)

type CSRF struct {
	isProd        bool
	errorRenderer ErrorRenderer
}

func NewCSRF(isProd bool, renderer ErrorRenderer) *CSRF {
	return &CSRF{
		isProd:        isProd,
		errorRenderer: renderer,
	}
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

		// we're in dev mode...
		if !c.isProd {
			csrfHandler.ExemptFunc(func(r *http.Request) bool {
				return true
			})
		}

		csrfHandler.SetFailureHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				logger.Warn("CSRF validation failed",
					"path", r.URL.Path,
					"ip", r.RemoteAddr,
					"reason", nosurf.Reason(r),
				)

				msg := "This form has expired. Please go back, refresh the page, and try again."
				c.errorRenderer(w, r, http.StatusBadRequest, "Bad Request", msg)

			}))

		return csrfHandler
	}
}
