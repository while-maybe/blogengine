package handlers

import (
	"blogengine/internal/components"
	"blogengine/internal/storage"
	"errors"
	"net/http"
	"strings"

	"github.com/justinas/nosurf"
	"golang.org/x/crypto/bcrypt"
)

func (h *BlogHandler) HandleRegisterPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// user already logged in, send home
		if h.Sessions.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		token := nosurf.Token(r)

		components.Register(h.Title, "", "", token).Render(r.Context(), w)
	})
}

func (h *BlogHandler) HandleRegister() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// user already logged in, send home
		if h.Sessions.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		username = strings.TrimSpace(username)

		password := r.FormValue("password")
		confirm := r.FormValue("confirm_password")

		token := nosurf.Token(r)

		if password != confirm {
			w.WriteHeader(http.StatusBadRequest)
			components.Register(h.Title, username, "Passwords do not match.", token).Render(r.Context(), w)
			return
		}

		if len(username) < 3 || len(password) < 8 {
			w.WriteHeader(http.StatusBadRequest)
			components.Register(h.Title, "", "Inputs too short.", token).Render(r.Context(), w)
			return
		}

		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			h.Logger.Error("hashing password", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if _, err = h.DB.CreateUser(r.Context(), username, string(hashedPwd)); err != nil {
			switch {
			case errors.Is(err, storage.ErrUniqueViolation):
				w.WriteHeader(http.StatusConflict)
				components.Register(h.Title, "", "Username already taken.", token).Render(r.Context(), w)
			default:
				h.Logger.Error("creating user", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

func (h *BlogHandler) HandleLoginPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// user already logged in, send home
		if h.Sessions.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		token := nosurf.Token(r)

		components.Login(h.Title, "", "", token).Render(r.Context(), w)
	})
}

// HandleLogin processes the login form submission
func (h *BlogHandler) HandleLogin() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// user already logged in, send home
		if h.Sessions.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		username = strings.TrimSpace(username)

		password := r.FormValue("password")

		token := nosurf.Token(r)

		user, err := h.DB.GetUserByUsername(r.Context(), username)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				w.WriteHeader(http.StatusUnauthorized)
				components.Login(h.Title, "", "Invalid username or password.", token).Render(r.Context(), w)
			default:
				h.Logger.Error("db error on login", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			components.Login(h.Title, "", "Invalid username or password.", token).Render(r.Context(), w)
			return
		}

		if err := h.Sessions.Manager.RenewToken(r.Context()); err != nil {
			h.Logger.Error("session token renewal", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		h.Sessions.Manager.Put(r.Context(), "userID", user.ID)
		h.Sessions.Manager.Put(r.Context(), "username", username)

		h.Logger.Info("user logged in", "id", user.ID, "username", user.Username)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func (h *BlogHandler) HandleLogout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// destroy session in db and clear cookie
		if err := h.Sessions.Manager.Destroy(r.Context()); err != nil {
			h.Logger.Error("destroy session", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		h.Logger.Info("user logged out")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

// GetUserFromSession is a helper to get the logged-in username
func (h *BlogHandler) GetUserFromSession(r *http.Request) string {
	return h.Sessions.Manager.GetString(r.Context(), "username")
}
