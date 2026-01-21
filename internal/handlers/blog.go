package handlers

import (
	"blogengine/internal/components"
	"blogengine/internal/content"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

type PostProvider interface {
	GetAll() []*content.Post
	Get(id uint32) (*content.Post, error)
}

// BlogHandler holds the state
type BlogHandler struct {
	Title  string
	Store  PostProvider
	Logger *slog.Logger
}

// NewBlogHandler creates the controller
func NewBlogHandler(store *content.Repository, title string, logger *slog.Logger) *BlogHandler {
	return &BlogHandler{
		Store:  store,
		Title:  title,
		Logger: logger,
	}
}

func (h *BlogHandler) HandleIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allPosts := h.Store.GetAll()
		blogTitle := h.Title

		components.Home(allPosts, blogTitle).Render(r.Context(), w)
	})
}

func (h *BlogHandler) HandlePost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/post/")
		id64, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// find the post
		post, err := h.Store.Get(uint32(id64))
		if err != nil {
			switch {
			case errors.Is(err, content.ErrPostNotFound):
				http.NotFound(w, r)
			default:
				http.Error(w, "internal server error", http.StatusInternalServerError)
				slog.Error("finding post", "id", id64, "err", err)
			}
			return
		}

		// load content
		htmlBytes, err := post.GetContent()
		if err != nil {
			switch {
			case errors.Is(err, content.ErrFileTooLarge):
				http.Error(w, "post too large", http.StatusRequestEntityTooLarge)
			default:
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			slog.Error("handling post", "title", post.Title, "err", err)
			return
		}

		body := templ.Raw(string(htmlBytes))
		blogTitle := h.Title

		components.BlogPost(blogTitle, body).Render(r.Context(), w)
	})
}
