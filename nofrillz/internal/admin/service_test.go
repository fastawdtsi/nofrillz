package admin

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/aitools"
	"nofrillz/internal/posts"
	"nofrillz/internal/users"
)

type adminGeneratorStub struct {
	post aigenerator.GeneratedPost
	err  error
}

type adminAIToolsStub struct {
	result aitools.GeneratedPost
	err    error
}

func (s *adminGeneratorStub) GeneratePost(ctx context.Context, account *aiaccounts.AIAccount) (aigenerator.GeneratedPost, error) {
	return s.post, s.err
}

func (s *adminAIToolsStub) GeneratePostContent(ctx context.Context, input aitools.GeneratePostInput) (aitools.GeneratedPost, error) {
	return s.result, s.err
}

type adminIDGeneratorStub struct {
	next []uint64
}

func (s *adminIDGeneratorStub) MustNext() uint64 {
	value := s.next[0]
	s.next = s.next[1:]
	return value
}

type adminTx struct{}

func (t *adminTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (t *adminTx) Commit() error   { return nil }
func (t *adminTx) Rollback() error { return nil }

type adminTxManager struct{}

func (m *adminTxManager) Begin(ctx context.Context) (transaction, error) {
	return &adminTx{}, nil
}

type adminUsersStub struct {
	created *users.User
	got     *users.User
}

func (s *adminUsersStub) CreateWithExecutor(ctx context.Context, executor users.CreateExecutor, user *users.User) error {
	s.created = user
	return nil
}

func (s *adminUsersStub) GetByID(ctx context.Context, id uint64) (*users.User, error) {
	return s.got, nil
}

type adminPostsStub struct {
	input posts.CreatePostInput
	post  *posts.Post
}

func (s *adminPostsStub) CreatePostWithExecutor(ctx context.Context, executor posts.CreateExecutor, input posts.CreatePostInput) (*posts.Post, error) {
	s.input = input
	return s.post, nil
}

type adminSuccessMark struct {
	accountID       uint64
	lastGeneratedAt time.Time
	nextGenerateAt  time.Time
}

type adminAIAccountsStub struct {
	gotAccount         *aiaccounts.AIAccount
	createdAccount     *aiaccounts.AIAccount
	createdGenerations []*aiaccounts.PostGeneration
	successMarks       []adminSuccessMark
	listGenerations    []*aiaccounts.PostGeneration
}

func (s *adminAIAccountsStub) CreateWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, account *aiaccounts.AIAccount) error {
	s.createdAccount = account
	return nil
}

func (s *adminAIAccountsStub) GetByID(ctx context.Context, id uint64) (*aiaccounts.AIAccount, error) {
	return s.gotAccount, nil
}

func (s *adminAIAccountsStub) ListPostGenerationsByAccountID(ctx context.Context, accountID uint64, limit int) ([]*aiaccounts.PostGeneration, error) {
	return s.listGenerations, nil
}

func (s *adminAIAccountsStub) CreatePostGeneration(ctx context.Context, generation *aiaccounts.PostGeneration) error {
	s.createdGenerations = append(s.createdGenerations, generation)
	return nil
}

func (s *adminAIAccountsStub) CreatePostGenerationWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, generation *aiaccounts.PostGeneration) error {
	s.createdGenerations = append(s.createdGenerations, generation)
	return nil
}

func (s *adminAIAccountsStub) MarkGenerationSuccessWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, accountID uint64, lastGeneratedAt time.Time, nextGenerateAt time.Time) error {
	s.successMarks = append(s.successMarks, adminSuccessMark{
		accountID:       accountID,
		lastGeneratedAt: lastGeneratedAt,
		nextGenerateAt:  nextGenerateAt,
	})
	return nil
}

type adminRepositoryStub struct{}

func (s *adminRepositoryStub) Stats(ctx context.Context) (*Stats, error) {
	return &Stats{}, nil
}

func (s *adminRepositoryStub) ListUsers(ctx context.Context, input ListUsersInput) ([]*UserSummary, error) {
	return nil, nil
}

func (s *adminRepositoryStub) GetUserByID(ctx context.Context, userID uint64) (*UserSummary, error) {
	return nil, nil
}

func (s *adminRepositoryStub) BlockUser(ctx context.Context, userID uint64) error {
	return nil
}

func (s *adminRepositoryStub) ListPosts(ctx context.Context, input ListPostsInput) ([]*PostSummary, error) {
	return nil, nil
}

func (s *adminRepositoryStub) DeletePost(ctx context.Context, postID uint64) (bool, error) {
	return false, nil
}

