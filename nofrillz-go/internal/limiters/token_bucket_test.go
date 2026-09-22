package limiters

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow_Consumes(t *testing.T) {
	b := NewTokenBucket(2, 0)

	if !b.Allow(1) {
		t.Fatalf("expected first Allow to succeed")
	}
	if !b.Allow(1) {
		t.Fatalf("expected second Allow to succeed")
	}
	if b.Allow(1) {
		t.Fatalf("expected third Allow to fail (bucket empty)")
	}
}

func TestTokenBucket_Allow_RefillsOverTime(t *testing.T) {
	b := NewTokenBucket(1, 5) // 5 tokens/sec

	if !b.Allow(1) {
		t.Fatalf("expected first Allow to succeed")
	}
	if b.Allow(1) {
		t.Fatalf("expected immediate second Allow to fail (no tokens yet)")
	}

	time.Sleep(250 * time.Millisecond) // ~1.25 tokens, capped at 1

	if !b.Allow(1) {
		t.Fatalf("expected Allow to succeed after refill")
	}
}
