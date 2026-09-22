package aicontent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
	"nofrillz/internal/feed"
	"nofrillz/internal/posts"
	"nofrillz/internal/research"
	"nofrillz/internal/testdb"
	"testing"
	"time"
)

type fixtureIDs struct{ n uint64 }

func (i *fixtureIDs) MustNext() uint64 { i.n++; return i.n }

type fixtureProvider struct {
	body          string
	err           error
	inputs        []aitools.GeneratePostInput
	reviews       []aitools.GeneratePostInput
	reviewResult  string
	novelty       []aitools.GeneratePostInput
	noveltyResult string
	noveltyError  error
	before        func()
}

func (p *fixtureProvider) GeneratePostContent(_ context.Context, in aitools.GeneratePostInput) (aitools.GeneratedPost, error) {
	if in.Novelty != nil {
		p.novelty = append(p.novelty, in)
		body := p.noveltyResult
		if body == "" {
			body = `{"decision":"new"}`
		}
		return aitools.GeneratedPost{Body: body, Model: "fixture-reviewer"}, p.noveltyError
	}
	if in.ReviewBody != "" {
		p.reviews = append(p.reviews, in)
		body := p.reviewResult
		if body == "" {
			body = "__APPROVED__"
		}
		return aitools.GeneratedPost{Body: body, Model: "fixture-reviewer"}, nil
	}
	p.inputs = append(p.inputs, in)
	if p.before != nil {
		p.before()
	}
	return aitools.GeneratedPost{Body: p.body, Model: "fixture-model", Prompt: aitools.BuildPostPrompt(in)}, p.err
}

type fixtureResearch struct {
	items []research.Candidate
	calls int
}

