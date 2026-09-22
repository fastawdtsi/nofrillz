package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"nofrillz/internal/app"
	"nofrillz/internal/bookmarks"
	"nofrillz/internal/limiters"
	"nofrillz/internal/posts"
)

type apiBookmarksRepoStub struct {
	bookmarkCreated   bool
	unbookmarkRemoved bool
	isBookmarked      bool
	listPosts         []*posts.Post

	bookmarkErr     error
	unbookmarkErr   error
	isBookmarkedErr error
	listErr         error
}

func (s *apiBookmarksRepoStub) BookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	return s.bookmarkCreated, s.bookmarkErr
}

func (s *apiBookmarksRepoStub) UnbookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	return s.unbookmarkRemoved, s.unbookmarkErr
}

func (s *apiBookmarksRepoStub) IsPostBookmarked(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	return s.isBookmarked, s.isBookmarkedErr
}

func (s *apiBookmarksRepoStub) ListBookmarkedPosts(ctx context.Context, userID uint64, cursor *bookmarks.Cursor, limit int) ([]*posts.Post, error) {
	return s.listPosts, s.listErr
}

type apiBookmarksPostsStub struct {
	post *posts.Post
	err  error
}

func (s *apiBookmarksPostsStub) GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*posts.Post, error) {
	return s.post, s.err
}

func TestBookmarksHandlerBookmarkAuthRequired(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{}, &apiBookmarksPostsStub{}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/posts/55/bookmark", nil)
	req.SetPathValue("post_id", "55")
	rr := httptest.NewRecorder()

	handler.Bookmark(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 response, got %d", rr.Code)
	}
}

func TestBookmarksHandlerBookmarkReturnsState(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{bookmarkCreated: false}, &apiBookmarksPostsStub{post: &posts.Post{ID: 55}}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/posts/55/bookmark", nil)
	req.SetPathValue("post_id", "55")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Bookmark(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := body["bookmarked"]; got != true {
		t.Fatalf("expected bookmarked true, got %#v", got)
	}
}

func TestBookmarksHandlerBookmarkInvalidPostID(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{}, &apiBookmarksPostsStub{post: &posts.Post{ID: 55}}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/posts/not-a-number/bookmark", nil)
	req.SetPathValue("post_id", "not-a-number")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Bookmark(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", rr.Code)
	}
}

func TestBookmarksHandlerBookmarkPostNotFound(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{}, &apiBookmarksPostsStub{}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/posts/55/bookmark", nil)
	req.SetPathValue("post_id", "55")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Bookmark(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 response, got %d", rr.Code)
	}
}

func TestBookmarksHandlerUnbookmarkReturnsState(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{unbookmarkRemoved: false}, &apiBookmarksPostsStub{post: &posts.Post{ID: 55}}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodDelete, "/posts/55/bookmark", nil)
	req.SetPathValue("post_id", "55")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Unbookmark(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := body["bookmarked"]; got != false {
		t.Fatalf("expected bookmarked false, got %#v", got)
	}
}

func TestBookmarksHandlerListReturnsPosts(t *testing.T) {
	logger := zerolog.Nop()
	bookmarkedAt := time.Date(2026, 7, 11, 18, 0, 0, 0, time.UTC)
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{
			listPosts: []*posts.Post{
				{
					ID:           55,
					Body:         "saved",
					Source:       posts.SourceHuman,
					IsBookmarked: true,
					BookmarkedAt: &bookmarkedAt,
					User: posts.PostUser{
						UserID:      44,
						Username:    "alice",
						FirstName:   "Alice",
						LastName:    "Anderson",
						AccountType: "human",
					},
				},
			},
		}, &apiBookmarksPostsStub{}),
		Limiters: limiters.NewLimiters(nil),
		Logger:   &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/bookmarks", nil)
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body struct {
		Posts []map[string]any `json:"posts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Posts) != 1 {
		t.Fatalf("expected 1 bookmarked post, got %d", len(body.Posts))
	}
	if got := body.Posts[0]["is_bookmarked"]; got != true {
		t.Fatalf("expected is_bookmarked true, got %#v", got)
	}
	if _, ok := body.Posts[0]["bookmarked_at"]; !ok {
		t.Fatalf("expected bookmarked_at in response")
	}
}

func TestBookmarksHandlerListInvalidCursor(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{}, &apiBookmarksPostsStub{}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/bookmarks?cursor=eyJmb28iOiJiYXIifQ", nil)
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 response, got %d", rr.Code)
	}
}

func TestBookmarksHandlerListAuthRequired(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewBookmarksHandler(&app.App{
		Bookmarks: bookmarks.NewService(&apiBookmarksRepoStub{}, &apiBookmarksPostsStub{}),
		Limiters:  limiters.NewLimiters(nil),
		Logger:    &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/bookmarks", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 response, got %d", rr.Code)
	}
}
