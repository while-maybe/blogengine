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
	"time"

	"github.com/a-h/templ"
	"github.com/justinas/nosurf"
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
	Title       string
	NeedsInvite bool
	InviteCode  string
	Store       content.PostService
	DB          storage.Store
	Renderer    *content.MarkDownRenderer
	Logger      *slog.Logger
	GeoStats    *middleware.GeoStats
	Tracer      trace.Tracer
	Metrics     *telemetry.Metrics
	Sessions    *middleware.Sessions
	StartTime   time.Time
}

// NewBlogHandler creates the controller
func NewBlogHandler(store content.PostService, db storage.Store, renderer *content.MarkDownRenderer, title string, needsInvite bool, inviteCode string, logger *slog.Logger, geo *middleware.GeoStats, tracer trace.Tracer, metrics *telemetry.Metrics, sm *middleware.Sessions, startTime time.Time) *BlogHandler {
	return &BlogHandler{
		Store:       store,
		DB:          db,
		Renderer:    renderer,
		Title:       title,
		NeedsInvite: needsInvite,
		InviteCode:  inviteCode,
		Logger:      logger,
		GeoStats:    geo,
		Tracer:      tracer,
		Metrics:     metrics,
		Sessions:    sm,
		StartTime:   startTime,
	}
}

// newCommonData is needed to prevent circular imports
func (h *BlogHandler) newCommonData(r *http.Request) components.CommonData {
	return components.CommonData{
		Title:     h.Title,
		Username:  h.GetUserFromSession(r),
		CSRFToken: nosurf.Token(r),
	}
}

func (h *BlogHandler) HandleIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.Tracer.Start(r.Context(), "HandleIndex")
		defer span.End()
		common := h.newCommonData(r)

		allPosts := h.Store.GetAll()
		span.SetAttributes(attribute.Int("posts.count", len(allPosts)))

		components.Home(allPosts, common).Render(ctx, w)
	})
}

func (h *BlogHandler) HandlePost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.Tracer.Start(r.Context(), "HandlePost")
		defer span.End()

		username := h.GetUserFromSession(r)
		if username != "" {
			span.SetAttributes(attribute.String("user.name", username))
		}

		idStr := r.PathValue("id")
		span.SetAttributes(attribute.String("post.id_str", idStr))

		id64, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			h.NotFound(w, r)
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
				h.NotFound(w, r)
			default:
				h.InternalError(w, r, err)
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
				h.renderError(w, r, http.StatusRequestEntityTooLarge, "413 entity too large", "post too large")
			default:
				h.InternalError(w, r, err)
			}
			slog.Error("handling post", "title", post.Title, "err", err)
			return
		}

		span.SetAttributes(attribute.Int("post.content_bytes", len(htmlBytes)))

		body := templ.Raw(string(htmlBytes))

		comments, err := h.DB.GetCommentsForPost(ctx, int64(postID), 0, 100)
		if err != nil {
			h.Logger.Error("failed to fetch comments", "post_id", postID, "err", err)
			comments = []*storage.Comment{}
		}

		common := h.newCommonData(r)

		components.BlogPost(common, post.ID, body, comments).Render(ctx, w)
	})
}
