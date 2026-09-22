package aicontent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"nofrillz/internal/posts"
	"nofrillz/internal/research"
)

func publishHistory(t *testing.T, s *Service, userID uint64, body string) uint64 {
	t.Helper()
	p, err := s.Posts.CreatePost(context.Background(), posts.CreatePostInput{AuthorID: userID, Body: body, Source: posts.SourceAI})
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func TestDuplicateAgainstOldDeletedLegacyPostIsSuccessfulQuietCheck(t *testing.T) {
	s, a, _, first, second, third := fixtureService(t, "generative")
	old := publishHistory(t, s, a.UserID, "Why did the orange stop halfway up the hill? It ran out of juice!")
	if _, err := s.DB.Exec(`UPDATE posts SET deleted=NOW() WHERE id=?`, old); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		publishHistory(t, s, a.UserID, fmt.Sprintf("Distinct archived observation number %d about astronomy.", i))
	}
	first.body = "WHY did the orange stop halfway up the hill—it ran out of juice."
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	saved, err := s.Accounts.GetByID(context.Background(), a.ID)
	if err != nil || saved.LastCheckOutcome != "duplicate" || saved.LastGeneratedAt != nil || saved.ConsecutiveFailures != 0 || saved.NextGenerateAt.Sub(s.Now()) > 2*time.Minute {
		t.Fatalf("repeat wasn't a successful quiet check: %+v %v", saved, err)
	}
	if len(first.novelty) != 0 || len(second.inputs) != 0 || len(third.inputs) != 0 {
		t.Fatal("exact repeat spent on reviews or additional model variants")
	}
	items, err := s.List(context.Background(), a.ID, 10)
	if err != nil || len(items) != 1 || items[0].Status != "skipped" || items[0].PublishedAt != nil || items[0].Variants[0].Status != "duplicate" {
		t.Fatalf("duplicate audit missing: %+v %v", items, err)
	}
	var count int
	if err = s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil || count != 26 {
		t.Fatal("duplicate created a post")
	}
	// Crash recovery retains the no-post decision and does not regenerate.
	if _, err = s.DB.Exec(`UPDATE ai_content_items SET status='processing' WHERE id=?`, items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Process(context.Background(), claim(t, s, a, s.Now().Add(3*time.Minute))); err != nil {
		t.Fatal(err)
	}
	if len(first.inputs) != 1 || len(second.inputs) != 0 {
		t.Fatal("recovery retried a duplicate")
	}
}

func TestRewordedOldInformationRetrievedBeyondRecentWindow(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "generative")
	old := publishHistory(t, s, a.UserID, "Serve meals on smaller plates to help keep portions manageable.")
	for i := 0; i < 25; i++ {
		publishHistory(t, s, a.UserID, fmt.Sprintf("Separate astronomy observation %d about distant galaxies.", i))
	}
	first.body = "Choose a small plate at dinner; it can make sensible portions easier."
	first.noveltyResult = fmt.Sprintf(`{"decision":"duplicate","post_id":"%d"}`, old)
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	if len(first.novelty) != 1 || len(second.inputs) != 0 {
		t.Fatal("semantic duplicate did not stop variant generation")
	}
	prior := first.novelty[0].Novelty.History
	found := false
	for _, h := range prior {
		found = found || h.PostID == old
	}
	if !found || len(prior) > matchingHistoryLimit+recentHistoryLimit {
		t.Fatal("older relevant content was omitted or prompt was unbounded")
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "duplicate" {
		t.Fatal("semantic repeat was not suppressed")
	}
}

func TestAllModelsShareHistoryButSameItemVariantsAreAllowed(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "generative")
	a.ModelOptions = []string{"openai", "claude"}
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	// Identical sibling versions are valid; they are one logical item in feeds.
	second.body = first.body
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil || count != 2 || len(first.novelty) != 0 {
		t.Fatal("same-item variants incorrectly treated as past content")
	}
	// Switch to only Claude: OpenAI's historical content must still be checked.
	a.ModelOptions = []string{"claude"}
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	if err := s.Process(context.Background(), claim(t, s, a, s.Now().Add(3*time.Minute))); err != nil {
		t.Fatal(err)
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "duplicate" || len(second.inputs[1].RecentPosts) != 1 {
		t.Fatal("account history was not shared or sibling history wasn't grouped")
	}
}

func TestResearchDifferentURLSameInformationSuppressedButUpdateAllowed(t *testing.T) {
	s, a, sources, first, _, _ := fixtureService(t, "research")
	a.ModelOptions = []string{"openai"}
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	var old uint64
	if err := s.DB.QueryRow(`SELECT id FROM posts LIMIT 1`).Scan(&old); err != nil {
		t.Fatal(err)
	}
	sources.items[0].Key = research.Fingerprint("https://example.net/syndicated-survey")
	sources.items[0].Title = "Astronomers release nearby-star mapping"
	sources.items[0].Context = "The observatory has released its survey mapping stars nearby."
	first.body = "A nearby-star mapping survey is now available from the observatory."
	first.noveltyResult = fmt.Sprintf(`{"decision":"duplicate","post_id":"%d"}`, old)
	if err := s.Process(context.Background(), claim(t, s, a, s.Now().Add(3*time.Minute))); err != nil {
		t.Fatal(err)
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "duplicate" {
		t.Fatal("syndicated/reworded source was posted again")
	}
	// A materially new result on the same beat should still publish.
	sources.items[0].Key = research.Fingerprint("https://example.org/survey-update")
	sources.items[0].Title = "Survey reveals six planets"
	sources.items[0].Context = "The observatory's survey has now identified six planets around nearby stars."
	first.body = "The observatory reported six newly identified planets in its nearby-star survey."
	first.noveltyResult = `{"decision":"new"}`
	if err := s.Process(context.Background(), claim(t, s, a, s.Now().Add(6*time.Minute))); err != nil {
		t.Fatal(err)
	}
	saved, _ = s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "published" {
		t.Fatal("new information on an existing topic was suppressed")
	}
}

func TestNoveltyReviewFailureCannotPublishUncheckedContent(t *testing.T) {
	for _, response := range []string{"invalid", `{"decision":"uncertain"}`, `{"decision":"duplicate","post_id":"999999"}`} {
		t.Run(response, func(t *testing.T) {
			s, a, _, first, second, _ := fixtureService(t, "generative")
			publishHistory(t, s, a.UserID, "Take a short break from sitting.")
			first.noveltyResult = response
			second.noveltyResult = response
			if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil || count != 1 {
				t.Fatal("unchecked content was published")
			}
			saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
			if saved.LastCheckOutcome != "failed" || saved.NextGenerateAt.Sub(s.Now()) < 15*time.Minute {
				t.Fatal("failed novelty review did not back off")
			}
		})
	}
}

func TestNoveltyReviewerOutageFallsBackWithoutBypass(t *testing.T) {
	s, a, _, first, second, _ := fixtureService(t, "generative")
	old := publishHistory(t, s, a.UserID, "An earlier star survey.")
	a.ModelOptions = []string{"claude"}
	if err := s.Accounts.UpdateWithExecutor(context.Background(), s.DB, a); err != nil {
		t.Fatal(err)
	}
	first.noveltyError = errors.New("fixture reviewer outage")
	second.noveltyResult = fmt.Sprintf(`{"decision":"duplicate","post_id":"%d"}`, old)
	if err := s.Process(context.Background(), claim(t, s, a, s.Now())); err != nil {
		t.Fatal(err)
	}
	if len(first.novelty) != 1 || len(second.novelty) != 1 {
		t.Fatal("reviewer did not fall back to the writer")
	}
	saved, _ := s.Accounts.GetByID(context.Background(), a.ID)
	if saved.LastCheckOutcome != "duplicate" {
		t.Fatal("fallback bypassed duplicate protection")
	}
}
