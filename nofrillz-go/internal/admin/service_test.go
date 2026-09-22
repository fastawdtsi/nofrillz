package admin

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
	"nofrillz/internal/users"
)

type adminAIToolsStub struct {
	result aitools.GeneratedPost
	err    error
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
		&adminAIToolsStub{},
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
		Description:  "Provide practical tips for simple software.",
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

func TestServiceGeneratePostContentUsesAITools(t *testing.T) {
	service := NewService(
		&adminTxManager{},
		&adminAIAccountsStub{},
		&adminUsersStub{},
		&adminAIToolsStub{result: aitools.GeneratedPost{
			Body:   "Small tools make room for better thinking.",
			Prompt: "prompt",
			Model:  "gpt-4.1",
		}},
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

func TestUpdateAccountPauseResumeAndValidation(t *testing.T) {
	now := time.Now().UTC()
	accounts := &adminAIAccountsStub{gotAccount: &aiaccounts.AIAccount{ID: 1, UserID: 2, Enabled: true, Topic: "gardens", Description: "Offer general gardening tips.", MinPostsPerDay: 1, MaxPostsPerDay: 3}}
	profile := &adminUsersStub{got: &users.User{ID: 2, FirstName: "Gardening", AccountType: users.AccountTypeAI}}
	svc := NewService(&adminTxManager{}, accounts, profile, &adminAIToolsStub{}, &adminIDGeneratorStub{}, &adminRepositoryStub{})
	svc.now = func() time.Time { return now }
	svc.SetSchedule(aiaccounts.Schedule{Development: true, MinInterval: time.Minute, MaxInterval: 2 * time.Minute})
	enabled := false
	style := "Dry and concrete"
	record, err := svc.UpdateAccount(context.Background(), 1, UpdateAccountInput{Enabled: &enabled, StylePrompt: &style})
	if err != nil {
		t.Fatal(err)
	}
	if record.Account.Enabled || record.Account.NextGenerateAt != nil || record.Account.StylePrompt != style {
		t.Fatal("pause or mission edit failed")
	}
	enabled = true
	record, err = svc.UpdateAccount(context.Background(), 1, UpdateAccountInput{Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	delta := record.Account.NextGenerateAt.Sub(now)
	if delta < time.Minute || delta > 2*time.Minute {
		t.Fatal("resume did not stagger schedule")
	}
	svc.SetSchedule(aiaccounts.Schedule{})
	daily := 86400
	record, err = svc.UpdateAccount(context.Background(), 1, UpdateAccountInput{CheckIntervalSeconds: &daily})
	if err != nil || record.Account.NextGenerateAt == nil || !record.Account.NextGenerateAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("editing daily account accelerated its next check: %+v %v", record, err)
	}
	invalid := 0
	if _, err = svc.UpdateAccount(context.Background(), 1, UpdateAccountInput{MinPostsPerDay: &invalid}); err != ErrInvalidPostsPerDay {
		t.Fatalf("invalid rate accepted: %v", err)
	}
}

func TestContentMissionValidation(t *testing.T) {
	registry, err := aitools.NewRegistry([]aitools.Option{{ID: "openai", Provider: "openai", Model: "test", Tools: aitools.NewMockTools()}, {ID: "claude", Provider: "anthropic", Model: "test"}})
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{models: registry}
	for _, tt := range []struct {
		name   string
		change func(*aiaccounts.AIAccount)
	}{
		{"missing mission", func(a *aiaccounts.AIAccount) { a.Description = "" }},
		{"missing research source", func(a *aiaccounts.AIAccount) { a.ContentMode = "research" }},
		{"private source", func(a *aiaccounts.AIAccount) {
			a.ContentMode = "research"
			a.SourceURLs = []string{"https://127.0.0.1/feed"}
		}},
		{"empty selection", func(a *aiaccounts.AIAccount) { a.ModelOptions = []string{} }},
		{"unknown model", func(a *aiaccounts.AIAccount) { a.ModelOptions = []string{"missing"} }},
		{"duplicate model", func(a *aiaccounts.AIAccount) { a.ModelOptions = []string{"openai", "openai"} }},
		{"unselected default", func(a *aiaccounts.AIAccount) { a.DefaultModelOption = "claude" }},
		{"no available models", func(a *aiaccounts.AIAccount) { a.ModelOptions = []string{"claude"}; a.DefaultModelOption = "claude" }},
		{"too frequent", func(a *aiaccounts.AIAccount) { a.CheckIntervalSeconds = 1 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &aiaccounts.AIAccount{Description: "Useful content mission", Enabled: true}
			tt.change(a)
			if service.validateContent(a) == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	valid := &aiaccounts.AIAccount{Description: "Provide source-grounded research", ContentMode: "research", SourceURLs: []string{"https://example.org/feed"}, Enabled: true, ModelOptions: []string{"openai", "claude"}, DefaultModelOption: "openai"}
	if err := service.validateContent(valid); err != nil {
		t.Fatal(err)
	}
	// Catalog entries may be saved as disabled drafts while a source adapter is
	// missing. Enabling still requires real source configuration.
	draft := &aiaccounts.AIAccount{Description: "Verified history for today's date", ContentMode: "research", Enabled: false}
	if err := service.validateContent(draft); err != nil {
		t.Fatalf("disabled research draft rejected: %v", err)
	}
	draft.Enabled = true
	if service.validateContent(draft) == nil {
		t.Fatal("source-less research draft could be enabled")
	}
}
