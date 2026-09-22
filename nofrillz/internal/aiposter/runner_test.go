package aiposter

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"testing"
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/posts"
)

type fakeGenerator struct {
	post aigenerator.GeneratedPost
	err  error
}

func (g *fakeGenerator) GeneratePost(ctx context.Context, account *aiaccounts.AIAccount) (aigenerator.GeneratedPost, error) {
	return g.post, g.err
}

type fakePostCreator struct {
	post  *posts.Post
	err   error
	input posts.CreatePostInput
}

func (c *fakePostCreator) CreatePostWithExecutor(ctx context.Context, executor posts.CreateExecutor, input posts.CreatePostInput) (*posts.Post, error) {
	c.input = input
	if c.err != nil {
		return nil, c.err
	}
	return c.post, nil
}

type fakeTransaction struct {
	committed  bool
	rolledBack bool
}

func (t *fakeTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (t *fakeTransaction) Commit() error {
	t.committed = true
	return nil
}

func (t *fakeTransaction) Rollback() error {
	t.rolledBack = true
	return nil
}

type fakeTransactionManager struct {
	tx *fakeTransaction
}

func (m *fakeTransactionManager) Begin(ctx context.Context) (transaction, error) {
	m.tx = &fakeTransaction{}
	return m.tx, nil
}

type fakeIDGenerator struct {
	next []uint64
}

func (g *fakeIDGenerator) MustNext() uint64 {
	value := g.next[0]
	g.next = g.next[1:]
	return value
}

type successMark struct {
	accountID       uint64
	lastGeneratedAt time.Time
	nextGenerateAt  time.Time
}

type failureMark struct {
	accountID       uint64
	nextGenerateAt  time.Time
	generationError string
}

type fakeAIAccountsService struct {
	finishErr      error
	claimed        []*aiaccounts.AIAccount
	generations    []*aiaccounts.PostGeneration
	successMarks   []successMark
	failureMarks   []failureMark
	claimBatchSize int
}

func (s *fakeAIAccountsService) ClaimDueAIAccounts(ctx context.Context, now time.Time, limit int, staleAfter time.Duration) ([]*aiaccounts.AIAccount, error) {
	s.claimBatchSize = limit
	if len(s.claimed) == 0 {
		return nil, nil
	}
	result := s.claimed[:1]
	s.claimed = s.claimed[1:]
	return result, nil
}

func (s *fakeAIAccountsService) CreatePostGenerationWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, generation *aiaccounts.PostGeneration) error {
	s.generations = append(s.generations, generation)
	return nil
}

func (s *fakeAIAccountsService) MarkGenerationSuccessWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, accountID uint64, lastGeneratedAt time.Time, nextGenerateAt time.Time) error {
	s.successMarks = append(s.successMarks, successMark{
		accountID:       accountID,
		lastGeneratedAt: lastGeneratedAt,
		nextGenerateAt:  nextGenerateAt,
	})
	return nil
}

func (s *fakeAIAccountsService) MarkGenerationFailureWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, accountID uint64, nextGenerateAt time.Time, generationError string) error {
	s.failureMarks = append(s.failureMarks, failureMark{
		accountID:       accountID,
		nextGenerateAt:  nextGenerateAt,
		generationError: generationError,
	})
	return nil
}

func TestRunnerRunOnceSuccessCreatesAIPost(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	account := &aiaccounts.AIAccount{
		Enabled: true, ClaimToken: "test-token", ID: 11,
		UserID:         22,
		MinPostsPerDay: 1,
		MaxPostsPerDay: 1,
	}

	aiSvc := &fakeAIAccountsService{claimed: []*aiaccounts.AIAccount{account}}
	postCreator := &fakePostCreator{post: &posts.Post{ID: 333, Body: "hello from ai", Source: posts.SourceAI}}
	runner := NewRunner(
		nil,
		&fakeTransactionManager{},
		aiSvc,
		postCreator,
		&fakeGenerator{post: aigenerator.GeneratedPost{Body: "hello from ai", Prompt: "prompt", Model: "mock"}},
		&fakeIDGenerator{next: []uint64{444}},
		time.Minute,
		10,
		15*time.Minute,
	)
	runner.now = func() time.Time { return now }
	runner.rng = randForTest()

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if postCreator.input.AuthorID != 22 || postCreator.input.Source != posts.SourceAI {
		t.Fatalf("expected AI post creation input to use ai source and account author")
	}
	if len(aiSvc.generations) != 1 || aiSvc.generations[0].Status != aiaccounts.PostGenerationStatusPosted {
		t.Fatalf("expected posted generation record to be created")
	}
	if len(aiSvc.successMarks) != 1 {
		t.Fatalf("expected success mark to be recorded")
	}
	if aiSvc.successMarks[0].accountID != 11 {
		t.Fatalf("expected success mark for claimed account")
	}
	if !aiSvc.successMarks[0].nextGenerateAt.After(aiSvc.successMarks[0].lastGeneratedAt) {
		t.Fatalf("expected next generate time after last generated time")
	}
	if len(aiSvc.failureMarks) != 0 {
		t.Fatalf("expected no failure marks on success")
	}
}

