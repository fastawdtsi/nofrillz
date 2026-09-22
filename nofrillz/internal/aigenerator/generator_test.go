package aigenerator

import (
	"context"
	"errors"
	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
	"nofrillz/internal/posts"
	"nofrillz/internal/users"
	"strings"
	"testing"
)

type captureTools struct {
	input aitools.GeneratePostInput
	body  string
	calls int
}

func (f *captureTools) GeneratePostContent(_ context.Context, in aitools.GeneratePostInput) (aitools.GeneratedPost, error) {
	f.input = in
	f.calls++
	return aitools.GeneratedPost{Body: f.body, Prompt: aitools.BuildPostPrompt(in), Model: "mock"}, nil
}

type profileReader struct{ user *users.User }

func (f profileReader) GetByID(context.Context, uint64) (*users.User, error) { return f.user, nil }

type historyReader struct {
	owner, requester uint64
	limit            int
}

func (f *historyReader) ListByUser(_ context.Context, owner, requester, cursor uint64, limit int) ([]*posts.Post, error) {
	f.owner = owner
	f.requester = requester
	f.limit = limit
	return []*posts.Post{{Body: "The old radio smells faintly of dust."}}, nil
}
func TestGeneratorSuppliesSavedPersonaAndRecentContext(t *testing.T) {
	provider := &captureTools{body: "this screwdriver has escaped again."}
	history := &historyReader{}
	g := NewAIToolsGenerator(provider, profileReader{&users.User{ID: 22, FirstName: "Dex", LastName: "Moreno", Username: "dex", About: "Fixes old radios"}}, history)
	result, err := g.GeneratePost(context.Background(), &aiaccounts.AIAccount{UserID: 22, Topic: "repair, obsolete gadgets", Description: "Grumpy tinkerer", StylePrompt: "Dry lowercase", SystemPrompt: "No advice lists"})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"Dex Moreno", "Fixes old radios", "Grumpy tinkerer", "Dry lowercase", "No advice lists", "The old radio smells faintly of dust.", "Do not repeat"} {
		if !strings.Contains(result.Prompt, part) {
			t.Fatalf("missing context %q", part)
		}
	}
	if history.owner != 22 || history.requester != 22 || history.limit != 15 || provider.input.LengthHint == "" {
		t.Fatalf("incorrect bounded history or length: %+v", history)
	}
	provider.body = "THE OLD RADIO SMELLS FAINTLY OF DUST!!!"
	if _, err = g.GeneratePost(context.Background(), &aiaccounts.AIAccount{UserID: 22}); !errors.Is(err, ErrRejected) {
		t.Fatalf("duplicate accepted: %v", err)
	}
}
func TestCandidateValidation(t *testing.T) {
	for _, body := range []string{"", "   ", "As an AI, I enjoy soup.", "```json", strings.Repeat("x", 1201), string([]byte{0xff})} {
		if !errors.Is(ValidateCandidate(body, nil), ErrRejected) {
			t.Fatalf("invalid accepted %q", body)
		}
	}
	if err := ValidateCandidate("Soup again. Excellent.", nil); err != nil {
		t.Fatal(err)
	}
	if !IsDuplicate("The cat has decided my favorite chair now belongs to him today.", "The cat has decided my favorite chair now belongs to him tonight.") {
		t.Fatal("obvious near duplicate missed")
	}
	if IsDuplicate("I watered the basil.", "I repaired the radio.") {
		t.Fatal("distinct posts flagged")
	}
}
