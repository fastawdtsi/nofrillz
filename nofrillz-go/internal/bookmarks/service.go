package bookmarks

import (
	"context"
	"errors"

	"nofrillz/internal/posts"
)

var (
	ErrInvalidUserID = errors.New("invalid user id")
	ErrInvalidPostID = errors.New("invalid post id")
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrPostNotFound  = errors.New("post not found")
)

type repository interface {
	BookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error)
	UnbookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error)
	IsPostBookmarked(ctx context.Context, postID uint64, userID uint64) (bool, error)
	ListBookmarkedPosts(ctx context.Context, userID uint64, cursor *Cursor, limit int) ([]*posts.Post, error)
}

type postReader interface {
	GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*posts.Post, error)
}

type Service struct {
	repository repository
	posts      postReader
}

func NewService(repository repository, posts postReader) *Service {
	return &Service{
		repository: repository,
		posts:      posts,
	}
}

func (s *Service) Bookmark(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	if userID == 0 {
		return false, ErrInvalidUserID
	}
	if postID == 0 {
		return false, ErrInvalidPostID
	}

	post, err := s.posts.GetByID(ctx, postID, userID)
	if err != nil {
		return false, err
	}
	if post == nil {
		return false, ErrPostNotFound
	}

	return s.repository.BookmarkPost(ctx, postID, userID)
}

func (s *Service) Unbookmark(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	if userID == 0 {
		return false, ErrInvalidUserID
	}
	if postID == 0 {
		return false, ErrInvalidPostID
	}

	post, err := s.posts.GetByID(ctx, postID, userID)
	if err != nil {
		return false, err
	}
	if post == nil {
		return false, ErrPostNotFound
	}

	return s.repository.UnbookmarkPost(ctx, postID, userID)
}

func (s *Service) IsPostBookmarked(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	if userID == 0 {
		return false, ErrInvalidUserID
	}
	if postID == 0 {
		return false, ErrInvalidPostID
	}

	post, err := s.posts.GetByID(ctx, postID, userID)
	if err != nil {
		return false, err
	}
	if post == nil {
		return false, ErrPostNotFound
	}

	return s.repository.IsPostBookmarked(ctx, postID, userID)
}

func (s *Service) ListByUser(ctx context.Context, userID uint64, cursor *Cursor, limit int) ([]*posts.Post, *Cursor, error) {
	if userID == 0 {
		return nil, nil, ErrInvalidUserID
	}
	if limit <= 0 {
		return nil, nil, ErrInvalidLimit
	}

	results, err := s.repository.ListBookmarkedPosts(ctx, userID, cursor, limit+1)
	if err != nil {
		return nil, nil, err
	}

	if len(results) <= limit {
		return results, nil, nil
	}

	results = results[:limit]
	last := results[len(results)-1]
	if last.BookmarkedAt == nil {
		return results, nil, nil
	}

	nextCursor := Cursor{
		Created: *last.BookmarkedAt,
		PostID:  last.ID,
	}

	return results, &nextCursor, nil
}
