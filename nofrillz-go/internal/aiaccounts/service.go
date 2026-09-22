package aiaccounts

import (
	"context"
	"time"
)

type repository interface {
	FinishClaimWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount, now, next time.Time, success bool, message string) error
	UpdateWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount) error
	Create(ctx context.Context, account *AIAccount) error
	CreateWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount) error
	GetByID(ctx context.Context, id uint64) (*AIAccount, error)
	ListPostGenerationsByAccountID(ctx context.Context, accountID uint64, limit int) ([]*PostGeneration, error)
	CreatePostGeneration(ctx context.Context, generation *PostGeneration) error
	ClaimDueAIAccounts(ctx context.Context, now time.Time, limit int, staleAfter time.Duration) ([]*AIAccount, error)
	CreatePostGenerationWithExecutor(ctx context.Context, executor UpdateExecutor, generation *PostGeneration) error
	MarkGenerationSuccessWithExecutor(ctx context.Context, executor UpdateExecutor, accountID uint64, lastGeneratedAt time.Time, nextGenerateAt time.Time) error
	MarkGenerationFailureWithExecutor(ctx context.Context, executor UpdateExecutor, accountID uint64, nextGenerateAt time.Time, generationError string) error
}

type Service struct {
	repository repository
}

func NewService(repository repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, account *AIAccount) error {
	return s.repository.Create(ctx, account)
}

func (s *Service) CreateWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount) error {
	return s.repository.CreateWithExecutor(ctx, executor, account)
}

func (s *Service) GetByID(ctx context.Context, id uint64) (*AIAccount, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) ListPostGenerationsByAccountID(ctx context.Context, accountID uint64, limit int) ([]*PostGeneration, error) {
	return s.repository.ListPostGenerationsByAccountID(ctx, accountID, limit)
}

func (s *Service) CreatePostGeneration(ctx context.Context, generation *PostGeneration) error {
	return s.repository.CreatePostGeneration(ctx, generation)
}

func (s *Service) ClaimDueAIAccounts(ctx context.Context, now time.Time, limit int, staleAfter time.Duration) ([]*AIAccount, error) {
	return s.repository.ClaimDueAIAccounts(ctx, now, limit, staleAfter)
}

func (s *Service) CreatePostGenerationWithExecutor(ctx context.Context, executor UpdateExecutor, generation *PostGeneration) error {
	return s.repository.CreatePostGenerationWithExecutor(ctx, executor, generation)
}

func (s *Service) MarkGenerationSuccessWithExecutor(ctx context.Context, executor UpdateExecutor, accountID uint64, lastGeneratedAt time.Time, nextGenerateAt time.Time) error {
	return s.repository.MarkGenerationSuccessWithExecutor(ctx, executor, accountID, lastGeneratedAt, nextGenerateAt)
}

func (s *Service) MarkGenerationFailureWithExecutor(ctx context.Context, executor UpdateExecutor, accountID uint64, nextGenerateAt time.Time, generationError string) error {
	return s.repository.MarkGenerationFailureWithExecutor(ctx, executor, accountID, nextGenerateAt, generationError)
}
