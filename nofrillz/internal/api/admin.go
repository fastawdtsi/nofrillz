package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nofrillz/internal/admin"
	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/app"
	"nofrillz/internal/users"
)

const (
	defaultAdminGenerationsLimit = 20
	maxAdminGenerationsLimit     = 100
	defaultAdminUsersLimit       = 25
	maxAdminUsersLimit           = 100
	defaultAdminPostsLimit       = 20
	maxAdminPostsLimit           = 100
)

type AdminHandler struct {
	app *app.App
}

type createAdminAIAccountRequest struct {
	Email          string     `json:"email"`
	Username       string     `json:"username"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	About          string     `json:"about"`
	Enabled        *bool      `json:"enabled"`
	Topic          string     `json:"topic"`
	Description    string     `json:"description"`
	SystemPrompt   string     `json:"system_prompt"`
	StylePrompt    string     `json:"style_prompt"`
	MinPostsPerDay *int       `json:"min_posts_per_day"`
	MaxPostsPerDay *int       `json:"max_posts_per_day"`
	NextGenerateAt *time.Time `json:"next_generate_at"`
}

type createAdminAIPostRequest struct {
	Body string `json:"body"`
}

type generateAdminAIPostContentRequest struct {
	Keywords     []string `json:"keywords"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	StylePrompt  string   `json:"style_prompt"`
}

type adminAIAccountResponse struct {
	Account any `json:"account"`
	User    any `json:"user"`
}

type adminAIGenerationsResponse struct {
	Generations any `json:"generations"`
}

type adminAIPreviewResponse struct {
	Generation any `json:"generation"`
}

type adminAIPostResponse struct {
	Post       any `json:"post,omitempty"`
	Generation any `json:"generation"`
}

type adminAIGeneratedPostResponse struct {
	Body   string `json:"body"`
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
}

type adminUsersResponse struct {
	Users      []*admin.UserSummary `json:"users"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type adminPostsResponse struct {
	Posts      []*admin.PostSummary `json:"posts"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

func NewAdminHandler(app *app.App) *AdminHandler {
	return &AdminHandler{app: app}
}

func (h *AdminHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	adminOnly := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware(RequireAdmin(h.app, next))
	}

	sm.HandleFunc("GET /admin/stats", adminOnly(h.Stats))
	sm.HandleFunc("GET /admin/users", adminOnly(h.ListUsers))
	sm.HandleFunc("POST /admin/users/{id}/block", adminOnly(h.BlockUser))
	sm.HandleFunc("GET /admin/posts", adminOnly(h.ListPosts))
	sm.HandleFunc("DELETE /admin/posts/{id}", adminOnly(h.DeletePost))

	sm.HandleFunc("POST /admin/ai/accounts", adminOnly(h.CreateAIAccount))
	sm.HandleFunc("PATCH /admin/ai/accounts/{id}", adminOnly(h.UpdateAIAccount))
	sm.HandleFunc("GET /admin/ai/status", adminOnly(h.AIStatus))
	sm.HandleFunc("GET /admin/ai/accounts/{id}", adminOnly(h.ShowAIAccount))
	sm.HandleFunc("GET /admin/ai/accounts/{id}/generations", adminOnly(h.ListAIGenerations))
	sm.HandleFunc("POST /admin/ai/accounts/{id}/preview", adminOnly(h.CreateAIPreview))
	sm.HandleFunc("POST /admin/ai/accounts/{id}/posts", adminOnly(h.CreateAIPost))
	sm.HandleFunc("POST /admin/ai/tools/generate-post", adminOnly(h.GenerateAIPostContent))
}

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.app.Admin.Stats(r.Context())
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Admin.Stats")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Respond(w, http.StatusOK, stats, h.app.Logger)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := defaultAdminUsersLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxAdminUsersLimit {
			parsedLimit = maxAdminUsersLimit
		}
		limit = parsedLimit
	}

	cursor, err := DecodeIDCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	users, nextCursor, err := h.app.Admin.ListUsers(r.Context(), admin.ListUsersInput{
		Query:       strings.TrimSpace(r.URL.Query().Get("q")),
		AccountType: strings.TrimSpace(r.URL.Query().Get("account_type")),
		Cursor:      cursor,
		Limit:       limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrInvalidLimit), errors.Is(err, admin.ErrInvalidAccountType):
			http.Error(w, "invalid query", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.ListUsers")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	h.setAdminUserAvatarURLs(users)
	encodedCursor, err := EncodeIDCursor(nextCursor)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error encoding admin users cursor")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Respond(w, http.StatusOK, adminUsersResponse{
		Users:      users,
		NextCursor: encodedCursor,
	}, h.app.Logger)
}

