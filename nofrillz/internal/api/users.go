package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nofrillz/internal/apns"
	"nofrillz/internal/app"
	"nofrillz/internal/notifications"
	"nofrillz/internal/posts"
	"nofrillz/internal/users"
)

type UsersHandler struct {
	app *app.App
}

const (
	defaultPostsLimit     = 20
	maxPostsLimit         = 100
	defaultSearchLimit    = 20
	maxSearchLimit        = 100
	defaultFollowersLimit = 25
	maxFollowersLimit     = 100
)

type CreateUserRequest struct {
	AIModelPreference *string `json:"ai_model_preference"`
	Email             string  `json:"email"`
	Username          string  `json:"username"`
	FirstName         string  `json:"first_name"`
	LastName          string  `json:"last_name"`
	About             string  `json:"about"`
	Password          string  `json:"password"`
}

type SearchUsersResponse struct {
	Users []*users.User `json:"users"`
}

type FollowersResponse struct {
	Followers  []*users.User `json:"followers"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type UpsertAPNSDeviceTokenRequest struct {
	Token string `json:"token"`
}

type APNSDeviceTokenResponse struct {
	Token string `json:"token"`
}

type UpdatePushNotificationSettingsRequest struct {
	Enabled      *bool `json:"enabled"`
	NewFollowers *bool `json:"new_followers"`
	NewLikes     *bool `json:"new_likes"`
	Replies      *bool `json:"replies"`
}

func NewUsersHandler(app *app.App) *UsersHandler {
	return &UsersHandler{
		app: app,
	}
}

func (h *UsersHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware(RequireAuthentication(next))
	}

	sm.HandleFunc("POST /users", middleware(h.Create))
	sm.HandleFunc("GET /users/search", requireAuth(h.Search))
	sm.HandleFunc("GET /users/followers", requireAuth(h.Followers))
	sm.HandleFunc("GET /users/following", requireAuth(h.Following))
	sm.HandleFunc("PUT /users/apns/device-tokens", requireAuth(h.UpsertAPNSDeviceToken))
	sm.HandleFunc("GET /users/push-notification-settings", requireAuth(h.ShowPushNotificationSettings))
	sm.HandleFunc("PATCH /users/push-notification-settings", requireAuth(h.UpdatePushNotificationSettings))
	sm.HandleFunc("GET /users/{id}", requireAuth(h.Show))
	sm.HandleFunc("GET /users/{id}/posts", requireAuth(h.Posts))
	sm.HandleFunc("POST /users/{id}/follow", requireAuth(h.Follow))
	sm.HandleFunc("POST /users/{id}/unfollow", requireAuth(h.Unfollow))
	sm.HandleFunc("GET /users/{id}/followers", requireAuth(h.Followers))
	sm.HandleFunc("GET /users/{id}/following", requireAuth(h.Following))
}

func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	if !AllowSignup(w, r, h.app) {
		return
	}

	signup, err := DecodeRequestBody[CreateUserRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid signup json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	signup.Email = strings.TrimSpace(signup.Email)
	signup.Username = strings.TrimSpace(signup.Username)

	if !users.ValidateEmail(signup.Email) || !users.ValidateUsername(signup.Username) || signup.Password == "" {
		h.app.Logger.Warn().Str("email", signup.Email).Str("username", signup.Username).Msg("invalid signup payload")
		http.Error(w, "invalid signup payload", http.StatusBadRequest)
		return
	}

	if signup.AIModelPreference != nil {
		if h.app.AIModels == nil {
			http.Error(w, "AI model catalog unavailable", 503)
			return
		}
		if _, ok := h.app.AIModels.Get(*signup.AIModelPreference); !ok {
			http.Error(w, "unknown AI model option", 400)
			return
		}
	}
	passwordHash, passwordSalt, err := users.GeneratePasswordHashAndSalt(signup.Password)
	if err != nil {
		h.app.Logger.Warn().Str("username", signup.Username).Msg("invalid signup password")
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}

	user := users.User{
		AIModelPreference: signup.AIModelPreference,
		Email:             signup.Email,
		Username:          signup.Username,
		FirstName:         strings.TrimSpace(signup.FirstName),
		LastName:          strings.TrimSpace(signup.LastName),
		About:             strings.TrimSpace(signup.About),
		AccountType:       users.AccountTypeHuman,
	}

	user.ID = h.app.IDGenerator.MustNext()
	user.PasswordHash = passwordHash
	user.PasswordSalt = passwordSalt

	err = h.app.Users.Create(r.Context(), &user)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in repository.Create")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.setAvatarURL(&user)
	h.app.Logger.Info().Uint64("user_id", user.ID).Str("username", user.Username).Msg("user created")

	Respond(w, http.StatusOK, user, h.app.Logger)
}

func (h *UsersHandler) Show(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	userId, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized get user attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.app.Users.GetByIDForRequester(r.Context(), userId, authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Users.GetByIDForRequester")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	h.setAvatarURL(user)
	h.app.Logger.Debug().Uint64("user_id", user.ID).Uint64("requester_user_id", authenticatedUser.ID).Msg("user fetched")

	Respond(w, http.StatusOK, user, h.app.Logger)
}

func (h *UsersHandler) Search(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}
	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized user search attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.app.Logger.Warn().Uint64("requester_user_id", authenticatedUser.ID).Msg("invalid user search query")
		http.Error(w, "invalid query", http.StatusBadRequest)
		return
	}

	limit := defaultSearchLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxSearchLimit {
			parsedLimit = maxSearchLimit
		}
		limit = parsedLimit
	}

	searchResults, err := h.app.Users.SearchByUsernamePrefixForRequester(r.Context(), query, authenticatedUser.ID, limit)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Users.SearchByUsernamePrefixForRequester")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.setAvatarURLs(searchResults)
	h.app.Logger.Debug().
		Uint64("requester_user_id", authenticatedUser.ID).
		Str("query", query).
		Int("limit", limit).
		Int("results", len(searchResults)).
		Msg("user search completed")

	Respond(w, http.StatusOK, SearchUsersResponse{Users: searchResults}, h.app.Logger)
}

func (h *UsersHandler) UpsertAPNSDeviceToken(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	request, err := DecodeRequestBody[UpsertAPNSDeviceTokenRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Uint64("user_id", authenticatedUser.ID).Msg("invalid apns device token json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	token, err := h.app.APNS.RegisterToken(r.Context(), authenticatedUser.ID, request.Token)
	if err != nil {
		switch {
		case errors.Is(err, apns.ErrInvalidUserID), errors.Is(err, apns.ErrInvalidToken):
			http.Error(w, "invalid token", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Uint64("user_id", authenticatedUser.ID).Msg("error registering apns device token")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	h.app.Logger.Info().Uint64("user_id", authenticatedUser.ID).Msg("registered apns device token")
	Respond(w, http.StatusOK, APNSDeviceTokenResponse{Token: token}, h.app.Logger)
}

func (h *UsersHandler) ShowPushNotificationSettings(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}
	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	settings, err := h.app.Notifications.GetByUserID(r.Context(), authenticatedUser.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Uint64("user_id", authenticatedUser.ID).Msg("error fetching push notification settings")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	Respond(w, http.StatusOK, settings, h.app.Logger)
}

func (h *UsersHandler) UpdatePushNotificationSettings(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	request, err := DecodeRequestBody[UpdatePushNotificationSettingsRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Uint64("user_id", authenticatedUser.ID).Msg("invalid push notification settings json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	settings, err := h.app.Notifications.UpdateUserSettings(r.Context(), authenticatedUser.ID, notifications.UpdateSettingsInput{
		Enabled:      request.Enabled,
		NewFollowers: request.NewFollowers,
		NewLikes:     request.NewLikes,
		Replies:      request.Replies,
	})
	if err != nil {
		switch {
		case errors.Is(err, notifications.ErrInvalidUserID):
			http.Error(w, "invalid user", http.StatusBadRequest)
		default:
			h.app.Logger.Error().Err(err).Uint64("user_id", authenticatedUser.ID).Msg("error updating push notification settings")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	h.app.Logger.Info().Uint64("user_id", authenticatedUser.ID).Msg("updated push notification settings")
	Respond(w, http.StatusOK, settings, h.app.Logger)
}

func (h *UsersHandler) Posts(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	userId, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	limit := defaultPostsLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxPostsLimit {
			parsedLimit = maxPostsLimit
		}
		limit = parsedLimit
	}

	var beforeId uint64
	rawBeforeId := r.URL.Query().Get("before_id")
	if rawBeforeId != "" {
		beforeId, err = strconv.ParseUint(rawBeforeId, 10, 64)
		if err != nil {
			http.Error(w, "invalid before_id", http.StatusBadRequest)
			return
		}
	}

	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userPosts, err := h.app.Posts.ListByUser(r.Context(), userId, authenticatedUser.ID, beforeId, limit)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Posts.ListByUser")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if userPosts == nil {
		userPosts = []*posts.Post{}
	}
	SetPostResponseFieldsMany(h.app, userPosts)
	h.app.Logger.Debug().
		Uint64("user_id", userId).
		Uint64("before_id", beforeId).
		Int("limit", limit).
		Int("results", len(userPosts)).
		Msg("user posts fetched")

	Respond(w, http.StatusOK, userPosts, h.app.Logger)
}

func (h *UsersHandler) Follow(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	followingId, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		h.app.Logger.Warn().Str("user_id", r.PathValue("id")).Msg("invalid follow user id")
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Uint64("following_id", followingId).Msg("unauthorized follow attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	followerId := authenticatedUser.ID

	if followingId == followerId {
		h.app.Logger.Warn().Uint64("user_id", followerId).Msg("cannot follow self")
		http.Error(w, "cannot follow yourself", http.StatusBadRequest)
		return
	}

	if !AllowWriteByUser(w, h.app, followerId) {
		return
	}

	created, err := h.app.Follows.Follow(r.Context(), followerId, followingId)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Follows.Follow")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if created {
		h.app.Logger.Info().Uint64("follower_id", followerId).Uint64("following_id", followingId).Msg("user followed")
		if err := h.app.Notifications.NotifyNewFollower(r.Context(), followingId, authenticatedUser.Username); err != nil {
			h.app.Logger.Error().Err(err).Uint64("follower_id", followerId).Uint64("following_id", followingId).Msg("failed to send new follower push notification")
		}
	} else {
		h.app.Logger.Debug().Uint64("follower_id", followerId).Uint64("following_id", followingId).Msg("follow already existed")
	}

	Respond(w, http.StatusOK, nil, h.app.Logger)
}

func (h *UsersHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	followingId, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		h.app.Logger.Warn().Str("user_id", r.PathValue("id")).Msg("invalid unfollow user id")
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Uint64("following_id", followingId).Msg("unauthorized unfollow attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	followerId := authenticatedUser.ID

	if followingId == followerId {
		h.app.Logger.Warn().Uint64("user_id", followerId).Msg("cannot unfollow self")
		http.Error(w, "cannot unfollow yourself", http.StatusBadRequest)
		return
	}

	if !AllowWriteByUser(w, h.app, followerId) {
		return
	}

	err = h.app.Follows.Unfollow(r.Context(), followerId, followingId)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Follows.Unfollow")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.app.Logger.Info().Uint64("follower_id", followerId).Uint64("following_id", followingId).Msg("user unfollowed")

	Respond(w, http.StatusOK, nil, h.app.Logger)
}

func (h *UsersHandler) Followers(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var userId uint64
	rawUserID := r.PathValue("id")
	if rawUserID == "" {
		userId = authenticatedUser.ID
	} else {
		parsedUserID, err := strconv.ParseUint(rawUserID, 10, 64)
		if err != nil {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		userId = parsedUserID
	}

	limit := defaultFollowersLimit
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxFollowersLimit {
			parsedLimit = maxFollowersLimit
		}
		limit = parsedLimit
	}

	cursor, err := DecodeIDCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		http.Error(w, "invalid cursor", http.StatusBadRequest)
		return
	}

	followers, nextCursor, err := h.app.Users.ListFollowersPageForRequester(r.Context(), userId, authenticatedUser.ID, cursor, limit)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Users.ListFollowersPageForRequester")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if followers == nil {
		followers = []*users.User{}
	}
	h.setAvatarURLs(followers)

	nextCursorValue, err := EncodeIDCursor(nextCursor)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error encoding followers cursor")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := FollowersResponse{
		Followers:  followers,
		NextCursor: nextCursorValue,
	}
	h.app.Logger.Debug().
		Uint64("user_id", userId).
		Uint64("requester_user_id", authenticatedUser.ID).
		Int("limit", limit).
		Int("results", len(followers)).
		Msg("followers fetched")
	Respond(w, http.StatusOK, response, h.app.Logger)
}

func (h *UsersHandler) avatarBaseURL() string {
	if h.app == nil || h.app.Config == nil {
		return ""
	}

	cfg := h.app.Config.BaseURLsConfig()
	if cfg == nil {
		return ""
	}

	return strings.TrimRight(cfg.Avatar, "/")
}

func (h *UsersHandler) setAvatarURL(user *users.User) {
	if user == nil {
		return
	}

	baseURL := h.avatarBaseURL()
	if baseURL == "" {
		return
	}

	user.AvatarURL = baseURL + "/users/" + strconv.FormatUint(user.ID, 10) + "/avatar.jpeg"
}

func (h *UsersHandler) setAvatarURLs(usersList []*users.User) {
	for _, user := range usersList {
		h.setAvatarURL(user)
	}
}

func (h *UsersHandler) Following(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	if !AllowReadByRequester(w, r, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var userId uint64
	rawUserID := r.PathValue("id")
	if rawUserID == "" {
		userId = authenticatedUser.ID
	} else {
		parsedUserID, err := strconv.ParseUint(rawUserID, 10, 64)
		if err != nil {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		userId = parsedUserID
	}

	following, err := h.app.Follows.ListFollowing(r.Context(), userId)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Follows.ListFollowing")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if following == nil {
		following = []uint64{}
	}
	h.app.Logger.Debug().Uint64("user_id", userId).Int("results", len(following)).Msg("following fetched")

	Respond(w, http.StatusOK, following, h.app.Logger)
}
