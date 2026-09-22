package users

import (
	"context"
	"database/sql"
)

type CreateExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type userRepository interface {
	Create(ctx context.Context, user *User) error
	CreateWithExecutor(ctx context.Context, executor CreateExecutor, user *User) error
	GetByID(ctx context.Context, ID uint64) (*User, error)
	GetByIDForRequester(ctx context.Context, ID uint64, requesterUserID uint64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	SearchByUsernamePrefix(ctx context.Context, prefix string, limit int) ([]*User, error)
	SearchByUsernamePrefixForRequester(ctx context.Context, prefix string, requesterUserID uint64, limit int) ([]*User, error)
	ListFollowersPageForRequester(ctx context.Context, userID uint64, requesterUserID uint64, cursor *uint64, limit int) ([]*User, error)
}

type Service struct {
	repository userRepository
}

func NewService(repository userRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, user *User) error {
	return s.repository.Create(ctx, user)
}

func (s *Service) CreateWithExecutor(ctx context.Context, executor CreateExecutor, user *User) error {
	return s.repository.CreateWithExecutor(ctx, executor, user)
}

func (s *Service) GetByID(ctx context.Context, ID uint64) (*User, error) {
	return s.repository.GetByID(ctx, ID)
}

func (s *Service) GetByIDForRequester(ctx context.Context, ID uint64, requesterUserID uint64) (*User, error) {
	return s.repository.GetByIDForRequester(ctx, ID, requesterUserID)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repository.GetByEmail(ctx, email)
}

func (s *Service) SearchByUsernamePrefix(ctx context.Context, prefix string, limit int) ([]*User, error) {
	return s.repository.SearchByUsernamePrefix(ctx, prefix, limit)
}

func (s *Service) SearchByUsernamePrefixForRequester(ctx context.Context, prefix string, requesterUserID uint64, limit int) ([]*User, error) {
	return s.repository.SearchByUsernamePrefixForRequester(ctx, prefix, requesterUserID, limit)
}

func (s *Service) ListFollowersPageForRequester(ctx context.Context, userID uint64, requesterUserID uint64, cursor *uint64, limit int) ([]*User, *uint64, error) {
	followers, err := s.repository.ListFollowersPageForRequester(ctx, userID, requesterUserID, cursor, limit+1)
	if err != nil {
		return nil, nil, err
	}

	if len(followers) <= limit {
		return followers, nil, nil
	}

	followers = followers[:limit]
	lastFollower := followers[len(followers)-1]
	nextCursor := lastFollower.ID

	return followers, &nextCursor, nil
}