func (h *AdminHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.app.Admin.BlockUser(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrInvalidUserID), errors.Is(err, admin.ErrCannotBlockSystem):
			http.Error(w, "invalid block request", http.StatusBadRequest)
		case errors.Is(err, admin.ErrUserNotFound):
			w.WriteHeader(http.StatusNotFound)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.BlockUser")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	h.setAdminUserAvatarURL(user)
	Respond(w, http.StatusOK, user, h.app.Logger)
}

func (h *AdminHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	limit := defaultAdminPostsLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxAdminPostsLimit {
			parsedLimit = maxAdminPostsLimit
		}
		limit = parsedLimit
	}

	cursor, err := DecodeIDCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	var userID *uint64
	if rawUserID := strings.TrimSpace(r.URL.Query().Get("user_id")); rawUserID != "" {
		parsedUserID, err := strconv.ParseUint(rawUserID, 10, 64)
		if err != nil {
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}
		userID = &parsedUserID
	}

	posts, nextCursor, err := h.app.Admin.ListPosts(r.Context(), admin.ListPostsInput{
		UserID: userID,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrInvalidLimit):
			http.Error(w, "invalid query", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.ListPosts")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	h.setAdminPostResponseFields(posts)
	encodedCursor, err := EncodeIDCursor(nextCursor)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error encoding admin posts cursor")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Respond(w, http.StatusOK, adminPostsResponse{
		Posts:      posts,
		NextCursor: encodedCursor,
	}, h.app.Logger)
}

func (h *AdminHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	deleted, err := h.app.Admin.DeletePost(r.Context(), postID)
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrInvalidPostID):
			http.Error(w, "invalid post id", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.DeletePost")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if !deleted {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	Respond(w, http.StatusOK, nil, h.app.Logger)
}

