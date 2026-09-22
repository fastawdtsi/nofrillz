package api

import (
	"net/http"
	"strconv"
	"time"

	"nofrillz/internal/app"
	"nofrillz/internal/feed"
)

const (
	defaultFeedLimit = 25
	maxFeedLimit     = 100
)

type FeedHandler struct {
	app *app.App
}

type feedPostUserResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	AccountType string `json:"account_type"`
}

type feedPostResponse struct {
	ContentItemID *uint64 `json:"content_item_id,omitempty,string"`
	ModelOption   string  `json:"model_option,omitempty"`
	Provider      string  `json:"provider,omitempty"`
	Model         string  `json:"model,omitempty"`

	ID           string               `json:"id"`
	URL          string               `json:"url"`
	Body         string               `json:"body"`
	Source       string               `json:"source"`
	Created      string               `json:"created"`
	Liked        bool                 `json:"liked"`
	Bookmarked   bool                 `json:"bookmarked"`
	IsBookmarked bool                 `json:"is_bookmarked"`
	LikeCount    uint32               `json:"like_count"`
	CommentCount uint32               `json:"comment_count"`
	User         feedPostUserResponse `json:"user"`
}

type feedResponse struct {
	Posts      []feedPostResponse `json:"posts"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

func NewFeedHandler(app *app.App) *FeedHandler {
	return &FeedHandler{
		app: app,
	}
}

func (h *FeedHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware(RequireAuthentication(next))
	}

	sm.HandleFunc("GET /feed", requireAuth(h.List))
	sm.HandleFunc("GET /feed/discover", requireAuth(h.List))
}

func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}
	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized feed request")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := defaultFeedLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxFeedLimit {
			parsedLimit = maxFeedLimit
		}
		limit = parsedLimit
	}

	cursor, err := DecodeIDCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		h.app.Logger.Warn().Uint64("user_id", authenticatedUser.ID).Str("cursor", r.URL.Query().Get("cursor")).Msg("invalid feed cursor")
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	var posts []*feed.Post
	var nextCursor *uint64
	if r.URL.Path == "/feed/discover" {
		posts, nextCursor, err = h.app.Feed.Discover(r.Context(), authenticatedUser.ID, cursor, limit)
	} else {
		posts, nextCursor, err = h.app.Feed.List(r.Context(), authenticatedUser.ID, cursor, limit)
	}
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Feed.List")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	responsePosts := make([]feedPostResponse, 0, len(posts))
	for _, post := range posts {
		responsePosts = append(responsePosts, feedPostResponse{
			ContentItemID: post.ContentItemID, ModelOption: post.ModelOption, Provider: post.Provider, Model: post.Model,
			ID:           strconv.FormatUint(post.ID, 10),
			URL:          PostURLPath(post.ID),
			Body:         post.Body,
			Source:       post.Source,
			Created:      post.Created.UTC().Format(time.RFC3339Nano),
			Liked:        post.Liked,
			Bookmarked:   post.IsBookmarked,
			IsBookmarked: post.IsBookmarked,
			User: feedPostUserResponse{
				ID:          strconv.FormatUint(post.UserID, 10),
				Username:    post.Username,
				FirstName:   post.FirstName,
				LastName:    post.LastName,
				AccountType: post.AccountType,
			},
		})
	}

	nextCursorValue, err := EncodeIDCursor(nextCursor)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error encoding feed cursor")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := feedResponse{
		Posts:      responsePosts,
		NextCursor: nextCursorValue,
	}
	h.app.Logger.Debug().
		Uint64("user_id", authenticatedUser.ID).
		Int("limit", limit).
		Int("results", len(responsePosts)).
		Str("next_cursor", nextCursorValue).
		Msg("feed fetched")
	Respond(w, http.StatusOK, response, h.app.Logger)
}
