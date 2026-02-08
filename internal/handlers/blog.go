package handlers

import (
	"blogengine/internal/components"
	"blogengine/internal/content"
	"blogengine/internal/middleware"
	"blogengine/internal/storage"
	"blogengine/internal/telemetry"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type PostProvider interface {
	GetAll() []*content.Post
	Get(id uint32) (*content.Post, error)
}

// BlogHandler holds the state
type BlogHandler struct {
	Title    string
	Store    content.PostService
	DB       storage.Store
	Renderer *content.MarkDownRenderer
	Logger   *slog.Logger
	GeoStats *middleware.GeoStats
	Tracer   trace.Tracer
	Metrics  *telemetry.Metrics
}

// NewBlogHandler creates the controller
func NewBlogHandler(store content.PostService, db storage.Store, renderer *content.MarkDownRenderer, title string, logger *slog.Logger, geo *middleware.GeoStats, tracer trace.Tracer, metrics *telemetry.Metrics) *BlogHandler {
	return &BlogHandler{
		Store:    store,
		DB:       db,
		Renderer: renderer,
		Title:    title,
		Logger:   logger,
		GeoStats: geo,
		Tracer:   tracer,
		Metrics:  metrics,
	}
}

func (h *BlogHandler) HandleIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.Tracer.Start(r.Context(), "HandleIndex")
		defer span.End()

		// only allow '/'
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		allPosts := h.Store.GetAll()
		span.SetAttributes(attribute.Int("posts.count", len(allPosts)))

		blogTitle := h.Title

		components.Home(allPosts, blogTitle).Render(ctx, w)
	})
}

func (h *BlogHandler) HandlePost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx, span := h.Tracer.Start(r.Context(), "HandlePost")
		defer span.End()

		idStr := strings.TrimPrefix(r.URL.Path, "/post/")
		span.SetAttributes(attribute.String("post.id_str", idStr))

		id64, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			http.NotFound(w, r)
			slog.Error("parsing file id", "idStr", idStr, "err", err)
			return
		}

		postID := uint32(id64)
		span.SetAttributes(attribute.Int("post.id", int(postID)))

		// find the post
		post, err := h.Store.Get(postID)
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

		span.SetAttributes(
			attribute.String("post.title", post.Title),
			attribute.String("post.author", post.Author),
		)

		// also record post view metrics
		h.Metrics.PostViewsTotal.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("post.title", post.Title),
				attribute.Int("post.id", int(postID)),
			),
		)

		// load content
		htmlBytes, err := post.GetContent(h.Renderer)
		// REMEMBER htmlBytes, err := h.Renderer.Render(post.Content) - if lazy loading, this won't work as Content will be nil
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

		span.SetAttributes(attribute.Int("post.content_bytes", len(htmlBytes)))

		body := templ.Raw(string(htmlBytes))
		blogTitle := h.Title

		components.BlogPost(blogTitle, body).Render(ctx, w)
	})
}
