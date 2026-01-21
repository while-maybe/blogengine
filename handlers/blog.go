package handlers

import (
	"blogengine/components"
	"blogengine/content"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

type PostProvider interface {
	GetAll() []*content.Post
	Get(id uint32) (*content.Post, bool)
}

// BlogHandler holds the state
type BlogHandler struct {
	Title string
	Store PostProvider
}

// NewBlogHandler creates the controller
func NewBlogHandler(store *content.Repository, title string) *BlogHandler {
	return &BlogHandler{
		Store: store,
		Title: title,
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
		post, ok := h.Store.Get(uint32(id64))
		if !ok {
			http.NotFound(w, r)
			return
		}

		// load content
		htmlBytes, err := post.GetContent()
		if err != nil {
			log.Printf("Error loading post %s: %v", post.Title, err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		body := templ.Raw(string(htmlBytes))
		blogTitle := h.Title

		components.BlogPost(blogTitle, body).Render(r.Context(), w)
	})
}
