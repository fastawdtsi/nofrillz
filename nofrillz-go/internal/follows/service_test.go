package follows

import (
	"context"
	"errors"
	"testing"
)

type followsRepoStub struct {
	followErr            error
	followCreated        bool
	unfollowErr          error
	listFollowers        []uint64
	listFollowersErr     error
	listFollowersPage    []*Follower
	listFollowersPageErr error
	listFollowing        []uint64
	listFollowingErr     error

	followFollowerID        uint64
	followFollowingID       uint64
	unfollowFollowerID      uint64
	unfollowFollowingID     uint64
	listFollowersUserID     uint64
	listFollowersPageUserID uint64
	listFollowersPageCursor *uint64
	listFollowersPageLimit  int
	listFollowingUserID     uint64
}

func (s *followsRepoStub) Follow(ctx context.Context, followerID uint64, followingID uint64) (bool, error) {
	s.followFollowerID = followerID
	s.followFollowingID = followingID
	return s.followCreated, s.followErr
}

func (s *followsRepoStub) Unfollow(ctx context.Context, followerID uint64, followingID uint64) error {
	s.unfollowFollowerID = followerID
	s.unfollowFollowingID = followingID
	return s.unfollowErr
}

func (s *followsRepoStub) ListFollowers(ctx context.Context, userID uint64) ([]uint64, error) {
	s.listFollowersUserID = userID
	return s.listFollowers, s.listFollowersErr
}

func (s *followsRepoStub) ListFollowersPage(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*Follower, error) {
	s.listFollowersPageUserID = userID
	s.listFollowersPageCursor = cursor
	s.listFollowersPageLimit = limit
	return s.listFollowersPage, s.listFollowersPageErr
}

func (s *followsRepoStub) ListFollowing(ctx context.Context, userID uint64) ([]uint64, error) {
	s.listFollowingUserID = userID
	return s.listFollowing, s.listFollowingErr
}

func TestServiceFollow(t *testing.T) {
	expectedErr := errors.New("follow error")
	repo := &followsRepoStub{followErr: expectedErr}
	service := NewService(repo)

	created, err := service.Follow(context.Background(), 10, 11)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected follow error, got %v", err)
	}
	if created {
		t.Fatalf("expected created to be false when repository returns an error")
	}
	if repo.followFollowerID != 10 || repo.followFollowingID != 11 {
		t.Fatalf("expected follow args to be forwarded")
	}
}

func TestServiceFollowReturnsCreated(t *testing.T) {
	repo := &followsRepoStub{followCreated: true}
	service := NewService(repo)

	created, err := service.Follow(context.Background(), 21, 22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatalf("expected created follow to be returned")
	}
}

func TestServiceUnfollow(t *testing.T) {
	repo := &followsRepoStub{}
	service := NewService(repo)

	err := service.Unfollow(context.Background(), 12, 13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.unfollowFollowerID != 12 || repo.unfollowFollowingID != 13 {
		t.Fatalf("expected unfollow args to be forwarded")
	}
}

func TestServiceListFollowers(t *testing.T) {
	expected := []uint64{1, 2, 3}
	repo := &followsRepoStub{listFollowers: expected}
	service := NewService(repo)

	got, err := service.ListFollowers(context.Background(), 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d followers, got %d", len(expected), len(got))
	}
	if repo.listFollowersUserID != 15 {
		t.Fatalf("expected user ID argument to be forwarded")
	}
}

func TestServiceListFollowing(t *testing.T) {
	expected := []uint64{7, 8}
	repo := &followsRepoStub{listFollowing: expected}
	service := NewService(repo)

	got, err := service.ListFollowing(context.Background(), 17)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d following, got %d", len(expected), len(got))
	}
	if repo.listFollowingUserID != 17 {
		t.Fatalf("expected user ID argument to be forwarded")
	}
}

func TestServiceListFollowersPage(t *testing.T) {
	page := []*Follower{
		{FollowerID: 10},
		{FollowerID: 9},
		{FollowerID: 8},
	}
	repo := &followsRepoStub{listFollowersPage: page}
	service := NewService(repo)

	got, next, err := service.ListFollowersPage(context.Background(), 20, nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 followers, got %d", len(got))
	}
	if next == nil || *next != 9 {
		t.Fatalf("expected next cursor to point to last returned follower")
	}
	if repo.listFollowersPageUserID != 20 || repo.listFollowersPageLimit != 3 {
		t.Fatalf("expected paginated list args to be forwarded with limit+1")
	}
}
