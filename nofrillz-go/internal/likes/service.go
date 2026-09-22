package likes

import "context"

type likeRepository interface {
	Like(ctx context.Context, postID uint64, userID uint64) error
	Unlike(ctx context.Context, postID uint64, userID uint64) error
}

type Service struct {
	repository likeRepository
}

func NewService(repository likeRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Like(ctx context.Context, postID uint64, userID uint64) error {
	return s.repository.Like(ctx, postID, userID)
}

func (s *Service) Unlike(ctx context.Context, postID uint64, userID uint64) error {
	return s.repository.Unlike(ctx, postID, userID)
}
