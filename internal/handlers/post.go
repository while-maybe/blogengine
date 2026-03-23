package handlers

import (
	"blogengine/internal/components"
	"blogengine/internal/storage"
	"errors"
	"io"
	"net/http"

	"github.com/a-h/templ"
)

func (h *BlogHandler) HandlePost() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := h.Tracer.Start(r.Context(), "HandlPost")
		defer span.End()
		common := h.newCommonData(r)

		blogSlug := r.PathValue("blog_slug")
		postSlug := r.PathValue("post_slug")

		post, err := h.DB.GetPostBySlugOrPublicID(ctx, blogSlug, postSlug)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				h.NotFound(w, r)
			default:
				h.InternalError(w, r, err)
			}
			return
		}

		rc, err := h.S3.Open(ctx, post.S3Key)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}
		defer rc.Close()

		contentBytes, err := io.ReadAll(rc)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}

		// render the fetched markdown to html
		htmlBytes, err := h.Renderer.Render(contentBytes)
		if err != nil {
			h.InternalError(w, r, err)
			return
		}

		body := templ.Raw(string(htmlBytes))

		comments, err := h.DB.GetCommentsForPost(ctx, post.ID, 0, 100)
		if err != nil {
			h.Logger.Error("failed to fetch comments", "post_id", post.ID, "err", err)
			comments = []*storage.Comment{}
		}

		components.Post(common, post, body, comments).Render(ctx, w)
	})
}
