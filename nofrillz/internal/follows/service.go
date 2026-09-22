package follows

import (
	"context"
)

type followRepository interface {
	Follow(ctx context.Context, followerID uint64, followingID uint64) (bool, error)
	Unfollow(ctx context.Context, followerID uint64, followingID uint64) error
	ListFollowers(ctx context.Context, userID uint64) ([]uint64, error)
	ListFollowersPage(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*Follower, error)
	ListFollowing(ctx context.Context, userID uint64) ([]uint64, error)
}

type Service struct {
	repository followRepository
}

func NewService(repository followRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Follow(ctx context.Context, followerID uint64, followingID uint64) (bool, error) {
	return s.repository.Follow(ctx, followerID, followingID)
}

func (s *Service) Unfollow(ctx context.Context, followerID uint64, followingID uint64) error {
	return s.repository.Unfollow(ctx, followerID, followingID)
}

func (s *Service) ListFollowers(ctx context.Context, userID uint64) ([]uint64, error) {
	return s.repository.ListFollowers(ctx, userID)
}

func (s *Service) ListFollowersPage(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*Follower, *uint64, error) {
	followers, err := s.repository.ListFollowersPage(ctx, userID, cursor, limit+1)
	if err != nil {
		return nil, nil, err
	}

	if len(followers) <= limit {
		return followers, nil, nil
	}

	followers = followers[:limit]
	lastFollower := followers[len(followers)-1]
	nextCursor := lastFollower.FollowerID

	return followers, &nextCursor, nil
}

func (s *Service) ListFollowing(ctx context.Context, userID uint64) ([]uint64, error) {
	return s.repository.ListFollowing(ctx, userID)
}
