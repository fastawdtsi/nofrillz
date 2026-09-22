package aigenerator

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"unicode"
	"unicode/utf8"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
	"nofrillz/internal/posts"
	"nofrillz/internal/users"
)

const RecentPostLimit = 15

var ErrRejected = errors.New("generated post rejected")

type GeneratedPost struct{ Body, Prompt, Model string }
type PostGenerator interface {
	GeneratePost(context.Context, *aiaccounts.AIAccount) (GeneratedPost, error)
}
type userReader interface {
	GetByID(context.Context, uint64) (*users.User, error)
}
type postReader interface {
	ListByUser(context.Context, uint64, uint64, uint64, int) ([]*posts.Post, error)
}
type AIToolsGenerator struct {
	tools aitools.Tools
	users userReader
	posts postReader
}

func NewAIToolsGenerator(tools aitools.Tools, users userReader, posts postReader) *AIToolsGenerator {
	return &AIToolsGenerator{tools: tools, users: users, posts: posts}
}

func (g *AIToolsGenerator) GeneratePost(ctx context.Context, account *aiaccounts.AIAccount) (GeneratedPost, error) {
	if account == nil {
		return GeneratedPost{}, fmt.Errorf("AI account is required")
	}
	user, err := g.users.GetByID(ctx, account.UserID)
	if err != nil {
		return GeneratedPost{}, fmt.Errorf("load persona profile: %w", err)
	}
	if user == nil || user.Deleted != nil || user.Blocked != nil {
		return GeneratedPost{}, fmt.Errorf("AI user is unavailable")
	}
	recent, err := g.posts.ListByUser(ctx, account.UserID, account.UserID, 0, RecentPostLimit)
	if err != nil {
		return GeneratedPost{}, fmt.Errorf("load recent posts: %w", err)
	}
	bodies := make([]string, 0, len(recent))
	for _, post := range recent {
		bodies = append(bodies, post.Body)
	}
	lengths := []string{"Just a few words, roughly 3–12 words.", "One natural sentence, roughly 10–30 words.", "A brief thought in one or two sentences, roughly 20–55 words.", "A short paragraph, roughly 40–100 words."}
	input := aitools.GeneratePostInput{
		PersonaName: strings.TrimSpace(user.FirstName + " " + user.LastName), Username: user.Username, Bio: user.About,
		Keywords: topicKeywords(account.Topic), Description: account.Description,
		SystemPrompt: account.SystemPrompt, StylePrompt: account.StylePrompt,
		RecentPosts: bodies, LengthHint: lengths[rand.Intn(len(lengths))],
	}
	result, err := g.tools.GeneratePostContent(ctx, input)
	generated := GeneratedPost{Body: strings.TrimSpace(result.Body), Prompt: result.Prompt, Model: result.Model}
	if err != nil {
		return generated, err
	}
	if err := ValidateCandidate(generated.Body, bodies); err != nil {
		return generated, err
	}
	return generated, nil
}

func ValidateCandidate(body string, recent []string) error {
	if strings.TrimSpace(body) == "" || !utf8.ValidString(body) || utf8.RuneCountInString(body) > 1200 {
		return fmt.Errorf("%w: empty, malformed, or longer than 1200 characters", ErrRejected)
	}
	lower := strings.ToLower(strings.TrimSpace(body))
	for _, prefix := range []string{"as an ai", "as a language model", "here are five", "let's dive into", "```", "{\""} {
		if strings.HasPrefix(lower, prefix) {
			return fmt.Errorf("%w: assistant or structured output", ErrRejected)
		}
	}
	for _, previous := range recent {
		if IsDuplicate(body, previous) {
			return fmt.Errorf("%w: repeats a recent post", ErrRejected)
		}
	}
	return nil
}

func normalizedWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
}

// Detect punctuation/case variants and obvious near-copies without a semantic index.
func IsDuplicate(a, b string) bool {
	aw, bw := normalizedWords(a), normalizedWords(b)
	if len(aw) == 0 || len(bw) == 0 {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	if strings.Join(aw, " ") == strings.Join(bw, " ") {
		return true
	}
	if len(aw) < 6 || len(bw) < 6 {
		return false
	}
	as, bs := map[string]bool{}, map[string]bool{}
	for _, w := range aw {
		as[w] = true
	}
	for _, w := range bw {
		bs[w] = true
	}
	intersection := 0
	for w := range as {
		if bs[w] {
			intersection++
		}
	}
	union := len(as) + len(bs) - intersection
	return float64(intersection)/float64(union) >= 0.85 && float64(min(len(aw), len(bw)))/float64(max(len(aw), len(bw))) >= 0.9
}

func topicKeywords(topic string) []string {
	var keywords []string
	for _, part := range strings.FieldsFunc(topic, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		if value := strings.TrimSpace(part); value != "" {
			keywords = append(keywords, value)
		}
	}
	return keywords
}
