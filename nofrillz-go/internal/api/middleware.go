package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"nofrillz/internal/app"
	"nofrillz/internal/sessions"
	"nofrillz/internal/users"
)

const authenticatedUserKey = "authenticated_user"

type AuthenticatedUser struct {
	ID        uint64 `json:"id"`
	SessionID uint64 `json:"session_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	About     string `json:"about"`
}

func SessionTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	authorization := r.Header.Get("Authorization")
	if authorization != "" {
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	return ""
}

func Middleware(app *app.App, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		responseWriter := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		token := SessionTokenFromRequest(r)
		if token == "" {
			next(responseWriter, r)
			logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
			return
		}

		if app == nil || app.Sessions == nil || app.Config == nil {
			next(responseWriter, r)
			logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
			return
		}

		claims, err := sessions.ParseAccessToken(app.Config.SessionConfig().JWTSecret, token)
		if err != nil {
			if app.Logger != nil {
				app.Logger.Debug().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("request provided invalid bearer token")
			}
			next(responseWriter, r)
			logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
			return
		}

		activeSession, err := app.Sessions.GetActiveByID(r.Context(), claims.SessionID)
		if err != nil {
			app.Logger.Error().Err(err).Msg("error in Sessions.GetActiveByID")
			next(responseWriter, r)
			logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
			return
		}
		if activeSession == nil {
			if app.Logger != nil {
				app.Logger.Debug().
					Uint64("session_id", claims.SessionID).
					Uint64("user_id", claims.UserID).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("request session not active")
			}
			next(responseWriter, r)
			logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
			return
		}

		var authenticatedDBUser *users.User
		if app.Users != nil {
			authenticatedDBUser, err := app.Users.GetByID(r.Context(), claims.UserID)
			if err != nil {
				app.Logger.Error().Err(err).Msg("error in Users.GetByID")
				next(responseWriter, r)
				logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
				return
			}
			if authenticatedDBUser == nil || authenticatedDBUser.Deleted != nil || authenticatedDBUser.Blocked != nil {
				if app.Logger != nil {
					app.Logger.Debug().
						Uint64("user_id", claims.UserID).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("request user is inactive")
				}
				next(responseWriter, r)
				logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
				return
			}
		}

		authenticatedUser := AuthenticatedUser{
			ID:        claims.UserID,
			SessionID: claims.SessionID,
		}
		if authenticatedDBUser != nil {
			authenticatedUser.Email = authenticatedDBUser.Email
			authenticatedUser.Username = authenticatedDBUser.Username
			authenticatedUser.FirstName = authenticatedDBUser.FirstName
			authenticatedUser.LastName = authenticatedDBUser.LastName
			authenticatedUser.About = authenticatedDBUser.About
		}

		ctx := context.WithValue(r.Context(), authenticatedUserKey, authenticatedUser)
		r = r.WithContext(ctx)

		next(responseWriter, r)
		logRequestCompletion(app, r, responseWriter.status, responseWriter.bytes, time.Since(start))
	}
}

func RequireAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := AuthenticatedUserForRequest(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func AdminAPIKeyFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	apiKey := strings.TrimSpace(r.Header.Get("X-Admin-API-Key"))
	if apiKey != "" {
		return apiKey
	}

	authorization := r.Header.Get("Authorization")
	if authorization != "" {
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	return ""
}

func RequireAdmin(app *app.App, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app == nil || app.Config == nil {
			http.Error(w, "admin api not configured", http.StatusServiceUnavailable)
			return
		}

		expected := strings.TrimSpace(app.Config.AdminConfig().APIKey)
		if expected == "" {
			if app.Logger != nil {
				app.Logger.Warn().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("admin api requested but api key is not configured")
			}
			http.Error(w, "admin api not configured", http.StatusServiceUnavailable)
			return
		}

		if AdminAPIKeyFromRequest(r) != expected {
			if app.Logger != nil {
				app.Logger.Warn().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("ip", ClientIP(r)).
					Msg("admin api authentication failed")
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func AuthenticatedUserForRequest(r *http.Request) (*AuthenticatedUser, bool) {
	if r == nil {
		return nil, false
	}

	value := r.Context().Value(authenticatedUserKey)
	if value == nil {
		return nil, false
	}

	if user, ok := value.(AuthenticatedUser); ok {
		return &user, true
	}

	return nil, false
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

func logRequestCompletion(app *app.App, r *http.Request, status int, bytes int, duration time.Duration) {
	if app == nil || app.Logger == nil || r == nil {
		return
	}

	event := app.Logger.Debug()
	switch {
	case status >= http.StatusInternalServerError:
		event = app.Logger.Error()
	case status >= http.StatusBadRequest:
		event = app.Logger.Warn()
	}

	event.
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("ip", ClientIP(r)).
		Int("status", status).
		Int("bytes", bytes).
		Dur("duration", duration)

	if pattern := r.Pattern; pattern != "" {
		event.Str("pattern", pattern)
	}

	if authenticatedUser, ok := AuthenticatedUserForRequest(r); ok {
		event.Uint64("user_id", authenticatedUser.ID)
		event.Uint64("session_id", authenticatedUser.SessionID)
	}

	event.Msg("request completed")
}
