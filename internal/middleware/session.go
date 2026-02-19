package middleware

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Sessions struct {
	Manager *scs.SessionManager
}

func NewSessionManager(ttl time.Duration, secure bool, db *sql.DB) *Sessions {
	sm := scs.New()

	sm.Lifetime = ttl
	sm.Store = sqlite3store.New(db) // original raw *sql.DB

	sm.Cookie.Name = "session_id"
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = secure
	sm.Cookie.Persist = true // desired for a blog

	return &Sessions{Manager: sm}
}

func (s *Sessions) Middleware(logger *slog.Logger, tracer trace.Tracer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), "middleware.Session")
			defer span.End()

			// tag span with cookie name
			span.SetAttributes(attribute.String("session.cookie", s.Manager.Cookie.Name))

			s.Manager.LoadAndSave(next).ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
