package bookmarks

import (
	"context"
	"errors"
	"testing"
	"time"

	"nofrillz/internal/posts"
)

type bookmarksRepoStub struct {
	bookmarkCreated   bool
	bookmarkErr       error
	unbookmarkRemoved bool
	unbookmarkErr     error
	isBookmarked      bool
	isBookmarkedErr   error
	listPosts         []*posts.Post
	listErr           error

	bookmarkPostID     uint64
	bookmarkUserID     uint64
	unbookmarkPostID   uint64
	unbookmarkUserID   uint64
	isBookmarkedPostID uint64
	isBookmarkedUserID uint64
	listUserID         uint64
	listCursor         *Cursor
	listLimit          int
}

func (s *bookmarksRepoStub) BookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	s.bookmarkPostID = postID
	s.bookmarkUserID = userID
	return s.bookmarkCreated, s.bookmarkErr
}

func (s *bookmarksRepoStub) UnbookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	s.unbookmarkPostID = postID
	s.unbookmarkUserID = userID
	return s.unbookmarkRemoved, s.unbookmarkErr
}

func (s *bookmarksRepoStub) IsPostBookmarked(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	s.isBookmarkedPostID = postID
	s.isBookmarkedUserID = userID
	return s.isBookmarked, s.isBookmarkedErr
}

func (s *bookmarksRepoStub) ListBookmarkedPosts(ctx context.Context, userID uint64, cursor *Cursor, limit int) ([]*posts.Post, error) {
	s.listUserID = userID
	s.listCursor = cursor
	s.listLimit = limit
	return s.listPosts, s.listErr
}

type bookmarksPostsStub struct {
	post *posts.Post
	err  error

	postID      uint64
	requesterID uint64
}

func (s *bookmarksPostsStub) GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*posts.Post, error) {
	s.postID = ID
	s.requesterID = requesterUserID
	return s.post, s.err
}

func TestServiceBookmarkVisiblePost(t *testing.T) {
	repo := &bookmarksRepoStub{bookmarkCreated: true}
	postReader := &bookmarksPostsStub{post: &posts.Post{ID: 44}}
	service := NewService(repo, postReader)

	created, err := service.Bookmark(context.Background(), 44, 99)
	if err != nil {
		t.Fatalf("Bookmark: %v", err)
	}
	if !created {
		t.Fatalf("expected created bookmark")
	}
	if postReader.postID != 44 || postReader.requesterID != 99 {
		t.Fatalf("expected post visibility lookup to receive requester and post ids")
	}
	if repo.bookmarkPostID != 44 || repo.bookmarkUserID != 99 {
		t.Fatalf("expected repository args to be forwarded")
	}
}

func TestServiceBookmarkInaccessiblePost(t *testing.T) {
	service := NewService(&bookmarksRepoStub{}, &bookmarksPostsStub{})

	_, err := service.Bookmark(context.Background(), 44, 99)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestServiceBookmarkDuplicate(t *testing.T) {
	repo := &bookmarksRepoStub{bookmarkCreated: false}
	service := NewService(repo, &bookmarksPostsStub{post: &posts.Post{ID: 44}})

	created, err := service.Bookmark(context.Background(), 44, 99)
	if err != nil {
		t.Fatalf("Bookmark: %v", err)
	}
	if created {
		t.Fatalf("expected duplicate bookmark to return created=false")
	}
}

func TestServiceUnbookmark(t *testing.T) {
	repo := &bookmarksRepoStub{unbookmarkRemoved: true}
	service := NewService(repo, &bookmarksPostsStub{post: &posts.Post{ID: 44}})

	removed, err := service.Unbookmark(context.Background(), 44, 99)
	if err != nil {
		t.Fatalf("Unbookmark: %v", err)
	}
	if !removed {
		t.Fatalf("expected bookmark removal to be reported")
	}
	if repo.unbookmarkPostID != 44 || repo.unbookmarkUserID != 99 {
		t.Fatalf("expected unbookmark args to be forwarded")
	}
}

func TestServiceIsPostBookmarked(t *testing.T) {
	repo := &bookmarksRepoStub{isBookmarked: true}
	service := NewService(repo, &bookmarksPostsStub{post: &posts.Post{ID: 44}})

	bookmarked, err := service.IsPostBookmarked(context.Background(), 44, 99)
	if err != nil {
		t.Fatalf("IsPostBookmarked: %v", err)
	}
	if !bookmarked {
		t.Fatalf("expected bookmarked=true")
	}
}

func TestServiceListByUserPagination(t *testing.T) {
	now := time.Date(2026, 7, 11, 18, 0, 0, 0, time.UTC)
	second := now.Add(-1 * time.Minute)
	repo := &bookmarksRepoStub{
		listPosts: []*posts.Post{
			{ID: 90, BookmarkedAt: &now},
			{ID: 80, BookmarkedAt: &second},
			{ID: 70, BookmarkedAt: &second},
		},
	}
	service := NewService(repo, &bookmarksPostsStub{})

	results, nextCursor, err := service.ListByUser(context.Background(), 55, nil, 2)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if repo.listLimit != 3 {
		t.Fatalf("expected repository to receive limit+1, got %d", repo.listLimit)
	}
	if nextCursor == nil {
		t.Fatalf("expected next cursor")
	}
	if !nextCursor.Created.Equal(second) || nextCursor.PostID != 80 {
		t.Fatalf("unexpected next cursor: %+v", *nextCursor)
	}
}
