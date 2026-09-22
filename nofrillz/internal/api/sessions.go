package api

import (
	"net/http"
	"strings"

	"nofrillz/internal/app"
	"nofrillz/internal/sessions"
	"nofrillz/internal/users"
)

type SessionsHandler struct {
	app *app.App
}

type CreateSessionRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateSessionResponse struct {
	User         AuthenticatedUser `json:"user"`
	SessionToken string            `json:"session_token"`
	RefreshToken string            `json:"refresh_token"`
}

type RefreshSessionRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshSessionResponse struct {
	SessionToken string `json:"session_token"`
	RefreshToken string `json:"refresh_token"`
}

func NewSessionsHandler(app *app.App) *SessionsHandler {
	return &SessionsHandler{
		app: app,
	}
}

func (h *SessionsHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware(RequireAuthentication(next))
	}

	sm.HandleFunc("POST /sessions", middleware(h.Create))
	sm.HandleFunc("POST /sessions/refresh", middleware(h.Refresh))
	sm.HandleFunc("DELETE /sessions/current", requireAuth(h.DeleteCurrent))
}

func (h *SessionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	credentials, err := DecodeRequestBody[CreateSessionRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid session create json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	credentials.Email = strings.TrimSpace(credentials.Email)

	if !users.ValidateEmail(credentials.Email) || credentials.Password == "" {
		h.app.Logger.Warn().Str("email", credentials.Email).Msg("invalid credentials payload")
		http.Error(w, "invalid credentials", http.StatusBadRequest)
		return
	}

	user, err := h.app.Users.GetByEmail(r.Context(), credentials.Email)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Users.GetByEmail")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if user == nil || !users.ValidatePassword(credentials.Password, user.PasswordHash, user.PasswordSalt) {
		h.app.Logger.Warn().Str("email", credentials.Email).Msg("login failed")
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if user.Deleted != nil || user.Blocked != nil {
		h.app.Logger.Warn().Str("email", credentials.Email).Msg("login attempted for inactive user")
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, user.ID) {
		return
	}

	accessToken, refreshToken, sessionID, err := issueTokensForUser(h.app, r.Context(), user.ID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error issuing tokens for user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authenticatedUser := AuthenticatedUser{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		About:     user.About,
	}

	response := CreateSessionResponse{
		User:         authenticatedUser,
		SessionToken: accessToken,
		RefreshToken: refreshToken,
	}
	h.app.Logger.Info().Uint64("user_id", user.ID).Uint64("session_id", sessionID).Str("email", user.Email).Msg("session created")
	Respond(w, http.StatusOK, response, h.app.Logger)
}

func (h *SessionsHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	refreshRequest, err := DecodeRequestBody[RefreshSessionRequest](w, r, 1<<20)
	if err != nil {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("invalid session refresh json")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if refreshRequest.RefreshToken == "" {
		h.app.Logger.Warn().Msg("missing refresh token")
		http.Error(w, "invalid refresh token", http.StatusBadRequest)
		return
	}

	refreshTokenHash := sessions.HashToken(refreshRequest.RefreshToken)

	refreshSession, err := h.app.Sessions.GetActiveRefreshByTokenHash(r.Context(), refreshTokenHash)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Sessions.GetActiveRefreshByTokenHash")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if refreshSession == nil {
		h.app.Logger.Warn().Msg("refresh token not active")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.app.Users.GetByID(r.Context(), refreshSession.UserID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Users.GetByID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if user == nil || user.Deleted != nil || user.Blocked != nil {
		h.app.Logger.Warn().Uint64("user_id", refreshSession.UserID).Msg("refresh attempted for inactive user")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, refreshSession.UserID) {
		return
	}

	revoked, err := h.app.Sessions.RevokeRefreshByTokenHash(r.Context(), refreshTokenHash)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Sessions.RevokeRefreshByTokenHash")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !revoked {
		h.app.Logger.Warn().Uint64("session_id", refreshSession.SessionID).Msg("refresh token revocation failed")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := issueTokensForSession(h.app, r.Context(), refreshSession.UserID, refreshSession.SessionID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error issuing tokens for user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := RefreshSessionResponse{
		SessionToken: accessToken,
		RefreshToken: refreshToken,
	}
	h.app.Logger.Info().Uint64("user_id", refreshSession.UserID).Uint64("session_id", refreshSession.SessionID).Msg("session refreshed")
	Respond(w, http.StatusOK, response, h.app.Logger)
}

func (h *SessionsHandler) DeleteCurrent(w http.ResponseWriter, r *http.Request) {
	if !AllowWriteGlobal(w, h.app) {
		return
	}

	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		h.app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("unauthorized delete current session attempt")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !AllowWriteByUser(w, h.app, authenticatedUser.ID) {
		return
	}

	if authenticatedUser.SessionID == 0 {
		h.app.Logger.Warn().Uint64("user_id", authenticatedUser.ID).Msg("delete current session missing session id")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	revoked, err := h.app.Sessions.RevokeByID(r.Context(), authenticatedUser.SessionID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Sessions.RevokeByID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !revoked {
		h.app.Logger.Warn().Uint64("session_id", authenticatedUser.SessionID).Msg("current session revoke returned false")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	_, err = h.app.Sessions.RevokeRefreshBySessionID(r.Context(), authenticatedUser.SessionID)
	if err != nil {
		h.app.Logger.Error().Err(err).Msg("error in Sessions.RevokeRefreshBySessionID")
	}
	h.app.Logger.Info().Uint64("user_id", authenticatedUser.ID).Uint64("session_id", authenticatedUser.SessionID).Msg("session deleted")

	Respond(w, http.StatusOK, nil, h.app.Logger)
}
