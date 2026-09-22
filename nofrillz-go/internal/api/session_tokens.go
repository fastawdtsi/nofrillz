package api

import (
	"context"
	"time"

	"nofrillz/internal/app"
	"nofrillz/internal/sessions"
)

func issueTokensForUser(app *app.App, ctx context.Context, userID uint64) (string, string, uint64, error) {
	sessionID := app.IDGenerator.MustNext()

	err := app.Sessions.Create(ctx, sessionID, userID)
	if err != nil {
		return "", "", 0, err
	}

	accessToken, refreshToken, err := issueTokensForSession(app, ctx, userID, sessionID)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, sessionID, nil
}

func issueTokensForSession(app *app.App, ctx context.Context, userID uint64, sessionID uint64) (string, string, error) {
	accessToken, err := sessions.GenerateAccessToken(app.Config.SessionConfig().JWTSecret, userID, sessionID, accessTokenTTL(app))
	if err != nil {
		return "", "", err
	}

	refreshToken, err := sessions.GenerateSessionToken()
	if err != nil {
		return "", "", err
	}
	refreshTokenHash := sessions.HashToken(refreshToken)

	refreshExpiresAt := time.Now().UTC().Add(refreshTokenTTL(app))
	err = app.Sessions.CreateRefresh(ctx, refreshTokenHash, sessionID, refreshExpiresAt)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func accessTokenTTL(app *app.App) time.Duration {
	sessionConfig := app.Config.SessionConfig()
	accessTTL := time.Duration(sessionConfig.AccessTTLSeconds) * time.Second
	if accessTTL <= 0 {
		return 15 * time.Minute
	}

	return accessTTL
}

func refreshTokenTTL(app *app.App) time.Duration {
	sessionConfig := app.Config.SessionConfig()
	refreshTTL := time.Duration(sessionConfig.RefreshTTLSeconds) * time.Second
	if refreshTTL <= 0 {
		return 365 * 24 * time.Hour
	}

	return refreshTTL
}
