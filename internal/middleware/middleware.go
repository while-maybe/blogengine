package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"time"
)

type Middleware func(next http.Handler) http.Handler

// Chain returns a single Handler chaining the provided individual Midllewares in the correct order
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range slices.Backward(middlewares) {
		h = middleware(h)
	}
	return h
}

// Recover recovers from a crash, logging the reason and a stack trace to the provided logger
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered", "err", err, "stack", string(debug.Stack()))

					// attempts to return a 500 to the user if it can (was code written before?)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func Logger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			next.ServeHTTP(w, r)

			logger.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
		})
	}
}
