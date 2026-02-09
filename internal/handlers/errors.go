package handlers

import (
	"blogengine/internal/components"
	"net/http"
)

// RenderNotFound is a helper to serve the custom 404 page
func (h *BlogHandler) HandleNotFound() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username := h.GetUserFromSession(r)

		h.Logger.Warn("404 not found", "path", r.URL.Path, "method", r.Method, "ip", r.RemoteAddr)
		w.WriteHeader(http.StatusNotFound)
		components.NotFound(h.Title, username).Render(r.Context(), w)
	})
}
