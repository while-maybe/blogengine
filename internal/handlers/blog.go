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
	"time"

	"github.com/justinas/nosurf"
	"go.opentelemetry.io/otel/trace"
)

// BlogHandler holds the state
type BlogHandler struct {
	Title       string
	NeedsInvite bool
	InviteCode  string
	DB          storage.Store
	S3          storage.S3Store
	GeoStats    *middleware.GeoStats
	Renderer    *content.MarkDownRenderer
	Logger      *slog.Logger
	Tracer      trace.Tracer
	Metrics     *telemetry.Metrics
	Sessions    *middleware.Sessions
	StartTime   time.Time
}

type HandlerConfig struct {
	Title       string
	NeedsInvite bool
	InviteCode  string
	DB          storage.Store
	S3          storage.S3Store
	GeoStats    *middleware.GeoStats
	Renderer    *content.MarkDownRenderer
	Logger      *slog.Logger
	Tracer      trace.Tracer
	Metrics     *telemetry.Metrics
	Sessions    *middleware.Sessions
	StartTime   time.Time
}

func NewHandler(cfg HandlerConfig) *BlogHandler {
	return &BlogHandler{
		Title:       cfg.Title,
		NeedsInvite: cfg.NeedsInvite,
		InviteCode:  cfg.InviteCode,
		DB:          cfg.DB,
		S3:          cfg.S3,
		GeoStats:    cfg.GeoStats,
		Renderer:    cfg.Renderer,
		Logger:      cfg.Logger,
		Tracer:      cfg.Tracer,
		Metrics:     cfg.Metrics,
		Sessions:    cfg.Sessions,
		StartTime:   cfg.StartTime,
	}
}

func (h *BlogHandler) newCommonData(r *http.Request) components.CommonData {
	return components.CommonData{
		Title:     h.Title,
		Username:  h.GetUserFromSession(r),
		CSRFToken: nosurf.Token(r),
	}
}

func (h *BlogHandler) HandleHome() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.Tracer.Start(r.Context(), "HandleHome")
		defer span.End()
		common := h.newCommonData(r)

		blogs, err := h.DB.GetPublicBlogs(ctx, 0, 5)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}

		posts, err := h.DB.GetLatestPublicPosts(ctx, 0, 10)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}

		components.Home(blogs, posts, common).Render(ctx, w)
	})
}

func (h *BlogHandler) HandleBlog() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.Tracer.Start(r.Context(), "HandleBlog")
		defer span.End()
		common := h.newCommonData(r)

		slug := r.PathValue("blog_slug")
		blog, err := h.DB.GetBlogBySlug(ctx, slug)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				h.NotFound(w, r)
			default:
				h.InternalError(w, r, err)
			}
			return
		}

		posts, err := h.DB.GetPostsByBlogID(ctx, blog.ID, 0, 10)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}

		components.Blog(blog, posts, common).Render(ctx, w)
	})
}

func (h *BlogHandler) HandleAbout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		common := h.newCommonData(r)
		components.About(common).Render(r.Context(), w)
	})
}

func (h *BlogHandler) HandleContact() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		common := h.newCommonData(r)
		components.Contact(common).Render(r.Context(), w)
	})
}

func (h *BlogHandler) HandleTerms() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		common := h.newCommonData(r)
		components.Terms(common).Render(r.Context(), w)
	})
}

func (h *BlogHandler) HandlePrivacy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		common := h.newCommonData(r)
		components.Privacy(common).Render(r.Context(), w)
	})
}
