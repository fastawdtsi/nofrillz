package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Anthropic struct {
	key, baseURL, model string
	client              *http.Client
}
type ChatCompletions struct {
	key, baseURL, model string
	client              *http.Client
}

func NewAnthropic(key, baseURL, model string) *Anthropic {
	return &Anthropic{key, strings.TrimRight(baseURL, "/"), model, &http.Client{Timeout: 30 * time.Second}}
}
func NewChatCompletions(key, baseURL, model string) *ChatCompletions {
	return &ChatCompletions{key, strings.TrimRight(baseURL, "/"), model, &http.Client{Timeout: 30 * time.Second}}
}
func providerRequest(ctx context.Context, client *http.Client, url string, payload any, headers map[string]string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("invalid provider endpoint")
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provider request failed")
	}
	defer response.Body.Close()
	result, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("provider response read failed")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("provider HTTP %d: %s", response.StatusCode, safeErrorCode(result))
	}
	return result, nil
}
func (p *Anthropic) GeneratePostContent(ctx context.Context, in GeneratePostInput) (GeneratedPost, error) {
	prompt := BuildPostPrompt(in)
	raw, err := providerRequest(ctx, p.client, p.baseURL+"/messages", map[string]any{"model": p.model, "max_tokens": 512, "system": Instructions(in), "messages": []map[string]string{{"role": "user", "content": prompt}}}, map[string]string{"x-api-key": p.key, "anthropic-version": "2023-06-01"})
	if err != nil {
		return GeneratedPost{}, err
	}
	var out struct {
		Model   string                        `json:"model"`
		Stop    string                        `json:"stop_reason"`
		Content []struct{ Type, Text string } `json:"content"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Stop != "end_turn" {
		return GeneratedPost{}, fmt.Errorf("incomplete or invalid Anthropic response")
	}
	var texts []string
	for _, part := range out.Content {
		if part.Type == "text" {
			texts = append(texts, part.Text)
		}
	}
	body := strings.TrimSpace(strings.Join(texts, "\n"))
	if body == "" {
		return GeneratedPost{}, ErrEmptyGeneratedBody
	}
	return GeneratedPost{Body: body, Prompt: prompt, Model: valueOr(out.Model, p.model)}, nil
}
func (p *ChatCompletions) GeneratePostContent(ctx context.Context, in GeneratePostInput) (GeneratedPost, error) {
	prompt := BuildPostPrompt(in)
	raw, err := providerRequest(ctx, p.client, p.baseURL+"/chat/completions", map[string]any{"model": p.model, "max_tokens": 512, "stream": false, "messages": []map[string]string{{"role": "system", "content": Instructions(in)}, {"role": "user", "content": prompt}}}, map[string]string{"Authorization": "Bearer " + p.key})
	if err != nil {
		return GeneratedPost{}, err
	}
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &out) != nil || len(out.Choices) != 1 || out.Choices[0].Finish != "stop" || out.Choices[0].Message.Refusal != "" {
		return GeneratedPost{}, fmt.Errorf("incomplete or invalid chat response")
	}
	body := strings.TrimSpace(out.Choices[0].Message.Content)
	if body == "" {
		return GeneratedPost{}, ErrEmptyGeneratedBody
	}
	return GeneratedPost{Body: body, Prompt: prompt, Model: valueOr(out.Model, p.model)}, nil
}
