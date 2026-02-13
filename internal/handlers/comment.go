package handlers

import (
	"blogengine/internal/storage"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (h *BlogHandler) HandleComment() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check auth
		userID := h.Sessions.Manager.GetInt64(r.Context(), "userID")
		if userID == 0 {
			h.Unauthorised(w, r)
			return
		}

		// check bot trap
		if r.FormValue("website") != "" {
			h.Logger.Warn("suspected bot: honeypot filled", "ip", r.RemoteAddr)
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		// we should get "post" "postID" "comment"
		postIDStr := r.PathValue("id")
		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil {
			h.NotFound(w, r)
			return
		}

		// find where to send the user to (don't trust Referer much)
		redirectTo := fmt.Sprintf("/post/%d", postID)

		// validate
		content := strings.TrimSpace(r.FormValue("content"))
		if content == "" || len(content) > 1000 {
			// redirect to same page if validation fails
			// TODO give feedback to user?
			http.Redirect(w, r, redirectTo, http.StatusSeeOther)
			return
		}

		// save if valid
		if _, err := h.DB.CreateComment(r.Context(), postID, userID, content); err != nil {
			h.InternalError(w, r, err)
			return
		}

		h.Logger.Info("new comment", "user_id", userID, "post_id", postID)

		// all good, redirect to same page to show the page with newly created comment
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
	})
}

func (h *BlogHandler) HandleDeleteComment() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// "POST /post/{id}/comment/{commentID}/delete"
		postIDStr := r.PathValue("id")
		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil {
			h.NotFound(w, r)
			return
		}

		commentIDStr := r.PathValue("commentID")
		commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
		if err != nil {
			h.NotFound(w, r)
			return
		}

		// check auth
		userID := h.Sessions.Manager.GetInt64(r.Context(), "userID")
		if userID == 0 {
			h.Unauthorised(w, r)
			return
		}

		if err := h.DB.DeleteComment(r.Context(), commentID, userID); err != nil {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				h.NotFound(w, r)
			default:
				h.InternalError(w, r, err)
			}
			return
		}

		h.Logger.Info("comment deleted", "user_id", userID, "comment_id", commentID)

		redirectTo := fmt.Sprintf("/post/%d", postID)
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
	})
}
