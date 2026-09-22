package aicontent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
)

const (
	matchingHistoryLimit = 12
	recentHistoryLimit   = 4
)

func normalizedContent(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

func boundedContent(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes)
}

type scoredHistory struct {
	content aitools.PriorContent
	score   float64
}

func historyKey(h aitools.PriorContent) uint64 {
	if h.ItemID != 0 {
		return h.ItemID
	}
	return h.PostID
}

var commonWords = func() map[string]bool {
	out := map[string]bool{}
	for _, word := range strings.Fields("a an and are as at be been but by can could did do does for from had has have how i if in into is it its may more not of on or our so some than that the their them there these they this those to was we were what when which who will with would you your") {
		out[word] = true
	}
	return out
}()

func contentTerms(text string) map[string]bool {
	terms := map[string]bool{}
	for _, word := range strings.Fields(normalizedContent(text)) {
		if !commonWords[word] {
			terms[word] = true
		}
	}
	return terms
}

// Similarity only retrieves candidates. It NEVER decides that a changed fact is
// a duplicate; an editorial comparison distinguishes updates from repetition.
func termOverlap(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	shared := 0
	for word := range a {
		if b[word] {
			shared++
		}
	}
	return 2 * float64(shared) / float64(len(a)+len(b))
}

func addMatch(matches []scoredHistory, candidate scoredHistory) []scoredHistory {
	if candidate.score <= 0 {
		return matches
	}
	for i, prior := range matches {
		if historyKey(prior.content) == historyKey(candidate.content) {
			if prior.score >= candidate.score {
				return matches
			}
			matches = append(matches[:i], matches[i+1:]...)
			break
		}
	}
	matches = append(matches, candidate)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].content.PostID > matches[j].content.PostID
		}
		return matches[i].score > matches[j].score
	})
	if len(matches) > matchingHistoryLimit {
		matches = matches[:matchingHistoryLimit]
	}
	return matches
}

// Scan durable account history, including legacy and soft-deleted posts, across
// ALL models and dates. Memory and external prompt size stay bounded. Excluding
// this logical item is essential: sibling model variants intentionally agree.
func (s *Service) priorContent(ctx context.Context, a *aiaccounts.AIAccount, item *Item, body string) ([]aitools.PriorContent, *aitools.PriorContent, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT p.id,COALESCE(v.content_item_id,0),p.body,COALESCE(i.context,'')
FROM posts p LEFT JOIN ai_content_variants v ON v.post_id=p.id
LEFT JOIN ai_content_items i ON i.id=v.content_item_id
WHERE p.user_id=? AND (v.content_item_id IS NULL OR v.content_item_id<>?) ORDER BY p.id DESC`, a.UserID, item.ID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	bodyKey, contextKey := normalizedContent(body), normalizedContent(item.Context)
	bodyTerms, contextTerms := contentTerms(body), contentTerms(item.Context)
	recent := []aitools.PriorContent{}
	matches := []scoredHistory{}
	recentSeen := map[uint64]bool{}
	for rows.Next() {
		var prior aitools.PriorContent
		if err = rows.Scan(&prior.PostID, &prior.ItemID, &prior.Body, &prior.Context); err != nil {
			return nil, nil, err
		}
		if bodyKey == normalizedContent(prior.Body) || (a.ContentMode == "research" && contextKey != "" && contextKey == normalizedContent(prior.Context)) {
			return nil, &prior, nil
		}
		priorBodyTerms, priorContextTerms := contentTerms(prior.Body), contentTerms(prior.Context)
		score := max(termOverlap(bodyTerms, priorBodyTerms), termOverlap(bodyTerms, priorContextTerms), termOverlap(contextTerms, priorContextTerms), termOverlap(contextTerms, priorBodyTerms))
		prior.Body = boundedContent(prior.Body, 1200)
		prior.Context = boundedContent(prior.Context, 1500)
		matches = addMatch(matches, scoredHistory{prior, score})
		if len(recent) < recentHistoryLimit && !recentSeen[historyKey(prior)] {
			recentSeen[historyKey(prior)] = true
			recent = append(recent, prior)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	selected := []aitools.PriorContent{}
	seen := map[uint64]bool{}
	for _, candidate := range matches {
		selected = append(selected, candidate.content)
		seen[historyKey(candidate.content)] = true
	}
	for _, candidate := range recent {
		if !seen[historyKey(candidate)] {
			selected = append(selected, candidate)
		}
	}
	return selected, nil, nil
}

// Every review fails closed on provider errors, uncertain decisions or malformed
// output. A duplicate ID must point to an actual entry sent to the reviewer.
func parseNoveltyDecision(body string, history []aitools.PriorContent) (*aitools.PriorContent, error) {
	var decision struct {
		Decision string `json:"decision"`
		PostID   string `json:"post_id"`
	}
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if len(body) > 2048 || decoder.Decode(&decision) != nil {
		return nil, fmt.Errorf("invalid novelty review response")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("invalid novelty review response")
	}
	if decision.Decision == "new" && decision.PostID == "" {
		return nil, nil
	}
	if decision.Decision == "duplicate" {
		for _, prior := range history {
			if strconv.FormatUint(prior.PostID, 10) == decision.PostID {
				return &prior, nil
			}
		}
	}
	return nil, fmt.Errorf("novelty review did not establish new content")
}

func (s *Service) review(ctx context.Context, a *aiaccounts.AIAccount, writer aitools.Option, unavailable map[string]bool, input aitools.GeneratePostInput) (aitools.GeneratedPost, aitools.Option, error) {
	reviewer, ok := s.Registry.Get(a.DefaultModelOption)
	if !ok || !reviewer.Available || unavailable[reviewer.ID] {
		reviewer = writer
	}
	result, err := reviewer.Tools.GeneratePostContent(ctx, input)
	if err != nil && reviewer.ID != writer.ID {
		unavailable[reviewer.ID] = true
		s.log(a, "default reviewer unavailable; using writer", map[string]any{"review_option": reviewer.ID, "option_id": writer.ID})
		reviewer = writer
		result, err = reviewer.Tools.GeneratePostContent(ctx, input)
	}
	return result, reviewer, err
}

func (s *Service) checkNovelty(ctx context.Context, a *aiaccounts.AIAccount, item *Item, option aitools.Option, unavailable map[string]bool, body string) (*aitools.PriorContent, error) {
	history, duplicate, err := s.priorContent(ctx, a, item, body)
	if err != nil || duplicate != nil || len(history) == 0 {
		return duplicate, err
	}
	input := aitools.GeneratePostInput{Novelty: &aitools.NoveltyInput{Candidate: body, Context: boundedContent(item.Context, 5000), History: history}}
	result, reviewer, err := s.review(ctx, a, option, unavailable, input)
	if err != nil {
		return nil, fmt.Errorf("novelty review failed: %w", err)
	}
	duplicate, err = parseNoveltyDecision(result.Body, history)
	s.log(a, "content novelty reviewed", map[string]any{"content_item_id": item.ID, "option_id": option.ID, "review_option": reviewer.ID, "history_candidates": len(history), "duplicate": duplicate != nil, "approved": err == nil && duplicate == nil})
	return duplicate, err
}

func (s *Service) saveDuplicate(ctx context.Context, a *aiaccounts.AIAccount, item *Item, optionID string, prior *aitools.PriorContent) error {
	return s.withClaim(ctx, a, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE ai_content_variants SET status='duplicate',error=? WHERE content_item_id=? AND option_id=? AND post_id IS NULL`, fmt.Sprintf("repeats previously published post %d (content item %d)", prior.PostID, prior.ItemID), item.ID, optionID)
		return err
	})
}
