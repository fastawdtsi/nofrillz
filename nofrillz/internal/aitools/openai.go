package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nofrillz/internal/config"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

var (
	ErrMissingAPIKey       = errors.New("openai api key is required")
	ErrEmptyGeneratedBody  = errors.New("openai generated an empty post body")
	ErrUnsupportedProvider = errors.New("unsupported ai tools provider")
)

type OpenAI struct {
	httpClient     *http.Client
	apiKey         string
	accountID      string
	organizationID string
	projectID      string
	baseURL        string
	model          string
}

type openAIResponsesRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	Instructions    string `json:"instructions"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	Store           bool   `json:"store"`
}

type openAIResponsesResponse struct {
	Status     string `json:"status"`
	Model      string `json:"model"`
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type openAIErrorResponse struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewFromConfig(cfg *config.AIToolsConfig) (Tools, error) {
	if cfg == nil {
		return NewMockTools(), nil
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "mock":
		return NewMockTools(), nil
	case "openai":
		return NewOpenAI(cfg.OpenAIConfig())
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, cfg.Provider)
	}
}

func NewOpenAI(cfg *config.OpenAIConfig) (*OpenAI, error) {
	if cfg == nil {
		return nil, ErrMissingAPIKey
	}

	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4.1"
	}

	return &OpenAI{
		httpClient:     &http.Client{Timeout: timeout},
		apiKey:         apiKey,
		accountID:      strings.TrimSpace(cfg.AccountID),
		organizationID: strings.TrimSpace(cfg.OrganizationID),
		projectID:      strings.TrimSpace(cfg.ProjectID),
		baseURL:        strings.TrimRight(baseURL, "/"),
		model:          model,
	}, nil
}

func (o *OpenAI) GeneratePostContent(ctx context.Context, input GeneratePostInput) (GeneratedPost, error) {
	prompt := BuildPostPrompt(input)
	requestBody := openAIResponsesRequest{
		Model:           o.model,
		Input:           prompt,
		Instructions:    PostInstructions,
		MaxOutputTokens: 256,
		Store:           false,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return GeneratedPost{}, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return GeneratedPost{}, fmt.Errorf("build openai request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if organizationID := o.organizationHeader(); organizationID != "" {
		req.Header.Set("OpenAI-Organization", organizationID)
	}
	if o.projectID != "" {
		req.Header.Set("OpenAI-Project", o.projectID)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return GeneratedPost{}, fmt.Errorf("request openai response: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return GeneratedPost{}, fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GeneratedPost{}, fmt.Errorf("openai responses api returned %s: %s", resp.Status, safeErrorCode(responseBody))
	}

	decoded := openAIResponsesResponse{}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return GeneratedPost{}, fmt.Errorf("decode openai response: %w", err)
	}

	if decoded.Status != "" && decoded.Status != "completed" {
		return GeneratedPost{}, fmt.Errorf("openai response was not completed")
	}
	for _, output := range decoded.Output {
		for _, part := range output.Content {
			if part.Type == "refusal" || part.Refusal != "" {
				return GeneratedPost{}, fmt.Errorf("openai declined this generation")
			}
		}
	}
	bodyText := strings.TrimSpace(extractOpenAIOutputText(decoded))
	if bodyText == "" {
		return GeneratedPost{}, ErrEmptyGeneratedBody
	}

	model := strings.TrimSpace(decoded.Model)
	if model == "" {
		model = o.model
	}

	return GeneratedPost{
		Body:   bodyText,
		Prompt: prompt,
		Model:  model,
	}, nil
}

func (o *OpenAI) organizationHeader() string {
	if o.organizationID != "" {
		return o.organizationID
	}

	return o.accountID
}

func extractOpenAIOutputText(response openAIResponsesResponse) string {
	if text := strings.TrimSpace(response.OutputText); text != "" {
		return text
	}

	parts := make([]string, 0)
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type != "output_text" {
				continue
			}
			text := strings.TrimSpace(content.Text)
			if text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// Error messages may echo credentials or request content. Persist only status
// and a bounded machine-readable code, never provider response bodies.
func safeErrorCode(body []byte) string {
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return "request_failed"
	}
	code := payload.Error.Code
	if len(code) == 0 || len(code) > 64 {
		return "request_failed"
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return "request_failed"
		}
	}
	return code
}
