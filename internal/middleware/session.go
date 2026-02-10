package middleware

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
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

func (s *Sessions) Middleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {

		logger.Info("session manager", "id", s.Manager.Cookie.Name)
		return s.Manager.LoadAndSave(next)

	}
}
