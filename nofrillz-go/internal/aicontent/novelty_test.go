package aicontent

import (
	"fmt"
	"strings"
	"testing"

	"nofrillz/internal/aitools"
)

func TestNoveltyDecisionRequiresKnownReferenceAndStrictResponse(t *testing.T) {
	history := []aitools.PriorContent{{PostID: 123, Body: "An earlier joke"}}
	for _, input := range []string{`{"decision":"uncertain"}`, `{"decision":"duplicate","post_id":"999"}`, `{"decision":"duplicate"}`, `{"decision":"new","post_id":"123"}`, `{"decision":"new","extra":true}`, `{"decision":"new"} trailing`, `{"decision":"new"} {}`, "__APPROVED__", "", strings.Repeat(" ", 2049)} {
		if _, err := parseNoveltyDecision(input, history); err == nil {
			t.Fatalf("invalid review accepted: %q", input)
		}
	}
	if got, err := parseNoveltyDecision(`{"decision":"duplicate","post_id":"123"}`, history); err != nil || got == nil || got.PostID != 123 {
		t.Fatalf("duplicate reference lost: %+v %v", got, err)
	}
	if got, err := parseNoveltyDecision(`{"decision":"new"}`, history); err != nil || got != nil {
		t.Fatalf("new content rejected: %+v %v", got, err)
	}
}

func TestHistoryMatchingIsBoundedAndGroupsModelVariants(t *testing.T) {
	var matches []scoredHistory
	for i := 1; i <= 100; i++ {
		matches = addMatch(matches, scoredHistory{aitools.PriorContent{PostID: uint64(i), ItemID: uint64(i / 2), Body: fmt.Sprint(i)}, float64(i)})
	}
	if len(matches) != matchingHistoryLimit || matches[0].content.PostID != 100 {
		t.Fatal("did not retain bounded best history matches")
	}
	seen := map[uint64]bool{}
	for _, match := range matches {
		if seen[historyKey(match.content)] {
			t.Fatal("variants crowded out distinct historical items")
		}
		seen[historyKey(match.content)] = true
	}
}

func TestNormalizedContentRetainsChangedFacts(t *testing.T) {
	if normalizedContent("THE Orange—ran out of juice!") != normalizedContent("the orange ran out of juice.") {
		t.Fatal("case and punctuation changes escaped normalization")
	}
	if normalizedContent("The score is 3–1.") == normalizedContent("The score is 4–1.") {
		t.Fatal("changed facts collapsed into exact match")
	}
}
