package aigenerator

import (
	"errors"
	"strings"
	"testing"
)

func TestCandidateValidation(t *testing.T) {
	for _, body := range []string{"", "   ", "As an AI, I enjoy soup.", "```json", strings.Repeat("x", 1201), string([]byte{0xff})} {
		if !errors.Is(ValidateCandidate(body, nil), ErrRejected) {
			t.Fatalf("invalid accepted %q", body)
		}
	}
	if err := ValidateCandidate("Soup again. Excellent.", nil); err != nil {
		t.Fatal(err)
	}
	if !IsDuplicate("The cat has decided my favorite chair now belongs to him today.", "The cat has decided my favorite chair now belongs to him tonight.") {
		t.Fatal("obvious near duplicate missed")
	}
	if IsDuplicate("I watered the basil.", "I repaired the radio.") {
		t.Fatal("distinct posts flagged")
	}
}
