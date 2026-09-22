package aiaccounts_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/feed"
	"nofrillz/internal/posts"
	"nofrillz/internal/testdb"
)

type testIDs struct{ next uint64 }

func (g *testIDs) MustNext() uint64 { g.next++; return g.next }
func TestMySQLClaimFencingAndNormalDiscover(t *testing.T) {
	db := testdb.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	due := now.Add(-time.Minute)
	repo := aiaccounts.NewRepository(db)
	for i := uint64(1); i <= 6; i++ {
		_, err := db.Exec("INSERT INTO users (id,email,username,first_name,last_name,about,password_hash,account_type) VALUES (?,?,?,?,?,?,?,'ai')", i, fmt.Sprintf("%d@example.invalid", i), fmt.Sprintf("content%d", i), "Test", "Account", "test", []byte("unused"))
		if err != nil {
			t.Fatal(err)
		}
		a := &aiaccounts.AIAccount{ID: i, UserID: i, Enabled: i != 4, Topic: "test", SystemPrompt: "test", MinPostsPerDay: 1, MaxPostsPerDay: 2, NextGenerateAt: &due, GenerationStatus: aiaccounts.GenerationStatusIdle}
		if i == 5 {
			future := now.Add(time.Hour)
			a.NextGenerateAt = &future
		}
		if err := repo.Create(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("UPDATE users SET blocked=? WHERE id=6", now); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	claims := make(chan *aiaccounts.AIAccount, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := repo.ClaimDueAIAccounts(ctx, now, 1, 15*time.Minute)
			if err != nil {
				errs <- err
				return
			}
			for _, a := range rows {
				claims <- a
			}
		}()
	}
	wg.Wait()
	close(claims)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	owned := map[uint64]*aiaccounts.AIAccount{}
	for a := range claims {
		if a.ID > 3 || owned[a.ID] != nil || a.ClaimToken == "" {
			t.Fatalf("invalid/duplicate claim: %+v", a)
		}
		owned[a.ID] = a
	}
	// A locking scan can briefly lock more rows than LIMIT. SKIP LOCKED
	// deliberately returns early; the next poll must recover every remaining row.
	remaining, err := repo.ClaimDueAIAccounts(ctx, now, 3, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range remaining {
		if owned[a.ID] != nil || a.ID > 3 {
			t.Fatal("duplicate or ineligible claim")
		}
		owned[a.ID] = a
	}
	if len(owned) != 3 {
		t.Fatalf("expected only three due accounts; got %d", len(owned))
	}
	later := now.Add(20 * time.Minute)
	reclaimed, err := repo.ClaimDueAIAccounts(ctx, later, 3, 15*time.Minute)
	if err != nil || len(reclaimed) != 3 {
		t.Fatalf("stale recovery: %v, %d", err, len(reclaimed))
	}
	current := reclaimed[0]
	old := owned[current.ID]
	next := later.Add(time.Hour)
	if current.ClaimToken == old.ClaimToken {
		t.Fatal("reused claim token")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishClaimWithExecutor(ctx, tx, old, later, next, true, ""); !errors.Is(err, aiaccounts.ErrClaimLost) {
		t.Fatalf("stale owner can publish: %v", err)
	}
	tx.Rollback()
	// Rollback after reserving completion must leave the current claim usable.
	tx, _ = db.BeginTx(ctx, nil)
	if err := repo.FinishClaimWithExecutor(ctx, tx, current, later, next, true, ""); err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	tx, _ = db.BeginTx(ctx, nil)
	if err := repo.FinishClaimWithExecutor(ctx, tx, current, later, next, true, ""); err != nil {
		t.Fatal(err)
	}
	service := posts.NewService(posts.NewRepository(db), &testIDs{100})
	post, err := service.CreatePostWithExecutor(ctx, tx, posts.CreatePostInput{AuthorID: current.UserID, Body: "An ordinary post from an AI account.", Source: posts.SourceAI})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreatePostGenerationWithExecutor(ctx, tx, &aiaccounts.PostGeneration{ID: 500, AIAccountID: current.ID, PostID: &post.ID, Status: aiaccounts.PostGenerationStatusPosted, FinalBody: post.Body}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishClaimWithExecutor(ctx, db, current, later, next, true, ""); !errors.Is(err, aiaccounts.ErrClaimLost) {
		t.Fatal("same claim completed twice")
	}
	saved, err := repo.GetByID(ctx, current.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.LastGeneratedAt == nil || !saved.LastGeneratedAt.Equal(later) || saved.NextGenerateAt == nil || !saved.NextGenerateAt.Equal(next) || saved.ClaimToken != "" || saved.GenerationStatus != "idle" {
		t.Fatalf("schedule not finalized: %+v", saved)
	}
	feedRepo := feed.NewRepository(db)
	found, err := feedRepo.ListDiscover(ctx, 4, nil, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != post.ID || found[0].UserID != current.UserID || found[0].Source != "ai" {
		t.Fatalf("normal discover missing AI post: %+v", found)
	}
	for _, query := range []string{"INSERT INTO likes(post_id,user_id) VALUES (?,4)", "INSERT INTO post_bookmarks(post_id,user_id) VALUES (?,4)"} {
		if _, err := db.Exec(query, post.ID); err != nil {
			t.Fatal(err)
		}
	}
	found, err = feedRepo.ListDiscover(ctx, 4, nil, 20)
	if err != nil || !found[0].Liked || !found[0].IsBookmarked {
		t.Fatalf("ordinary interactions lost: %v", err)
	}
	hidden, err := feedRepo.ListDiscover(ctx, 4, &post.ID, 20)
	if err != nil || len(hidden) != 0 {
		t.Fatal("discover cursor failed")
	}
	// Pausing invalidates an in-flight claim, even after its generation returned.
	paused := reclaimed[1]
	paused.Enabled = false
	paused.NextGenerateAt = nil
	if err := repo.UpdateWithExecutor(ctx, db, paused); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishClaimWithExecutor(ctx, db, paused, later, next, true, ""); !errors.Is(err, aiaccounts.ErrClaimLost) {
		t.Fatal("paused account published")
	}
	if _, err := db.Exec("UPDATE users SET blocked=? WHERE id=?", later, current.UserID); err != nil {
		t.Fatal(err)
	}
	hidden, err = feedRepo.ListDiscover(ctx, 4, nil, 20)
	if err != nil || len(hidden) != 0 {
		t.Fatal("blocked author visible")
	}
}
