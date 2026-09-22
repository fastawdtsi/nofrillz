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

func TestBuildPostPromptIncludesKeywordsAndDescription(t *testing.T) {
	prompt := BuildPostPrompt(GeneratePostInput{
		Keywords:     []string{"minimalism", "tools", "minimalism"},
		Description:  "Thoughtful notes about deliberate product choices.",
		SystemPrompt: "Write short reflective posts.",
		StylePrompt:  "Use plain language.",
	})

	for _, expected := range []string{
		"Keywords:",
		"- minimalism",
		"- tools",
		"Description:",
		"Thoughtful notes about deliberate product choices.",
		"System prompt:",
		"Write short reflective posts.",
		"Style prompt:",
		"Use plain language.",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q\nprompt=%s", expected, prompt)
		}
	}
}

func TestOpenAIGeneratePostContentUsesResponsesAPI(t *testing.T) {
	var authHeader string
	var organizationHeader string
	var projectHeader string
	var requestBody openAIResponsesRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		organizationHeader = r.Header.Get("OpenAI-Organization")
		projectHeader = r.Header.Get("OpenAI-Project")

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"gpt-4.1",
			"output":[
				{"content":[{"type":"output_text","text":"Quiet products leave more room to think."}]}
			]
		}`))
	}))
	defer server.Close()

	client, err := NewOpenAI(&config.OpenAIConfig{
		APIKey:         "test-key",
		AccountID:      "acct_123",
		OrganizationID: "",
		ProjectID:      "proj_123",
		BaseURL:        server.URL,
		Model:          "gpt-4.1",
		TimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("NewOpenAI: %v", err)
	}

	result, err := client.GeneratePostContent(context.Background(), GeneratePostInput{
		Keywords:    []string{"product", "clarity"},
		Description: "A short post about simple software.",
	})
	if err != nil {
		t.Fatalf("GeneratePostContent: %v", err)
	}

	if authHeader != "Bearer test-key" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if organizationHeader != "acct_123" {
		t.Fatalf("expected account id to populate organization header fallback, got %q", organizationHeader)
	}
	if projectHeader != "proj_123" {
		t.Fatalf("expected project header, got %q", projectHeader)
	}
	if requestBody.Model != "gpt-4.1" {
		t.Fatalf("expected model to be forwarded")
	}
	if !strings.Contains(requestBody.Input, "A short post about simple software.") {
		t.Fatalf("expected prompt to include description")
	}
	if result.Body != "Quiet products leave more room to think." {
		t.Fatalf("unexpected body %q", result.Body)
	}
	if result.Model != "gpt-4.1" {
		t.Fatalf("unexpected model %q", result.Model)
	}
}
