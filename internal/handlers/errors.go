package handlers

import (
	"blogengine/internal/components"
	"net/http"
)

// InternalError handles 500 errors
func (h *BlogHandler) InternalError(w http.ResponseWriter, r *http.Request, err error) {
	h.Logger.Error("500 internal server error", "err", err, "path", r.URL.Path)
	h.renderError(w, r, http.StatusInternalServerError,
		"Internal Server Error",
		"Something went wrong on our end. We've logged the error and will look into it.",
	)
}

// Unauthorised handles 401 errors
func (h *BlogHandler) Unauthorised(w http.ResponseWriter, r *http.Request) {
	h.Logger.Error("401 unauthorised", "path", r.URL.Path, "ip", r.RemoteAddr)
	h.renderError(w, r, http.StatusUnauthorized,
		"Unauthorised",
		"You need to be logged in to access this page or perform this action.",
	)
}

// RenderNotFound is a helper to serve the custom 404 page
func (h *BlogHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.Logger.Warn("404 not found", "path", r.URL.Path, "method", r.Method, "ip", r.RemoteAddr)
	h.renderError(w, r, http.StatusNotFound,
		"Page Not Found",
		"The page you are looking for doesn't exist or has been moved.",
	)
}

// renderError writes a header code wraps the call to the ErrorPage component with common data
func (h *BlogHandler) renderError(w http.ResponseWriter, r *http.Request, code int, title, message string) {
	w.WriteHeader(code)
	common := h.newCommonData(r)
	components.ErrorPage(common, code, title, message).Render(r.Context(), w)
}
