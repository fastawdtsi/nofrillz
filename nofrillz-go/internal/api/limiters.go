package api

import (
	"net/http"
	"strconv"

	"nofrillz/internal/app"
)

func AllowSignup(w http.ResponseWriter, r *http.Request, app *app.App) bool {
	ip := ClientIP(r)

	if !app.Limiters.SignupIP.Allow(ip) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Str("ip", ip).Str("method", r.Method).Str("path", r.URL.Path).Msg("signup rate limited by ip")
		}
		w.Header().Set("Retry-After", "5")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return false
	}

	if !app.Limiters.SignupGlobal.Allow(1) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Str("method", r.Method).Str("path", r.URL.Path).Msg("signup rate limited globally")
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "server busy", http.StatusTooManyRequests)
		return false
	}

	return true
}

func AllowReadByIP(w http.ResponseWriter, app *app.App, ip string) bool {
	if !app.Limiters.ReadIP.Allow(ip) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Str("ip", ip).Msg("read rate limited by ip")
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return false
	}

	if !app.Limiters.ReadGlobal.Allow(1) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Msg("read rate limited globally")
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "server busy", http.StatusTooManyRequests)
		return false
	}

	return true
}

func AllowReadByUser(w http.ResponseWriter, app *app.App, userID uint64) bool {
	if !app.Limiters.ReadUser.Allow(strconv.FormatUint(userID, 10)) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Uint64("user_id", userID).Msg("read rate limited by user")
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return false
	}

	return true
}

func AllowReadByRequester(w http.ResponseWriter, r *http.Request, app *app.App) bool {
	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if !ok {
		return true
	}

	return AllowReadByUser(w, app, authenticatedUser.ID)
}

func AllowWriteGlobal(w http.ResponseWriter, app *app.App) bool {
	if !app.Limiters.WriteGlobal.Allow(1) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Msg("write rate limited globally")
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "server busy", http.StatusTooManyRequests)
		return false
	}

	return true
}

func AllowWriteByUser(w http.ResponseWriter, app *app.App, userID uint64) bool {
	if !app.Limiters.WriteUser.Allow(strconv.FormatUint(userID, 10)) {
		if app != nil && app.Logger != nil {
			app.Logger.Warn().Uint64("user_id", userID).Msg("write rate limited by user")
		}
		w.Header().Set("Retry-After", "1")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return false
	}

	return true
}

func AllowWriteByRequesterOrUser(w http.ResponseWriter, r *http.Request, app *app.App, fallbackUserID uint64) bool {
	authenticatedUser, ok := AuthenticatedUserForRequest(r)
	if ok {
		return AllowWriteByUser(w, app, authenticatedUser.ID)
	}

	return AllowWriteByUser(w, app, fallbackUserID)
}