func TestServiceCreateAccountCreatesAIUserAndAccount(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	accounts := &adminAIAccountsStub{}
	usersSvc := &adminUsersStub{}
	service := NewService(
		&adminTxManager{},
		accounts,
		usersSvc,
		&adminPostsStub{},
		&adminAIToolsStub{},
		&adminGeneratorStub{},
		&adminIDGeneratorStub{next: []uint64{101, 202}},
		&adminRepositoryStub{},
	)
	service.now = func() time.Time { return now }

	record, err := service.CreateAccount(context.Background(), CreateAccountInput{
		Email:        "ai@example.com",
		Username:     "nofrillz-ai",
		FirstName:    "NoFrillz",
		LastName:     "AI",
		Enabled:      true,
		Topic:        "minimalism",
		SystemPrompt: "Write briefly.",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if record.User == nil || record.User.AccountType != users.AccountTypeAI {
		t.Fatalf("expected AI user account type")
	}
	if len(record.User.PasswordHash) == 0 || len(record.User.PasswordSalt) == 0 {
		t.Fatalf("expected generated credentials for ai user")
	}
	if record.Account == nil || record.Account.UserID != record.User.ID {
		t.Fatalf("expected ai account to point at created user")
	}
	if record.Account.NextGenerateAt == nil || !record.Account.NextGenerateAt.After(now) {
		t.Fatalf("expected initial posting to be staggered")
	}
	if accounts.createdAccount == nil || usersSvc.created == nil {
		t.Fatalf("expected both user and account to be persisted")
	}
}

func TestServiceCreatePreviewPersistsGeneratedPreview(t *testing.T) {
	account := &aiaccounts.AIAccount{
		ID:           55,
		UserID:       66,
		Topic:        "writing",
		SystemPrompt: "Be concise.",
	}
	accounts := &adminAIAccountsStub{gotAccount: account}
	service := NewService(
		&adminTxManager{},
		accounts,
		&adminUsersStub{},
		&adminPostsStub{},
		&adminAIToolsStub{},
		&adminGeneratorStub{post: aigenerator.GeneratedPost{Body: "Simple tools usually win.", Prompt: "p", Model: "mock"}},
		&adminIDGeneratorStub{next: []uint64{777}},
		&adminRepositoryStub{},
	)

	generation, err := service.CreatePreview(context.Background(), 55)
	if err != nil {
		t.Fatalf("CreatePreview: %v", err)
	}
	if generation.Status != aiaccounts.PostGenerationStatusGenerated {
		t.Fatalf("expected generated status, got %q", generation.Status)
	}
	if len(accounts.createdGenerations) != 1 {
		t.Fatalf("expected one stored generation")
	}
	if accounts.createdGenerations[0].CandidateBody != "Simple tools usually win." {
		t.Fatalf("expected candidate body to be persisted")
	}
}

func TestServiceCreatePostPublishesAIPostAndMarksSuccess(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	account := &aiaccounts.AIAccount{
		ID:             55,
		UserID:         66,
		Topic:          "writing",
		SystemPrompt:   "Be concise.",
		MinPostsPerDay: 1,
		MaxPostsPerDay: 1,
	}
	accounts := &adminAIAccountsStub{gotAccount: account}
	postsSvc := &adminPostsStub{post: &posts.Post{ID: 999, Body: "Simple tools usually win.", Source: posts.SourceAI}}
	service := NewService(
		&adminTxManager{},
		accounts,
		&adminUsersStub{},
		postsSvc,
		&adminAIToolsStub{},
		&adminGeneratorStub{post: aigenerator.GeneratedPost{Body: "Simple tools usually win.", Prompt: "p", Model: "mock"}},
		&adminIDGeneratorStub{next: []uint64{777}},
		&adminRepositoryStub{},
	)
	service.now = func() time.Time { return now }

	result, err := service.CreatePost(context.Background(), 55, "")
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	if postsSvc.input.AuthorID != 66 || postsSvc.input.Source != posts.SourceAI {
		t.Fatalf("expected ai post create input to use ai source and ai account user")
	}
	if result.Generation == nil || result.Generation.Status != aiaccounts.PostGenerationStatusPosted {
		t.Fatalf("expected posted generation record")
	}
	if len(accounts.successMarks) != 1 {
		t.Fatalf("expected success mark")
	}
	if !accounts.successMarks[0].nextGenerateAt.After(accounts.successMarks[0].lastGeneratedAt) {
		t.Fatalf("expected next generate time after last generated time")
	}
}

func TestServiceGeneratePostContentUsesAITools(t *testing.T) {
	service := NewService(
		&adminTxManager{},
		&adminAIAccountsStub{},
		&adminUsersStub{},
		&adminPostsStub{},
		&adminAIToolsStub{result: aitools.GeneratedPost{
			Body:   "Small tools make room for better thinking.",
			Prompt: "prompt",
			Model:  "gpt-4.1",
		}},
		&adminGeneratorStub{},
		&adminIDGeneratorStub{},
		&adminRepositoryStub{},
	)

	result, err := service.GeneratePostContent(context.Background(), GeneratePostContentInput{
		Keywords:    []string{"tools", "clarity", "tools"},
		Description: "A concise post about deliberate software.",
	})
	if err != nil {
		t.Fatalf("GeneratePostContent: %v", err)
	}
	if result.Body != "Small tools make room for better thinking." {
		t.Fatalf("unexpected body %q", result.Body)
	}
	if result.Model != "gpt-4.1" {
		t.Fatalf("unexpected model %q", result.Model)
	}
}

func (s *adminAIAccountsStub) UpdateWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, a *aiaccounts.AIAccount) error {
	s.gotAccount = a
	return nil
}
