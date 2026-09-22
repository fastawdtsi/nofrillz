package aitools

import (
	"context"
	"fmt"
	"strings"
)

type MockTools struct{}

func NewMockTools() *MockTools {
	return &MockTools{}
}

func (t *MockTools) GeneratePostContent(ctx context.Context, input GeneratePostInput) (GeneratedPost, error) {
	_ = ctx

	keywords := normalizeKeywords(input.Keywords)
	subject := "writing"
	if len(keywords) > 0 {
		subject = keywords[0]
	}

	description := strings.TrimSpace(input.Description)
	body := fmt.Sprintf("On %s: simple tools usually win because they reduce friction.", subject)
	if description != "" {
		body = fmt.Sprintf("On %s: %s", subject, description)
	}

	return GeneratedPost{
		Body:   body,
		Prompt: BuildPostPrompt(input),
		Model:  "mock-generator/v1",
	}, nil
}