func (h *AdminHandler) CreateAIAccount(w http.ResponseWriter, r *http.Request) {
	req, err := DecodeRequestBody[createAdminAIAccountRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid admin ai account create json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	minPostsPerDay := 1
	if req.MinPostsPerDay != nil {
		minPostsPerDay = *req.MinPostsPerDay
	}

	maxPostsPerDay := 2
	if req.MaxPostsPerDay != nil {
		maxPostsPerDay = *req.MaxPostsPerDay
	}

	record, err := h.app.Admin.CreateAccount(r.Context(), admin.CreateAccountInput{
		Email:          req.Email,
		Username:       req.Username,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		About:          req.About,
		Enabled:        enabled,
		Topic:          req.Topic,
		Description:    req.Description,
		SystemPrompt:   req.SystemPrompt,
		StylePrompt:    req.StylePrompt,
		MinPostsPerDay: minPostsPerDay,
		MaxPostsPerDay: maxPostsPerDay,
		NextGenerateAt: req.NextGenerateAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrInvalidEmail),
			errors.Is(err, admin.ErrInvalidUsername),
			errors.Is(err, admin.ErrInvalidTopic),
			errors.Is(err, admin.ErrInvalidSystemPrompt),
			errors.Is(err, admin.ErrInvalidPostsPerDay), errors.Is(err, admin.ErrInvalidProfile):
			h.app.Logger.Warn().Str("email", req.Email).Str("username", req.Username).Msg("invalid admin ai account payload")
			http.Error(w, "invalid ai account payload", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.CreateAccount")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	record.User.AvatarURL = AvatarURL(h.app, record.User.ID)
	h.app.Logger.Info().
		Uint64("ai_account_id", record.Account.ID).
		Uint64("user_id", record.User.ID).
		Str("username", record.User.Username).
		Msg("admin ai account created")

	Respond(w, http.StatusOK, adminAIAccountResponse{
		Account: record.Account,
		User:    adminUser(record.User),
	}, h.app.Logger)
}

func (h *AdminHandler) ShowAIAccount(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid ai account id", http.StatusBadRequest)
		return
	}

	record, err := h.app.Admin.GetAccount(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, admin.ErrAIAccountNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.app.Logger.Error().Err(err).Msg("error in Admin.GetAccount")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	record.User.AvatarURL = AvatarURL(h.app, record.User.ID)
	h.app.Logger.Debug().Uint64("ai_account_id", record.Account.ID).Uint64("user_id", record.User.ID).Msg("admin ai account fetched")

	Respond(w, http.StatusOK, adminAIAccountResponse{
		Account: record.Account,
		User:    adminUser(record.User),
	}, h.app.Logger)
}

func (h *AdminHandler) ListAIGenerations(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid ai account id", http.StatusBadRequest)
		return
	}

	limit := defaultAdminGenerationsLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxAdminGenerationsLimit {
			parsedLimit = maxAdminGenerationsLimit
		}
		limit = parsedLimit
	}

	generations, err := h.app.Admin.ListPostGenerations(r.Context(), accountID, limit)
	if err != nil {
		if errors.Is(err, admin.ErrAIAccountNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.app.Logger.Error().Err(err).Msg("error in Admin.ListPostGenerations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.app.Logger.Debug().Uint64("ai_account_id", accountID).Int("limit", limit).Msg("admin ai generations fetched")

	Respond(w, http.StatusOK, adminAIGenerationsResponse{Generations: generations}, h.app.Logger)
}

func (h *AdminHandler) CreateAIPreview(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid ai account id", http.StatusBadRequest)
		return
	}

	generation, err := h.app.Admin.CreatePreview(r.Context(), accountID)
	if err != nil {
		switch {
		case errors.Is(err, aiaccounts.ErrClaimLost):
			http.Error(w, "Account is currently posting; retry shortly", http.StatusConflict)
		case errors.Is(err, admin.ErrAIAccountNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, admin.ErrPreviewRejected):
			h.app.Logger.Warn().Uint64("ai_account_id", accountID).Str("status", generation.Status).Msg("admin ai preview rejected")
			Respond(w, http.StatusUnprocessableEntity, adminAIPreviewResponse{Generation: generation}, h.app.Logger)
		case errors.Is(err, admin.ErrPreviewGenerationFailed):
			h.app.Logger.Warn().Uint64("ai_account_id", accountID).Msg("admin ai preview generation failed")
			Respond(w, http.StatusBadGateway, adminAIPreviewResponse{Generation: generation}, h.app.Logger)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.CreatePreview")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	h.app.Logger.Info().Uint64("ai_account_id", accountID).Uint64("generation_id", generation.ID).Msg("admin ai preview created")

	Respond(w, http.StatusOK, adminAIPreviewResponse{Generation: generation}, h.app.Logger)
}

func (h *AdminHandler) CreateAIPost(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid ai account id", http.StatusBadRequest)
		return
	}

	req, err := DecodeRequestBody[createAdminAIPostRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid admin ai post create json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := h.app.Admin.CreatePost(r.Context(), accountID, req.Body)
	if err != nil {
		switch {
		case errors.Is(err, aiaccounts.ErrClaimLost):
			http.Error(w, "Account is currently posting; retry shortly", http.StatusConflict)
		case errors.Is(err, admin.ErrAIAccountNotFound):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, admin.ErrPreviewRejected):
			h.app.Logger.Warn().Uint64("ai_account_id", accountID).Msg("admin ai post rejected")
			Respond(w, http.StatusUnprocessableEntity, adminAIPostResponse{Generation: result.Generation}, h.app.Logger)
		case errors.Is(err, admin.ErrPostGenerationFailed):
			h.app.Logger.Warn().Uint64("ai_account_id", accountID).Msg("admin ai post generation failed")
			Respond(w, http.StatusBadGateway, adminAIPostResponse{Generation: result.Generation}, h.app.Logger)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.CreatePost")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	createdPost, err := h.app.Posts.GetByID(r.Context(), result.Post.ID, 0)
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
		Uint64("ai_account_id", accountID).
		Uint64("post_id", createdPost.ID).
		Uint64("generation_id", result.Generation.ID).
		Str("source", createdPost.Source).
		Msg("admin ai post created")

	Respond(w, http.StatusOK, adminAIPostResponse{
		Post:       createdPost,
		Generation: result.Generation,
	}, h.app.Logger)
}

func (h *AdminHandler) GenerateAIPostContent(w http.ResponseWriter, r *http.Request) {
	req, err := DecodeRequestBody[generateAdminAIPostContentRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid admin ai tools json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := h.app.Admin.GeneratePostContent(r.Context(), admin.GeneratePostContentInput{
		Keywords:     req.Keywords,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		StylePrompt:  req.StylePrompt,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrInvalidKeywords), errors.Is(err, admin.ErrInvalidDescription):
			http.Error(w, "invalid ai generation payload", http.StatusBadRequest)
		case errors.Is(err, admin.ErrContentGenerationFailed):
			http.Error(w, "ai content generation failed", http.StatusBadGateway)
		default:
			h.app.Logger.Error().Err(err).Msg("error in Admin.GeneratePostContent")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	h.app.Logger.Info().
		Strs("keywords", req.Keywords).
		Str("model", result.Model).
		Msg("admin ai post content generated")

	Respond(w, http.StatusOK, adminAIGeneratedPostResponse{
		Body:   result.Body,
		Prompt: result.Prompt,
		Model:  result.Model,
	}, h.app.Logger)
}

func (h *AdminHandler) setAdminUserAvatarURL(user *admin.UserSummary) {
	if user == nil {
		return
	}

	user.AvatarURL = AvatarURL(h.app, user.ID)
}

func (h *AdminHandler) setAdminUserAvatarURLs(users []*admin.UserSummary) {
	for _, user := range users {
		h.setAdminUserAvatarURL(user)
	}
}

func (h *AdminHandler) setAdminPostResponseFields(posts []*admin.PostSummary) {
	for _, post := range posts {
		if post == nil {
			continue
		}
		post.User.AvatarURL = AvatarURL(h.app, post.User.UserID)
		post.URL = PostURLPath(post.ID)
	}
}

// Browser clients must not round 64-bit IDs through JavaScript's Number type.
func adminUser(user *users.User) any {
	return struct {
		*users.User
		ID string `json:"id"`
	}{user, strconv.FormatUint(user.ID, 10)}
}

func (h *AdminHandler) UpdateAIAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid ai account id", 400)
		return
	}
	input, err := DecodeRequestBody[admin.UpdateAccountInput](w, r, 1<<20)
	if err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	record, err := h.app.Admin.UpdateAccount(r.Context(), id, input)
	if err != nil {
		switch {
		case errors.Is(err, admin.ErrAIAccountNotFound), errors.Is(err, admin.ErrUserNotFound):
			http.Error(w, "account not found or unavailable", 404)
		case errors.Is(err, admin.ErrInvalidTopic), errors.Is(err, admin.ErrInvalidSystemPrompt), errors.Is(err, admin.ErrInvalidPostsPerDay), errors.Is(err, admin.ErrInvalidProfile):
			http.Error(w, err.Error(), 400)
		default:
			h.app.Logger.Error().Err(err).Uint64("ai_account_id", id).Msg("update AI account failed")
			w.WriteHeader(500)
		}
		return
	}
	record.User.AvatarURL = AvatarURL(h.app, record.User.ID)
	h.app.Logger.Info().Uint64("ai_account_id", id).Bool("enabled", record.Account.Enabled).Msg("AI account updated; schedule reset")
	Respond(w, 200, adminAIAccountResponse{Account: record.Account, User: adminUser(record.User)}, h.app.Logger)
}

func (h *AdminHandler) AIStatus(w http.ResponseWriter, r *http.Request) {
	cfg := h.app.Config.AIPosterConfig()
	tools := h.app.Config.AIToolsConfig()
	Respond(w, 200, map[string]any{"provider": tools.Provider, "model": tools.OpenAI.Model, "development_mode": cfg.DevelopmentMode, "development_min_interval_seconds": cfg.DevelopmentMinIntervalSeconds, "development_max_interval_seconds": cfg.DevelopmentMaxIntervalSeconds, "poll_interval_seconds": cfg.PollIntervalSeconds}, h.app.Logger)
}
