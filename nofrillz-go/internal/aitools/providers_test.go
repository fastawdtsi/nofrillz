package aitools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nofrillz/internal/config"
)

func TestProviderContracts(t *testing.T) {
	for _, kind := range []string{"anthropic", "xai"} {
		t.Run(kind, func(t *testing.T) {
			for _, state := range []string{"success", "incomplete", "error"} {
				t.Run(state, func(t *testing.T) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var payload map[string]any
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
							t.Error(err)
						}
						if payload["model"] != "configured-model" || payload["max_tokens"] != float64(512) {
							t.Error("missing configured model or output budget")
						}
						if !strings.Contains(stringMustJSON(payload), "source evidence") {
							t.Error("shared source context was not sent")
						}
						if kind == "anthropic" {
							if r.URL.Path != "/messages" || r.Header.Get("x-api-key") != "fixture-key" || r.Header.Get("anthropic-version") != "2023-06-01" {
								t.Error("incorrect Anthropic contract")
							}
						} else if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer fixture-key" {
							t.Error("incorrect xAI contract")
						}
						if state == "error" {
							w.WriteHeader(429)
							w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"secret-should-never-appear"}}`))
							return
						}
						if kind == "anthropic" {
							stop := "end_turn"
							if state == "incomplete" {
								stop = "max_tokens"
							}
							json.NewEncoder(w).Encode(map[string]any{"model": "resolved-model", "stop_reason": stop, "content": []map[string]string{{"type": "text", "text": "Source-based post."}}})
						} else {
							stop := "stop"
							if state == "incomplete" {
								stop = "length"
							}
							json.NewEncoder(w).Encode(map[string]any{"model": "resolved-model", "choices": []map[string]any{{"finish_reason": stop, "message": map[string]string{"content": "Source-based post."}}}})
						}
					}))
					defer server.Close()
					var provider Tools = NewAnthropic("fixture-key", server.URL, "configured-model")
					if kind == "xai" {
						provider = NewChatCompletions("fixture-key", server.URL, "configured-model")
					}
					got, err := provider.GeneratePostContent(context.Background(), GeneratePostInput{ContentMode: "research", Context: "source evidence"})
					if state == "success" {
						if err != nil || got.Body != "Source-based post." || got.Model != "resolved-model" {
							t.Fatalf("bad result: %+v %v", got, err)
						}
					} else if err == nil || strings.Contains(err.Error(), "secret-should-never-appear") {
						t.Fatalf("unsafe/missing error: %v", err)
					}
				})
			}
		})
	}
}
func stringMustJSON(value any) string { b, _ := json.Marshal(value); return string(b) }

func TestRegistryExtendsStableOptionsAndDoesNotExposeCredentials(t *testing.T) {
	t.Setenv("NOFRILLZ_ID_GENERATOR_REGION", "0")
	t.Setenv("NOFRILLZ_ID_GENERATOR_NODE", "0")
	t.Setenv("NOFRILLZ_AI_TOOLS_PROVIDER", "openai")
	t.Setenv("NOFRILLZ_AI_TOOLS_OPENAI_API_KEY", "fixture-secret")
	t.Setenv("NOFRILLZ_AI_TOOLS_ANTHROPIC_API_KEY", "")
	t.Setenv("NOFRILLZ_AI_TOOLS_XAI_API_KEY", "")
	t.Setenv("NOFRILLZ_AI_TOOLS_MODEL_OPTIONS_JSON", `[{"id":"openai_compact","name":"Compact","provider":"openai","model":"different-concrete-model"}]`)
	cfg, err := config.NewConfig("")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewRegistryFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Options()) != 4 {
		t.Fatal("catalog replacement lost stable defaults")
	}
	o, _ := registry.Get("openai_compact")
	if !o.Available || o.Model != "different-concrete-model" {
		t.Fatal("additional model not configured")
	}
	for _, id := range []string{"claude", "grok"} {
		o, _ := registry.Get(id)
		if o.Available {
			t.Fatal("unconfigured provider reported available")
		}
	}
	if strings.Contains(stringMustJSON(registry.Options()), "fixture-secret") {
		t.Fatal("catalog exposes credentials")
	}
}

func TestSharedSeedPromptDoesNotAskForDifferentItem(t *testing.T) {
	prompt := BuildPostPrompt(GeneratePostInput{ContentMode: "generative", SharedDraft: true, Context: "What do you call fake spaghetti? An impasta.", Description: "Create one clean joke.", RecentPosts: []string{"A previous joke."}})
	if strings.Contains(prompt, "Choose a different angle or subject") || !strings.Contains(prompt, "REPHRASE the shared content seed") {
		t.Fatal("variant prompt conflicts with shared-item identity")
	}
	instructions := Instructions(GeneratePostInput{ContentMode: "generative", ReviewBody: "A different joke"})
	if !strings.Contains(instructions, "SAME underlying item") || strings.Contains(instructions, "source metadata") {
		t.Fatal("wrong validation task for evergreen variant")
	}
}
