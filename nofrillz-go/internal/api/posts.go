package api

import (
	"errors"
	"net/http"
	"strconv"

	"nofrillz/internal/app"
	"nofrillz/internal/posts"
)

type PostsHandler struct {
	app *app.App
}

type createPostRequest struct {
	Body string `json:"body"`
}

func NewPostsHandler(app *app.App) *PostsHandler {
	return &PostsHandler{
		app: app,
	}
}

func (h *PostsHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware(RequireAuthentication(next))
	}

	sm.HandleFunc("POST /posts", requireAuth(h.Create))
	sm.HandleFunc("GET /posts/{id}", requireAuth(h.Show))
	sm.HandleFunc("POST /posts/{post_id}/like", requireAuth(h.Like))
	sm.HandleFunc("DELETE /posts/{post_id}/like", requireAuth(h.Unlike))
}

func (h *PostsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	createRequest, err := DecodeRequestBody[createPostRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid create post json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized create post attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeUserID := authenticatedUser.ID

	if !AllowWriteByUser(w, h.app, writeUserID) {
		return
	}

	post, err := h.app.Posts.CreatePost(r.Context(), posts.CreatePostInput{
		AuthorID: writeUserID,
		Body:     createRequest.Body,
		Source:   posts.SourceHuman,
	})
	if err != nil {
		if errors.Is(err, posts.ErrInvalidAuthorID) || errors.Is(err, posts.ErrInvalidBody) || errors.Is(err, posts.ErrInvalidSource) {
			h.app.Logger.Warn().Uint64("author_id", writeUserID).Msg("invalid post payload")
			http.Error(w, "invalid post payload", http.StatusBadRequest)
			return
		}
		h.app.Logger.Error().Err(err).Msg("error in repository.Create")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	createdPost, err := h.app.Posts.GetByID(r.Context(), post.ID, writeUserID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Posts.GetByID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if createdPost == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	SetPostResponseFields(h.app, createdPost)
	h.app.Logger.Info().
		Uint64("post_id", createdPost.ID).
		Uint64("author_id", writeUserID).
		Str("source", createdPost.Source).
		Msg("post created")

	Respond(w, http.StatusOK, createdPost, h.app.Logger)
}

func (h *PostsHandler) Show(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)

	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized get post attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postId, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	post, err := h.app.Posts.GetByID(r.Context(), postId, authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Posts.GetByID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if post == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	SetPostResponseFields(h.app, post)
	h.app.Logger.Debug().
		Uint64("post_id", post.ID).
		Uint64("requester_user_id", authenticatedUser.ID).
		Str("source", post.Source).
		Msg("post fetched")

	Respond(w, http.StatusOK, post, h.app.Logger)
}

func (h *PostsHandler) Like(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	postID, err := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err != nil {
		h.app.Logger.Warn().Str("post_id", r.PathValue("post_id")).Msg("invalid like post id")
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Uint64("post_id", postID).Msg("unauthorized like attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	post, err := h.app.Posts.GetByID(r.Context(), postID, authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Posts.GetByID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if post == nil || post.Deleted != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = h.app.Likes.Like(r.Context(), postID, authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Likes.Like")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.app.Logger.Info().Uint64("post_id", postID).Uint64("user_id", authenticatedUser.ID).Msg("post liked")

	Respond(w, http.StatusOK, nil, h.app.Logger)
}

func (h *PostsHandler) Unlike(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	postID, err := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err != nil {
		h.app.Logger.Warn().Str("post_id", r.PathValue("post_id")).Msg("invalid unlike post id")
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Uint64("post_id", postID).Msg("unauthorized unlike attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	post, err := h.app.Posts.GetByID(r.Context(), postID, authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Posts.GetByID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if post == nil || post.Deleted != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = h.app.Likes.Unlike(r.Context(), postID, authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Likes.Unlike")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.app.Logger.Info().Uint64("post_id", postID).Uint64("user_id", authenticatedUser.ID).Msg("post unliked")

	Respond(w, http.StatusOK, nil, h.app.Logger)
}
