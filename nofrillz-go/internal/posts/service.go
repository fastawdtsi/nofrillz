package posts

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrInvalidAuthorID    = errors.New("invalid author id")
	ErrInvalidBody        = errors.New("invalid post body")
	ErrInvalidSource      = errors.New("invalid post source")
	ErrMissingIDGenerator = errors.New("missing id generator")
)

type idGenerator interface {
	MustNext() uint64
}

type CreateExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type CreatePostInput struct {
	AuthorID uint64
	Body     string
	Source   string
}

type postRepository interface {
	Create(ctx context.Context, post *Post) error
	CreateWithExecutor(ctx context.Context, executor CreateExecutor, post *Post) error
	GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*Post, error)
	ListByUser(ctx context.Context, userID uint64, requesterUserID uint64, beforeID uint64, limit int) ([]*Post, error)
}

type Service struct {
	repository  postRepository
	idGenerator idGenerator
}

func NewService(repository postRepository, idGenerator idGenerator) *Service {
	return &Service{
		repository:  repository,
		idGenerator: idGenerator,
	}
}

func (s *Service) Create(ctx context.Context, post *Post) error {
	return s.repository.Create(ctx, post)
}

func (s *Service) CreatePost(ctx context.Context, input CreatePostInput) (*Post, error) {
	return s.CreatePostWithExecutor(ctx, nil, input)
}

func (s *Service) CreatePostWithExecutor(ctx context.Context, executor CreateExecutor, input CreatePostInput) (*Post, error) {
	if s.idGenerator == nil {
		return nil, ErrMissingIDGenerator
	}

	if input.AuthorID == 0 {
		return nil, ErrInvalidAuthorID
	}
	if strings.TrimSpace(input.Body) == "" || len(input.Body) > MaxBodyBytes {
		return nil, ErrInvalidBody
	}

	source := input.Source
	if source == "" {
		source = SourceHuman
	}
	switch source {
	case SourceHuman, SourceAI, SourceSystem:
	default:
		return nil, ErrInvalidSource
	}

	post := &Post{
		ID:     s.idGenerator.MustNext(),
		Body:   input.Body,
		Source: source,
		User: PostUser{
			UserID: input.AuthorID,
		},
	}

	var err error
	if executor == nil {
		err = s.repository.Create(ctx, post)
	} else {
		err = s.repository.CreateWithExecutor(ctx, executor, post)
	}
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (s *Service) GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*Post, error) {
	return s.repository.GetByID(ctx, ID, requesterUserID)
}

func (s *Service) ListByUser(ctx context.Context, userID uint64, requesterUserID uint64, beforeID uint64, limit int) ([]*Post, error) {
	return s.repository.ListByUser(ctx, userID, requesterUserID, beforeID, limit)
}
