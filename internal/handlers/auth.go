package handlers

import (
	"blogengine/internal/components"
	"blogengine/internal/storage"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func (h *BlogHandler) HandleRegisterPage() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// user already logged in, send home
		if h.Sessions.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		common := h.newCommonData(r)
		components.Register(common, "").Render(r.Context(), w)
	})
}

func (h *BlogHandler) HandleRegister() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// user already logged in, send home
		if h.Sessions.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		common := h.newCommonData(r)

		username := r.FormValue("username")
		username = strings.TrimSpace(username)

		password := r.FormValue("password")
		confirm := r.FormValue("confirm_password")

		if password != confirm {
			w.WriteHeader(http.StatusBadRequest)
			components.Register(common, "Passwords do not match.").Render(r.Context(), w)
			return
		}

		if len(username) < 3 || len(password) < 8 {
			w.WriteHeader(http.StatusBadRequest)
			components.Register(common, "Inputs too short.").Render(r.Context(), w)
			return
		}

		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}

		if _, err = h.DB.CreateUser(r.Context(), username, string(hashedPwd)); err != nil {
			switch {
			case errors.Is(err, storage.ErrUniqueViolation):
				w.WriteHeader(http.StatusConflict)
				components.Register(common, "Username already taken.").Render(r.Context(), w)
			default:
				h.InternalError(w, r, err)
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
		common := h.newCommonData(r)
		components.Login(common, "").Render(r.Context(), w)
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

		common := h.newCommonData(r)

		user, err := h.DB.GetUserByUsername(r.Context(), username)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				w.WriteHeader(http.StatusUnauthorized)
				components.Login(common, "Invalid username or password.").Render(r.Context(), w)
			default:
				h.InternalError(w, r, err)
			}
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			components.Login(common, "Invalid username or password.").Render(r.Context(), w)
			return
		}

		if err := h.Sessions.Manager.RenewToken(r.Context()); err != nil {
			h.InternalError(w, r, err)
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
			h.InternalError(w, r, err)
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
