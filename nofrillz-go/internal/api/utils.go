package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"nofrillz/internal/app"
	"nofrillz/internal/bookmarks"
	"nofrillz/internal/posts"
)

func ClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		ip := strings.Split(forwardedFor, ",")[0]
		return strings.TrimSpace(ip)
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

func DecodeRequestBody[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (T, error) {
	var zero T

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var v T
	if err := dec.Decode(&v); err != nil {
		return zero, err
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return zero, errors.New("invalid json")
	}

	return v, nil
}

func Respond(w http.ResponseWriter, status int, payload any, logger *zerolog.Logger) {
	if payload == nil {
		w.WriteHeader(status)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		if logger != nil {
			logger.Error().Err(err).Msg("error encoding response body")
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}

type idCursorPayload struct {
	ID string `json:"id"`
}

type bookmarkCursorPayload struct {
	Created string `json:"created"`
	PostID  string `json:"post_id"`
}

func EncodeIDCursor(cursor *uint64) (string, error) {
	if cursor == nil {
		return "", nil
	}

	payload := idCursorPayload{
		ID: strconv.FormatUint(*cursor, 10),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal cursor payload: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(body), nil
}

func DecodeIDCursor(raw string) (*uint64, error) {
	if raw == "" {
		return nil, nil
	}

	body, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	payload := idCursorPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal cursor payload: %w", err)
	}
	if payload.ID == "" {
		return nil, fmt.Errorf("invalid cursor payload")
	}

	id, err := strconv.ParseUint(payload.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse cursor id: %w", err)
	}

	return &id, nil
}

func EncodeBookmarkCursor(cursor *bookmarks.Cursor) (string, error) {
	if cursor == nil {
		return "", nil
	}

	payload := bookmarkCursorPayload{
		Created: cursor.Created.UTC().Format(time.RFC3339Nano),
		PostID:  strconv.FormatUint(cursor.PostID, 10),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal bookmark cursor payload: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(body), nil
}

func DecodeBookmarkCursor(raw string) (*bookmarks.Cursor, error) {
	if raw == "" {
		return nil, nil
	}

	body, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode bookmark cursor: %w", err)
	}

	payload := bookmarkCursorPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal bookmark cursor payload: %w", err)
	}
	if payload.Created == "" || payload.PostID == "" {
		return nil, fmt.Errorf("invalid bookmark cursor payload")
	}

	created, err := time.Parse(time.RFC3339Nano, payload.Created)
	if err != nil {
		return nil, fmt.Errorf("parse bookmark cursor created: %w", err)
	}

	postID, err := strconv.ParseUint(payload.PostID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse bookmark cursor post id: %w", err)
	}

	return &bookmarks.Cursor{
		Created: created,
		PostID:  postID,
	}, nil
}

func AvatarURL(app *app.App, userID uint64) string {
	if app == nil || app.Config == nil {
		return ""
	}

	base := strings.TrimRight(app.Config.BaseURLsConfig().Avatar, "/")
	if base == "" {
		return ""
	}

	return base + "/users/" + strconv.FormatUint(userID, 10) + "/avatar.jpeg"
}

func PostURLPath(postID uint64) string {
	if postID == 0 {
		return ""
	}

	return "/posts/" + strconv.FormatUint(postID, 10)
}

func SetPostResponseFields(app *app.App, post *posts.Post) {
	if post == nil {
		return
	}

	post.User.AvatarURL = AvatarURL(app, post.User.UserID)
	post.URL = PostURLPath(post.ID)
}

func SetPostResponseFieldsMany(app *app.App, postsList []*posts.Post) {
	for _, post := range postsList {
		SetPostResponseFields(app, post)
	}
}