func TestRunnerRunOnceFailureCreatesFailureRecord(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	account := &aiaccounts.AIAccount{
		Enabled: true, ClaimToken: "test-token", ID: 11,
		UserID:         22,
		MinPostsPerDay: 1,
		MaxPostsPerDay: 1,
	}

	aiSvc := &fakeAIAccountsService{claimed: []*aiaccounts.AIAccount{account}}
	runner := NewRunner(
		nil,
		&fakeTransactionManager{},
		aiSvc,
		&fakePostCreator{},
		&fakeGenerator{err: errors.New("generator boom")},
		&fakeIDGenerator{next: []uint64{777}},
		time.Minute,
		10,
		15*time.Minute,
	)
	runner.now = func() time.Time { return now }
	runner.rng = randForTest()

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(aiSvc.generations) != 1 || aiSvc.generations[0].Status != aiaccounts.PostGenerationStatusFailed {
		t.Fatalf("expected failed generation record to be created")
	}
	if len(aiSvc.failureMarks) != 1 {
		t.Fatalf("expected failure mark to be recorded")
	}
	if aiSvc.failureMarks[0].generationError != "generator boom" {
		t.Fatalf("expected failure reason to be recorded")
	}
	if len(aiSvc.successMarks) != 0 {
		t.Fatalf("expected no success marks on failure")
	}
}

func randForTest() *rand.Rand {
	return rand.New(rand.NewSource(1))
}

func (s *fakeAIAccountsService) FinishClaimWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, a *aiaccounts.AIAccount, now, next time.Time, success bool, message string) error {
	if s.finishErr != nil {
		return s.finishErr
	}
	if success {
		return s.MarkGenerationSuccessWithExecutor(ctx, executor, a.ID, now, next)
	}
	return s.MarkGenerationFailureWithExecutor(ctx, executor, a.ID, next, message)
}

func TestRunnerDisabledAndLostClaimsNeverCreatePosts(t *testing.T) {
	for _, tt := range []struct {
		name     string
		enabled  bool
		claimErr error
	}{{"disabled", false, nil}, {"stale owner", true, aiaccounts.ErrClaimLost}} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAIAccountsService{claimed: []*aiaccounts.AIAccount{{ID: 1, UserID: 2, Enabled: tt.enabled, ClaimToken: "old"}}, finishErr: tt.claimErr}
			creator := &fakePostCreator{}
			runner := NewRunner(nil, &fakeTransactionManager{}, svc, creator, &fakeGenerator{post: aigenerator.GeneratedPost{Body: "hello"}}, &fakeIDGenerator{}, time.Minute, 10, 15*time.Minute)
			if err := runner.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			if creator.input.AuthorID != 0 || len(svc.generations) != 0 {
				t.Fatal("cancelled account published")
			}
		})
	}
}
func TestFailureBackoffGrowsAndIsBounded(t *testing.T) {
	now := time.Now()
	for failures := 0; failures < 12; failures++ {
		delta := computeRetryAtWithRand(now, randForTest(), failures).Sub(now)
		base := 15 * time.Minute * time.Duration(1<<min(failures, 4))
		if delta < base || delta > base+base/2 {
			t.Fatalf("bad failure delay: %s", delta)
		}
	}
}

type selectiveGenerator struct{}

func (selectiveGenerator) GeneratePost(_ context.Context, a *aiaccounts.AIAccount) (aigenerator.GeneratedPost, error) {
	if a.ID == 1 {
		return aigenerator.GeneratedPost{}, errors.New("provider unavailable")
	}
	return aigenerator.GeneratedPost{Body: "second account succeeds"}, nil
}
func TestRunnerContinuesAfterGenerationFailure(t *testing.T) {
	svc := &fakeAIAccountsService{claimed: []*aiaccounts.AIAccount{{ID: 1, UserID: 11, Enabled: true, ClaimToken: "a"}, {ID: 2, UserID: 22, Enabled: true, ClaimToken: "b"}}}
	creator := &fakePostCreator{post: &posts.Post{ID: 123, Body: "second account succeeds"}}
	runner := NewRunner(nil, &fakeTransactionManager{}, svc, creator, selectiveGenerator{}, &fakeIDGenerator{next: []uint64{101, 102}}, time.Minute, 10, 15*time.Minute)
	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(svc.failureMarks) != 1 || len(svc.successMarks) != 1 || creator.input.AuthorID != 22 || svc.claimBatchSize != 1 {
		t.Fatal("failure prevented subsequent account processing")
	}
	if svc.generations[0].PostID != nil {
		t.Fatal("failed generation acquired a post")
	}
}
