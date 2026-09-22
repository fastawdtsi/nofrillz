package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"nofrillz/internal/app"
	"nofrillz/internal/feed"
	"nofrillz/internal/limiters"
	"nofrillz/internal/posts"
)

type apiPostsIDGeneratorStub struct {
	next uint64
}

func (s *apiPostsIDGeneratorStub) MustNext() uint64 {
	return s.next
}

type apiPostsRepoStub struct {
	created *posts.Post
	getByID *posts.Post
	list    []*posts.Post
}

func (s *apiPostsRepoStub) Create(ctx context.Context, post *posts.Post) error {
	s.created = post
	return nil
}

func (s *apiPostsRepoStub) CreateWithExecutor(ctx context.Context, executor posts.CreateExecutor, post *posts.Post) error {
	s.created = post
	return nil
}

func (s *apiPostsRepoStub) GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*posts.Post, error) {
	if s.getByID != nil {
		return s.getByID, nil
	}

	if s.created == nil {
		return nil, nil
	}
	return &posts.Post{
		ID:     s.created.ID,
		Body:   s.created.Body,
		Source: s.created.Source,
		User: posts.PostUser{
			UserID: s.created.User.UserID,
		},
	}, nil
}

func (s *apiPostsRepoStub) ListByUser(ctx context.Context, userID uint64, requesterUserID uint64, beforeID uint64, limit int) ([]*posts.Post, error) {
	return s.list, nil
}

type apiFeedRepoStub struct {
	posts []*feed.Post
}

func (s *apiFeedRepoStub) List(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*feed.Post, error) {
	return s.posts, nil
}

func TestPostsHandlerCreateUsesHumanSource(t *testing.T) {
	repo := &apiPostsRepoStub{}
	service := posts.NewService(repo, &apiPostsIDGeneratorStub{next: 123})
	logger := zerolog.Nop()
	handler := NewPostsHandler(&app.App{
		Posts:    service,
		Limiters: limiters.NewLimiters(nil),
		Logger:   &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"body":"hello"}`))
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}
	if repo.created == nil {
		t.Fatalf("expected post to be created")
	}
	if repo.created.Source != posts.SourceHuman {
		t.Fatalf("expected human post source, got %q", repo.created.Source)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := body["url"]; got != "/posts/123" {
		t.Fatalf("expected url %q in response, got %#v", "/posts/123", got)
	}
	if got := body["is_bookmarked"]; got != false {
		t.Fatalf("expected is_bookmarked false in response, got %#v", got)
	}
}

func TestPostsHandlerShowJSONIncludesSource(t *testing.T) {
	repo := &apiPostsRepoStub{
		getByID: &posts.Post{
			ID:           123,
			Body:         "hello",
			Source:       posts.SourceAI,
			IsBookmarked: true,
			User: posts.PostUser{
				UserID: 44,
			},
		},
	}
	service := posts.NewService(repo, &apiPostsIDGeneratorStub{next: 123})
	logger := zerolog.Nop()
	handler := NewPostsHandler(&app.App{
		Posts:    service,
		Limiters: limiters.NewLimiters(nil),
		Logger:   &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/123", nil)
	req.SetPathValue("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := body["source"]; got != posts.SourceAI {
		t.Fatalf("expected source %q in response, got %#v", posts.SourceAI, got)
	}
	if got := body["url"]; got != "/posts/123" {
		t.Fatalf("expected url %q in response, got %#v", "/posts/123", got)
	}
	if got := body["is_bookmarked"]; got != true {
		t.Fatalf("expected is_bookmarked true in response, got %#v", got)
	}
}

func TestUsersHandlerPostsJSONIncludesSource(t *testing.T) {
	repo := &apiPostsRepoStub{
		list: []*posts.Post{
			{
				ID:           321,
				Body:         "from ai",
				Source:       posts.SourceAI,
				Liked:        true,
				IsBookmarked: true,
				User: posts.PostUser{
					UserID: 55,
				},
			},
		},
	}
	service := posts.NewService(repo, &apiPostsIDGeneratorStub{next: 123})
	logger := zerolog.Nop()
	handler := NewUsersHandler(&app.App{
		Posts:    service,
		Limiters: limiters.NewLimiters(nil),
		Logger:   &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/users/55/posts", nil)
	req.SetPathValue("id", "55")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.Posts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("expected 1 post in response, got %d", len(body))
	}
	if got := body[0]["source"]; got != posts.SourceAI {
		t.Fatalf("expected source %q in response, got %#v", posts.SourceAI, got)
	}
	if got := body[0]["url"]; got != "/posts/321" {
		t.Fatalf("expected url %q in response, got %#v", "/posts/321", got)
	}
	if got := body[0]["liked"]; got != true {
		t.Fatalf("expected liked true in response, got %#v", got)
	}
	if got := body[0]["is_bookmarked"]; got != true {
		t.Fatalf("expected is_bookmarked true in response, got %#v", got)
	}
}

func TestFeedHandlerListJSONIncludesSource(t *testing.T) {
	feedService := feed.NewService(&apiFeedRepoStub{
		posts: []*feed.Post{
			{
				ID:           777,
				Body:         "feed post",
				Source:       posts.SourceAI,
				IsBookmarked: true,
				UserID:       55,
				Created:      repoTime(),
			},
		},
	})
	logger := zerolog.Nop()
	handler := NewFeedHandler(&app.App{
		Feed:     feedService,
		Limiters: limiters.NewLimiters(nil),
		Logger:   &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
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
		t.Fatalf("expected 1 post in feed response, got %d", len(body.Posts))
	}
	if got := body.Posts[0]["source"]; got != posts.SourceAI {
		t.Fatalf("expected source %q in response, got %#v", posts.SourceAI, got)
	}
	if got := body.Posts[0]["url"]; got != "/posts/777" {
		t.Fatalf("expected url %q in response, got %#v", "/posts/777", got)
	}
	if got := body.Posts[0]["bookmarked"]; got != true {
		t.Fatalf("expected bookmarked true in response, got %#v", got)
	}
	if got := body.Posts[0]["is_bookmarked"]; got != true {
		t.Fatalf("expected is_bookmarked true in response, got %#v", got)
	}
}

func repoTime() time.Time {
	return time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
}

func (s *apiFeedRepoStub) ListDiscover(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*feed.Post, error) {
	return s.List(ctx, requesterUserID, cursor, limit)
}
