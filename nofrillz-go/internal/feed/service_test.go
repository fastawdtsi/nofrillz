package feed

import (
	"context"
	"testing"
	"time"
)

type inMemoryRepository struct {
	posts   []*Post
	follows map[uint64]map[uint64]bool
}

func (r *inMemoryRepository) List(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, error) {
	visible := map[uint64]bool{}
	visible[requesterUserID] = true
	for followingID := range r.follows[requesterUserID] {
		visible[followingID] = true
	}

	result := make([]*Post, 0, limit)
	for _, post := range r.posts {
		if !visible[post.UserID] {
			continue
		}
		if cursor != nil {
			if !(post.ID < *cursor) {
				continue
			}
		}
		result = append(result, post)
		if len(result) == limit {
			break
		}
	}

	return result, nil
}

func TestServiceListPaginationIncludesRequesterPosts(t *testing.T) {
	now := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)

	repo := &inMemoryRepository{
		posts: []*Post{
			{ID: 110, UserID: 1, Username: "u1", Body: "p1", Created: now.Add(-1 * time.Minute)},
			{ID: 109, UserID: 2, Username: "u2", Body: "p2", Created: now.Add(-2 * time.Minute)},
			{ID: 108, UserID: 3, Username: "u3", Body: "p3", Created: now.Add(-3 * time.Minute)},
			{ID: 107, UserID: 1, Username: "u1", Body: "p4", Created: now.Add(-4 * time.Minute)},
			{ID: 106, UserID: 2, Username: "u2", Body: "p5", Created: now.Add(-5 * time.Minute)},
			{ID: 105, UserID: 3, Username: "u3", Body: "p6", Created: now.Add(-6 * time.Minute)},
			{ID: 104, UserID: 1, Username: "u1", Body: "p7", Created: now.Add(-7 * time.Minute)},
		},
		follows: map[uint64]map[uint64]bool{
			1: {
				2: true,
				3: true,
			},
		},
	}

	service := NewService(repo)

	page1, cursor1, err := service.List(context.Background(), 1, nil, 3)
	if err != nil {
		t.Fatalf("page1 List: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 posts on page1, got %d", len(page1))
	}
	if cursor1 == nil {
		t.Fatalf("expected next cursor for page1")
	}

	page2, cursor2, err := service.List(context.Background(), 1, cursor1, 3)
	if err != nil {
		t.Fatalf("page2 List: %v", err)
	}
	if len(page2) != 3 {
		t.Fatalf("expected 3 posts on page2, got %d", len(page2))
	}
	if cursor2 == nil {
		t.Fatalf("expected next cursor for page2")
	}

	page3, cursor3, err := service.List(context.Background(), 1, cursor2, 3)
	if err != nil {
		t.Fatalf("page3 List: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("expected 1 post on page3, got %d", len(page3))
	}
	if cursor3 != nil {
		t.Fatalf("expected no next cursor for page3")
	}

	seen := map[uint64]bool{}
	all := append(append(page1, page2...), page3...)
	for _, post := range all {
		if seen[post.ID] {
			t.Fatalf("duplicate post across pages: %d", post.ID)
		}
		seen[post.ID] = true
	}

	for i := 1; i < len(all); i++ {
		prev := all[i-1]
		cur := all[i]
		ordered := prev.ID > cur.ID
		if !ordered {
			t.Fatalf("unstable order at index %d prev=%+v cur=%+v", i, prev, cur)
		}
	}

	feedPosts, _, err := service.List(context.Background(), 1, nil, 20)
	if err != nil {
		t.Fatalf("feed List: %v", err)
	}
	foundRequesterPost := false
	for _, post := range feedPosts {
		if post.UserID == 1 {
			foundRequesterPost = true
			break
		}
	}
	if !foundRequesterPost {
		t.Fatalf("expected feed to include requester posts")
	}
}

func (r *inMemoryRepository) ListDiscover(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, error) {
	var items []*Post
	for _, p := range r.posts {
		if cursor == nil || p.ID < *cursor {
			items = append(items, p)
			if len(items) == limit {
				break
			}
		}
	}
	return items, nil
}
