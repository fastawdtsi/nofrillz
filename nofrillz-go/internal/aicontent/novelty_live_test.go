package aicontent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"nofrillz/internal/aitools"
	"nofrillz/internal/config"
)

// Explicit, bounded paid-provider verification. Fixtures live only in the
// isolated integration database; no fabricated content enters the local feed.
func TestLiveNoveltyReview(t *testing.T) {
	if os.Getenv("NOFRILLZ_LIVE_AI_TESTS") != "1" {
		t.Skip("set NOFRILLZ_LIVE_AI_TESTS=1 for four paid OpenAI novelty reviews")
	}
	key := os.Getenv("NOFRILLZ_AI_TOOLS_OPENAI_API_KEY")
	if key == "" {
		t.Fatal("live review requires NOFRILLZ_AI_TOOLS_OPENAI_API_KEY")
	}
	provider, err := aitools.NewOpenAI(&config.OpenAIConfig{APIKey: key, Model: "gpt-4.1-mini", TimeoutSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, old, candidate string
		duplicate            bool
	}{
		{"reworded_old_joke", "Why did the orange stop halfway up the hill? It ran out of juice!", "An orange couldn't finish climbing the hill. The reason? It had run out of juice.", true},
		{"reworded_advice", "Serve meals on smaller plates to help keep portions manageable.", "Choose a small plate at dinner; it can make sensible portions easier.", true},
		{"repeated_information", "The observatory published a new survey of nearby stars on Monday.", "A new catalog mapping neighboring stars was released by the observatory on Monday.", true},
		{"substantive_update", "The observatory announced that its nearby-star survey has identified five planets.", "The observatory announced that its nearby-star survey has identified twelve planets.", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, a, _, _, _, _ := fixtureService(t, "generative")
			oldID := publishHistory(t, s, a.UserID, tt.old)
			for i := 0; i < 20; i++ {
				publishHistory(t, s, a.UserID, fmt.Sprintf("A separate archived entry %d about knitting and embroidery patterns.", i))
			}
			option := aitools.Option{ID: "openai", Provider: "openai", Model: "gpt-4.1-mini", Available: true, Tools: provider}
			s.Registry, err = aitools.NewRegistry([]aitools.Option{option})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			prior, err := s.checkNovelty(ctx, a, &Item{ID: s.IDs.MustNext()}, option, map[string]bool{}, tt.candidate)
			if err != nil {
				t.Fatal(err)
			}
			if (prior != nil) != tt.duplicate || (prior != nil && prior.PostID != oldID) {
				t.Fatalf("wrong live novelty decision; duplicate=%v want=%v", prior != nil, tt.duplicate)
			}
		})
	}
}
