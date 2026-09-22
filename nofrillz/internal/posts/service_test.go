package posts

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type postsIDGeneratorStub struct {
	next uint64
}

func (s *postsIDGeneratorStub) MustNext() uint64 {
	return s.next
}

type postsExecutorStub struct{}

func (s *postsExecutorStub) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

type postsRepoStub struct {
	createErr     error
	getByID       *Post
	getByIDErr    error
	listByUser    []*Post
	listByUserErr error

	created            *Post
	idArg              uint64
	requesterID        uint64
	userIDArg          uint64
	listRequesterIDArg uint64
	beforeIDArg        uint64
	limitArg           int
}

func (s *postsRepoStub) Create(ctx context.Context, post *Post) error {
	s.created = post
	return s.createErr
}

func (s *postsRepoStub) CreateWithExecutor(ctx context.Context, executor CreateExecutor, post *Post) error {
	s.created = post
	return s.createErr
}

func (s *postsRepoStub) GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*Post, error) {
	s.idArg = ID
	s.requesterID = requesterUserID
	return s.getByID, s.getByIDErr
}

func (s *postsRepoStub) ListByUser(ctx context.Context, userID uint64, requesterUserID uint64, beforeID uint64, limit int) ([]*Post, error) {
	s.userIDArg = userID
	s.listRequesterIDArg = requesterUserID
	s.beforeIDArg = beforeID
	s.limitArg = limit
	return s.listByUser, s.listByUserErr
}

func TestServiceCreate(t *testing.T) {
	expectedErr := errors.New("create error")
	repo := &postsRepoStub{createErr: expectedErr}
	service := NewService(repo, &postsIDGeneratorStub{next: 99})
	post := &Post{ID: 10}

	err := service.Create(context.Background(), post)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected create error, got %v", err)
	}
	if repo.created != post {
		t.Fatalf("expected repository to receive post pointer")
	}
}

func TestServiceGetByID(t *testing.T) {
	expected := &Post{ID: 12}
	repo := &postsRepoStub{getByID: expected}
	service := NewService(repo, &postsIDGeneratorStub{next: 99})

	got, err := service.GetByID(context.Background(), 12, 88)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected same post pointer from repository")
	}
	if repo.idArg != 12 {
		t.Fatalf("expected ID argument to be forwarded")
	}
	if repo.requesterID != 88 {
		t.Fatalf("expected requester ID argument to be forwarded")
	}
}

func TestServiceListByUser(t *testing.T) {
	expected := []*Post{{ID: 1}, {ID: 2}}
	repo := &postsRepoStub{listByUser: expected}
	service := NewService(repo, &postsIDGeneratorStub{next: 99})

	got, err := service.ListByUser(context.Background(), 44, 88, 22, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d posts, got %d", len(expected), len(got))
	}
	if repo.userIDArg != 44 || repo.listRequesterIDArg != 88 || repo.beforeIDArg != 22 || repo.limitArg != 50 {
		t.Fatalf("expected list arguments to be forwarded")
	}
}

func TestServiceUnicodeRoundTrip(t *testing.T) {
	unicodeBody := "hello 👋 こんにちは مرحبا 你好"
	unicodeUsername := "ユーザー🌏"

	repo := &postsRepoStub{
		getByID:    &Post{ID: 77, Body: unicodeBody, User: PostUser{UserID: 99, Username: unicodeUsername}},
		listByUser: []*Post{{ID: 78, Body: unicodeBody, User: PostUser{UserID: 99, Username: unicodeUsername}}},
	}
	service := NewService(repo, &postsIDGeneratorStub{next: 99})

	createPost := &Post{ID: 76, Body: unicodeBody, User: PostUser{UserID: 99, Username: unicodeUsername}}
	if err := service.Create(context.Background(), createPost); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.created == nil || repo.created.Body != unicodeBody || repo.created.User.Username != unicodeUsername {
		t.Fatalf("unicode data changed in create path")
	}

	gotByID, err := service.GetByID(context.Background(), 77, 99)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if gotByID == nil || gotByID.Body != unicodeBody || gotByID.User.Username != unicodeUsername {
		t.Fatalf("unicode data changed in get-by-id path")
	}

	list, err := service.ListByUser(context.Background(), 99, 77, 0, 10)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 1 || list[0].Body != unicodeBody || list[0].User.Username != unicodeUsername {
		t.Fatalf("unicode data changed in list-by-user path")
	}
}

func TestServiceCreatePost(t *testing.T) {
	repo := &postsRepoStub{}
	service := NewService(repo, &postsIDGeneratorStub{next: 123})

	post, err := service.CreatePost(context.Background(), CreatePostInput{
		AuthorID: 44,
		Body:     "hello world",
		Source:   SourceAI,
	})
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if post.ID != 123 {
		t.Fatalf("expected generated ID 123, got %d", post.ID)
	}
	if post.Source != SourceAI {
		t.Fatalf("expected source %q, got %q", SourceAI, post.Source)
	}
	if repo.created == nil || repo.created.Source != SourceAI || repo.created.User.UserID != 44 {
		t.Fatalf("expected repository create to receive source and author")
	}
}

func TestServiceCreatePostDefaultsSource(t *testing.T) {
	repo := &postsRepoStub{}
	service := NewService(repo, &postsIDGeneratorStub{next: 123})

	post, err := service.CreatePost(context.Background(), CreatePostInput{
		AuthorID: 44,
		Body:     "hello world",
	})
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if post.Source != SourceHuman {
		t.Fatalf("expected default source %q, got %q", SourceHuman, post.Source)
	}
}

func TestServiceCreatePostInvalidBody(t *testing.T) {
	repo := &postsRepoStub{}
	service := NewService(repo, &postsIDGeneratorStub{next: 123})

	_, err := service.CreatePost(context.Background(), CreatePostInput{
		AuthorID: 44,
		Body:     "   ",
		Source:   SourceHuman,
	})
	if !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("expected invalid body error, got %v", err)
	}
}

func TestServiceCreatePostWithExecutor(t *testing.T) {
	repo := &postsRepoStub{}
	service := NewService(repo, &postsIDGeneratorStub{next: 999})

	post, err := service.CreatePostWithExecutor(context.Background(), &postsExecutorStub{}, CreatePostInput{
		AuthorID: 55,
		Body:     "executor path",
		Source:   SourceSystem,
	})
	if err != nil {
		t.Fatalf("CreatePostWithExecutor: %v", err)
	}
	if post.ID != 999 || post.Source != SourceSystem {
		t.Fatalf("expected executor path to preserve ID and source")
	}
}
