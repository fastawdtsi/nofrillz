package likes

import (
	"context"
	"errors"
	"testing"
)

type likesRepoStub struct {
	likeErr   error
	unlikeErr error

	likedPostID   uint64
	likedUserID   uint64
	unlikedPostID uint64
	unlikedUserID uint64
}

func (s *likesRepoStub) Like(ctx context.Context, postID uint64, userID uint64) error {
	s.likedPostID = postID
	s.likedUserID = userID
	return s.likeErr
}

func (s *likesRepoStub) Unlike(ctx context.Context, postID uint64, userID uint64) error {
	s.unlikedPostID = postID
	s.unlikedUserID = userID
	return s.unlikeErr
}

func TestServiceLike(t *testing.T) {
	repo := &likesRepoStub{}
	service := NewService(repo)

	if err := service.Like(context.Background(), 123, 999); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.likedPostID != 123 || repo.likedUserID != 999 {
		t.Fatalf("expected like args to be forwarded")
	}
}

func TestServiceLikeError(t *testing.T) {
	expectedErr := errors.New("like error")
	repo := &likesRepoStub{likeErr: expectedErr}
	service := NewService(repo)

	err := service.Like(context.Background(), 1, 2)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected like error, got %v", err)
	}
}

func TestServiceUnlike(t *testing.T) {
	repo := &likesRepoStub{}
	service := NewService(repo)

	if err := service.Unlike(context.Background(), 44, 66); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.unlikedPostID != 44 || repo.unlikedUserID != 66 {
		t.Fatalf("expected unlike args to be forwarded")
	}
}

func TestServiceUnlikeError(t *testing.T) {
	expectedErr := errors.New("unlike error")
	repo := &likesRepoStub{unlikeErr: expectedErr}
	service := NewService(repo)

	err := service.Unlike(context.Background(), 3, 4)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected unlike error, got %v", err)
	}
}
