package handlers

// func (h *BlogHandler) HandlePost() http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		ctx, span := h.Tracer.Start(r.Context(), "HandlePost")
// 		defer span.End()

// 		username := h.GetUserFromSession(r)
// 		if username != "" {
// 			span.SetAttributes(attribute.String("user.name", username))
// 		}

// 		idStr := r.PathValue("id")
// 		span.SetAttributes(attribute.String("post.id_str", idStr))

// 		id64, err := strconv.ParseUint(idStr, 10, 32)
// 		if err != nil {
// 			h.NotFound(w, r)
// 			slog.Error("parsing file id", "idStr", idStr, "err", err)
// 			return
// 		}

// 		postID := uint32(id64)
// 		span.SetAttributes(attribute.Int("post.id", int(postID)))

// 		// find the post
// 		post, err := h.Store.Get(postID)
// 		if err != nil {
// 			switch {
// 			case errors.Is(err, content.ErrPostNotFound):
// 				h.NotFound(w, r)
// 			default:
// 				h.InternalError(w, r, err)
// 				slog.Error("finding post", "id", id64, "err", err)
// 			}
// 			return
// 		}

// 		span.SetAttributes(
// 			attribute.String("post.title", post.Title),
// 			attribute.String("post.author", post.Author),
// 		)

// 		// also record post view metrics
// 		h.Metrics.PostViewsTotal.Add(ctx, 1,
// 			metric.WithAttributes(
// 				attribute.String("post.title", post.Title),
// 				attribute.Int("post.id", int(postID)),
// 			),
// 		)

// 		// load content
// 		htmlBytes, err := post.GetContent(h.Renderer)
// 		// REMEMBER htmlBytes, err := h.Renderer.Render(post.Content) - if lazy loading, this won't work as Content will be nil
// 		if err != nil {
// 			switch {
// 			case errors.Is(err, content.ErrFileTooLarge):
// 				h.RenderError(w, r, http.StatusRequestEntityTooLarge, "413 entity too large", "post too large")
// 			default:
// 				h.InternalError(w, r, err)
// 			}
// 			slog.Error("handling post", "title", post.Title, "err", err)
// 			return
// 		}

// 		span.SetAttributes(attribute.Int("post.content_bytes", len(htmlBytes)))

// 		body := templ.Raw(string(htmlBytes))

// 		comments, err := h.DB.GetCommentsForPost(ctx, int64(postID), 0, 100)
// 		if err != nil {
// 			h.Logger.Error("failed to fetch comments", "post_id", postID, "err", err)
// 			comments = []*storage.Comment{}
// 		}

// 		common := h.newCommonData(r)

// 		components.BlogPost(common, post.ID, body, comments).Render(ctx, w)
// 	})
// }