func (r *fixtureResearch) Discover(context.Context, research.Request) ([]research.Candidate, error) {
	r.calls++
	return r.items, nil
}
func fixtureUser(t *testing.T, db *sql.DB, id uint64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users(id,email,username,first_name,last_name,about,account_type,password_hash) VALUES(?,?,?,?,?,?,'ai',?)`, id, fmt.Sprintf("test%d@example.invalid", id), fmt.Sprintf("test%d", id), "Content", "Account", "Mission", []byte("unused")); err != nil {
		t.Fatal(err)
	}
}
func claim(t *testing.T, s *Service, a *aiaccounts.AIAccount, now time.Time) *aiaccounts.AIAccount {
	t.Helper()
	if _, err := s.DB.Exec(`UPDATE ai_accounts SET next_generate_at=? WHERE id=?`, now, a.ID); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Accounts.ClaimDueAIAccounts(context.Background(), now, 1, 15*time.Minute)
	if err != nil || len(rows) != 1 {
		t.Fatalf("claim: %v (%d)", err, len(rows))
	}
	return rows[0]
}
func fixtureService(t *testing.T, mode string) (*Service, *aiaccounts.AIAccount, *fixtureResearch, *fixtureProvider, *fixtureProvider, *fixtureProvider) {
	t.Helper()
	db := testdb.Open(t)
	for _, id := range []uint64{1, 10, 11, 12} {
		fixtureUser(t, db, id)
	}
	openai := &fixtureProvider{body: "The observatory published a new survey of nearby stars."}
	claude := &fixtureProvider{body: "A new nearby-star survey was published by the observatory."}
	grok := &fixtureProvider{err: errors.New("fixture provider unavailable")}
	registry, err := aitools.NewRegistry([]aitools.Option{{ID: "openai", Provider: "openai", Model: "test-a", Tools: openai}, {ID: "claude", Provider: "anthropic", Model: "test-b", Tools: claude}, {ID: "grok", Provider: "xai", Model: "test-c", Tools: grok}})
	if err != nil {
		t.Fatal(err)
	}
	ids := &fixtureIDs{n: 100}
	now := time.Now().UTC().Truncate(time.Second)
	source := &fixtureResearch{items: []research.Candidate{{Key: research.Fingerprint("https://example.org/survey"), Title: "New survey released", Context: "The observatory published a new survey of nearby stars. No other findings have been announced.", Sources: []research.Source{{URL: "https://example.org/survey", Name: "Observatory", Title: "New survey released", PublishedAt: &now, DiscoveredAt: now}}}}}
	s := New(db, registry, source, posts.NewService(posts.NewRepository(db), ids), ids, aiaccounts.Schedule{Development: true, MinInterval: time.Minute, MaxInterval: 2 * time.Minute}, nil)
	s.Now = func() time.Time { return now }
	a := &aiaccounts.AIAccount{ID: 2, UserID: 1, Enabled: true, ContentMode: mode, Topic: "Space", Description: "Report meaningful astronomy developments without invented claims.", StylePrompt: "Factual and concise", SourceURLs: []string{"https://example.org/feed"}, SourceMaxAgeHours: 168, CheckIntervalSeconds: 1800, ModelOptions: []string{"openai", "claude", "grok"}, DefaultModelOption: "openai", MinPostsPerDay: 1, MaxPostsPerDay: 2, GenerationStatus: "idle"}
	if err = s.Accounts.Create(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	return s, a, source, openai, claude, grok
}
func TestResearchVariantsPreferencesDedupAndRecovery(t *testing.T) {
	s, a, sources, openai, claude, grok := fixtureService(t, "research")
	ctx := context.Background()
	now := s.Now()
	if err := s.Process(ctx, claim(t, s, a, now)); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(ctx, a.ID, 20)
	if err != nil || len(items) != 1 {
		t.Fatalf("items: %v %d", err, len(items))
	}
	item := items[0]
	if len(item.Variants) != 3 || len(item.Sources) != 1 || item.PublishedAt == nil || item.Status != "complete" {
		t.Fatalf("missing relationship/provenance: %+v", item)
	}
	if sources.calls != 1 || len(openai.inputs) != 1 || len(claude.inputs) != 1 || len(grok.inputs) != 1 {
		t.Fatal("incorrect provider/research volume")
	}
	if openai.inputs[0].Context != claude.inputs[0].Context || openai.inputs[0].Sources != claude.inputs[0].Sources {
		t.Fatal("providers did not share facts")
	}
	feedRepo := feed.NewRepository(s.DB)
	for _, tt := range []struct {
		user     uint64
		global   string
		override string
		want     string
	}{{10, "openai", "", "openai"}, {11, "claude", "", "claude"}, {12, "openai", "claude", "claude"}, {12, "claude", "grok", "claude"}, {12, "grok", "", "openai"}} {
		if _, err = s.DB.Exec(`UPDATE users SET ai_model_preference=? WHERE id=?`, tt.global, tt.user); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(`INSERT INTO follows(follower_id,following_id,ai_model_override) VALUES(?,1,NULLIF(?,'')) ON DUPLICATE KEY UPDATE ai_model_override=VALUES(ai_model_override)`, tt.user, tt.override); err != nil {
			t.Fatal(err)
		}
		for _, discover := range []bool{true, false} {
			var result []*feed.Post
			if discover {
				result, err = feedRepo.ListDiscover(ctx, tt.user, nil, 10)
			} else {
				result, err = feedRepo.List(ctx, tt.user, nil, 10)
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != 1 || result[0].ModelOption != tt.want || result[0].ContentItemID == nil || *result[0].ContentItemID != item.ID || result[0].UserID != 1 {
				t.Fatalf("wrong variant for %+v: %+v", tt, result)
			}
			cursor := result[0].SortID
			page, e := feedRepo.ListDiscover(ctx, tt.user, &cursor, 10)
			if e != nil || len(page) != 0 {
				t.Fatal("logical cursor repeats variants")
			}
		}
		profile, err := s.Posts.ListByUser(ctx, 1, tt.user, 0, 10)
		if err != nil || len(profile) != 1 {
			t.Fatalf("profile variants duplicated: %v", err)
		}
		page, err := s.Posts.ListByUser(ctx, 1, tt.user, profile[0].ID, 10)
		if err != nil || len(page) != 0 {
			t.Fatalf("profile pagination repeated logical item: %v", err)
		}
	}
	// Simulate recovery after variants committed but before account finalization.
	if _, err = s.DB.Exec(`UPDATE ai_content_items SET status='processing' WHERE id=?`, item.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Process(ctx, claim(t, s, a, now.Add(3*time.Minute))); err != nil {
		t.Fatal(err)
	}
	if sources.calls != 1 || len(openai.inputs) != 1 || len(claude.inputs) != 1 || len(grok.inputs) != 1 {
		t.Fatal("recovery regenerated or re-researched an existing item")
	}
	if err = s.Process(ctx, claim(t, s, a, now.Add(6*time.Minute))); err != nil {
		t.Fatal(err)
	}
	saved, err := s.Accounts.GetByID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.LastCheckOutcome != "no_content" || saved.LastGeneratedAt == nil || saved.LastCheckedAt == nil || saved.ConsecutiveFailures != 0 {
		t.Fatalf("quiet check was not successful: %+v", saved)
	}
	items, _ = s.List(ctx, a.ID, 20)
	if len(items) != 1 || len(openai.inputs) != 1 {
		t.Fatal("duplicate source generated again")
	}
	// With both explicit preferences/default missing, selection remains deterministic.
	if _, err = s.DB.Exec(`UPDATE ai_accounts SET default_model_option='grok' WHERE id=?`, a.ID); err != nil {
		t.Fatal(err)
	}
	fallback, err := feedRepo.ListDiscover(ctx, 12, nil, 10)
	if err != nil || len(fallback) != 1 || fallback[0].ModelOption != "claude" {
		t.Fatal("lexical successful-variant fallback failed")
	}
	// Reactions stay attached to the version the same reader actually read.
	var original uint64
	if err = s.DB.QueryRow(`SELECT post_id FROM ai_content_variants WHERE content_item_id=? AND option_id='openai'`, item.ID).Scan(&original); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(`INSERT INTO likes(post_id,user_id) VALUES(?,10)`, original); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(`INSERT INTO post_bookmarks(post_id,user_id) VALUES(?,10)`, original); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(`UPDATE users SET ai_model_preference='claude' WHERE id=10`); err != nil {
		t.Fatal(err)
	}
	changed, err := feedRepo.ListDiscover(ctx, 10, nil, 10)
	if err != nil || len(changed) != 1 || changed[0].Liked || changed[0].IsBookmarked {
		t.Fatal("reactions were merged across variants")
	}
	retained, err := s.Posts.GetByID(ctx, original, 10)
	if err != nil || !retained.Liked || !retained.IsBookmarked {
		t.Fatal("switching preference lost the original variant's reactions")
	}
}

func TestGenerativeSharedSeedAndProviderDisable(t *testing.T) {
	s, a, sources, first, second, third := fixtureService(t, "generative")
	a.ModelOptions = []string{"openai", "claude"}
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	if sources.calls != 0 || len(third.inputs) != 0 {
		t.Fatal("generative content researched or disabled provider called")
	}
	if len(second.inputs) != 1 || !second.inputs[0].SharedDraft || second.inputs[0].Context != first.body {
		t.Fatal("generative variants did not share an underlying item")
	}
}
func TestResearchCanRejectWithoutPublishing(t *testing.T) {
	s, a, _, first, second, third := fixtureService(t, "research")
	first.body = "__NO_POST__"
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "not_significant" || saved.LastGeneratedAt != nil || saved.ConsecutiveFailures != 0 {
		t.Fatal("no-post check misclassified")
	}
	if len(second.inputs) != 0 || len(third.inputs) != 0 {
		t.Fatal("rejected item spent on other providers")
	}
	// Recover after persisting the editorial rejection but before finalizing its item.
	if _, err := s.DB.Exec(`UPDATE ai_content_items SET status='processing' WHERE ai_account_id=?`, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Process(context.Background(), claim(t, s, a, s.Now().Add(3*time.Minute))); err != nil {
		t.Fatal(err)
	}
	if len(first.inputs) != 1 || len(second.inputs) != 0 || len(third.inputs) != 0 {
		t.Fatal("recovery regenerated a rejected research item")
	}

	var count int
	s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count)
	if count != 0 {
		t.Fatal("rejection published")
	}
}
func TestPauseWhileGeneratingFencesVariant(t *testing.T) {
	s, a, _, first, _, _ := fixtureService(t, "generative")
	first.before = func() {
		a.Enabled = false
		if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
			t.Fatal(err)
		}
	}
	err := s.Process(context.Background(), claim(t, s, a, s.Now()))
	if !errors.Is(err, aiaccounts.ErrClaimLost) {
		t.Fatalf("stale result accepted: %v", err)
	}
	var count int
	if e := s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); e != nil || count != 0 {
		t.Fatal("paused worker published")
	}
}

type failedPosts struct{ *posts.Repository }

func (p failedPosts) CreateWithExecutor(context.Context, posts.CreateExecutor, *posts.Post) error {
	return errors.New("fixture post write failure")
}
func TestPostFailureRollsBackVariantAndBacksOff(t *testing.T) {
	s, a, _, _, _, _ := fixtureService(t, "generative")
	s.Posts = posts.NewService(failedPosts{posts.NewRepository(s.DB)}, s.IDs)
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err == nil {
		t.Fatal("expected post creation failure")
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastGeneratedAt != nil || saved.LastCheckOutcome != "failed" || saved.NextGenerateAt.Sub(s.Now()) < 15*time.Minute {
		t.Fatal("failed post advanced success state")
	}
	var count int
	s.DB.QueryRow(`SELECT COUNT(*) FROM ai_content_variants WHERE post_id IS NOT NULL`).Scan(&count)
	if count != 0 {
		t.Fatal("failed post retained published variant")
	}
}

func TestResearchEvidenceReviewRejectsUnsupportedClaims(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "research")
	first.reviewResult = "__REJECTED__"
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("unsupported research draft was published")
	}
	if len(first.reviews) != 2 || first.reviews[0].ReviewBody != first.body || first.reviews[1].ReviewBody != second.body || first.reviews[0].Context != first.inputs[0].Context {
		t.Fatal("evidence review did not use original shared source")
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "failed" || saved.ConsecutiveFailures != 1 {
		t.Fatal("all-rejected item did not back off")
	}
}

func TestGenerativeRejectsDifferentItemAndKeepsSeed(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "generative")
	first.body = "What do you call fake spaghetti? An impasta."
	second.body = "Why did the coffee file a police report? Because it got mugged."
	first.reviewResult = "__REJECTED__"
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(context.Background(), a.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	successful := 0
	for _, variant := range items[0].Variants {
		if variant.Status == "published" {
			successful++
			if variant.OptionID != "openai" {
				t.Fatal("different joke published under same item")
			}
		}
	}
	if successful != 1 || len(first.reviews) != 1 || first.reviews[0].ContentMode != "generative" || first.reviews[0].Context != first.body {
		t.Fatal("shared seed validation did not run")
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "partially_published" || saved.ConsecutiveFailures != 0 {
		t.Fatal("rejected variant broke successful seed")
	}
}

func TestDefaultProviderFailureDoesNotBlockOtherResearchVariants(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "research")
	first.err = errors.New("fixture default provider outage")
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(context.Background(), a.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.reviews) != 0 || len(second.reviews) != 1 {
		t.Fatal("failed default provider was called again for review")
	}
	published := 0
	for _, v := range items[0].Variants {
		if v.Status == "published" {
			published++
			if v.OptionID != "claude" {
				t.Fatal("unexpected successful provider")
			}
		}
	}
	if published != 1 {
		t.Fatal("default outage blocked successful alternative")
	}
}

func TestEditorialExclusionIsSuccessfulQuietCheck(t *testing.T) {
	s, a, _, first, second, third := fixtureService(t, "research")
	a.Exclusions = "Promotional customer case studies"
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	first.reviewResult = "__NO_POST__"
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	if len(first.reviews) != 1 || first.reviews[0].Description != a.Description || first.reviews[0].Exclusions != a.Exclusions {
		t.Fatal("editorial criteria were omitted from review")
	}
	if len(second.inputs) != 0 || len(third.inputs) != 0 {
		t.Fatal("excluded item incurred additional variant calls")
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "not_significant" || saved.LastGeneratedAt != nil || saved.ConsecutiveFailures != 0 {
		t.Fatal("editorial no-post decision was treated as failure")
	}
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("excluded item published")
	}
}

func TestDailyScheduleDoesNotAccelerateAfterFailure(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "generative")
	s.Schedule = aiaccounts.Schedule{}
	a.CheckIntervalSeconds = 86400
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	first.err, second.err = errors.New("fixture outage"), errors.New("fixture outage")
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	saved, err := s.Accounts.GetByID(context.Background(), a.ID)
	if err != nil || saved.NextGenerateAt == nil || !saved.NextGenerateAt.Equal(s.Now().Add(24*time.Hour)) || saved.LastCheckOutcome != "failed" {
		t.Fatalf("failure shortened daily interval: %+v %v", saved, err)
	}
}
