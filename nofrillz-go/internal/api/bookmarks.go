package api

import (
	"errors"
	"net/http"
	"strconv"

	"nofrillz/internal/app"
	"nofrillz/internal/bookmarks"
	"nofrillz/internal/posts"
)

const (
	defaultBookmarksLimit = 25
	maxBookmarksLimit     = 100
)

type bookmarkStateResponse struct {
	Bookmarked bool `json:"bookmarked"`
}

type bookmarksResponse struct {
	Posts      []*posts.Post `json:"posts"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type BookmarksHandler struct {
	app *app.App
}

func NewBookmarksHandler(app *app.App) *BookmarksHandler {
	return &BookmarksHandler{app: app}
}

func (h *BookmarksHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware(RequireAuthentication(next))
	}

	sm.HandleFunc("POST /posts/{post_id}/bookmark", requireAuth(h.Bookmark))
	sm.HandleFunc("DELETE /posts/{post_id}/bookmark", requireAuth(h.Unbookmark))
	sm.HandleFunc("GET /bookmarks", requireAuth(h.List))
}

func (h *BookmarksHandler) Bookmark(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	postID, err := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err != nil {
		h.app.Logger.Warn().Str("post_id", r.PathValue("post_id")).Msg("invalid bookmark post id")
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Uint64("post_id", postID).Msg("unauthorized bookmark attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	created, err := h.app.Bookmarks.Bookmark(r.Context(), postID, authenticatedUser.ID)
	if err != nil {
		switch {
		case errors.Is(err, bookmarks.ErrInvalidPostID):
			http.Error(w, "invalid post id", http.StatusBadRequest)
		case errors.Is(err, bookmarks.ErrPostNotFound):
			w.WriteHeader(http.StatusNotFound)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Bookmarks.Bookmark")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if created {
		h.app.Logger.Info().Uint64("post_id", postID).Uint64("user_id", authenticatedUser.ID).Msg("post bookmarked")
	} else {
		h.app.Logger.Debug().Uint64("post_id", postID).Uint64("user_id", authenticatedUser.ID).Msg("bookmark already existed")
	}

	Respond(w, http.StatusOK, bookmarkStateResponse{Bookmarked: true}, h.app.Logger)
}

func (h *BookmarksHandler) Unbookmark(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	postID, err := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err != nil {
		h.app.Logger.Warn().Str("post_id", r.PathValue("post_id")).Msg("invalid unbookmark post id")
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Uint64("post_id", postID).Msg("unauthorized unbookmark attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	removed, err := h.app.Bookmarks.Unbookmark(r.Context(), postID, authenticatedUser.ID)
	if err != nil {
		switch {
		case errors.Is(err, bookmarks.ErrInvalidPostID):
			http.Error(w, "invalid post id", http.StatusBadRequest)
		case errors.Is(err, bookmarks.ErrPostNotFound):
			w.WriteHeader(http.StatusNotFound)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Bookmarks.Unbookmark")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if removed {
		h.app.Logger.Info().Uint64("post_id", postID).Uint64("user_id", authenticatedUser.ID).Msg("post unbookmarked")
	} else {
		h.app.Logger.Debug().Uint64("post_id", postID).Uint64("user_id", authenticatedUser.ID).Msg("bookmark did not exist")
	}

	Respond(w, http.StatusOK, bookmarkStateResponse{Bookmarked: false}, h.app.Logger)
}

func (h *BookmarksHandler) List(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized bookmarks request")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowReadByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	limit := defaultBookmarksLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxBookmarksLimit {
			parsedLimit = maxBookmarksLimit
		}
		limit = parsedLimit
	}

	cursor, err := DecodeBookmarkCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		h.app.Logger.Warn().Uint64("user_id", authenticatedUser.ID).Str("cursor", r.URL.Query().Get("cursor")).Msg("invalid bookmarks cursor")
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	results, nextCursor, err := h.app.Bookmarks.ListByUser(r.Context(), authenticatedUser.ID, cursor, limit)
	if err != nil {
		switch {
		case errors.Is(err, bookmarks.ErrInvalidLimit):
			http.Error(w, "invalid limit", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Bookmarks.ListByUser")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if results == nil {
		results = []*posts.Post{}
	}
	SetPostResponseFieldsMany(h.app, results)

	nextCursorValue, err := EncodeBookmarkCursor(nextCursor)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error encoding bookmarks cursor")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.app.Logger.Debug().
		Uint64("user_id", authenticatedUser.ID).
		Int("limit", limit).
		Int("results", len(results)).
		Str("next_cursor", nextCursorValue).
		Msg("bookmarks fetched")

	Respond(w, http.StatusOK, bookmarksResponse{
		Posts:      results,
		NextCursor: nextCursorValue,
	}, h.app.Logger)
}
