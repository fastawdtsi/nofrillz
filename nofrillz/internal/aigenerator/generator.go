package aigenerator

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrRejected = errors.New("generated post rejected")

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
	// One-word substitutions in a longer sentence are usually recycled posts.
	if len(aw) == len(bw) && len(aw) >= 10 {
		same := 0
		for i := range aw {
			if aw[i] == bw[i] {
				same++
			}
		}
		if float64(same)/float64(len(aw)) >= 0.9 {
			return true
		}
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
	return float64(intersection)/float64(union) >= 0.9 && float64(min(len(aw), len(bw)))/float64(max(len(aw), len(bw))) >= 0.9
}
