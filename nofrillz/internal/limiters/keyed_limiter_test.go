package limiters

import (
	"testing"
	"time"
)

func TestKeyedLimiter_PerKeyIsolationAndBurst(t *testing.T) {
	l := NewKeyedLimiter(2, 0, 10*time.Minute)

	k1 := "user:1"
	k2 := "user:2"

	if !l.Allow(k1) || !l.Allow(k1) {
		t.Fatalf("expected first two allows for k1 to succeed")
	}
	if l.Allow(k1) {
		t.Fatalf("expected third allow for k1 to fail (burst exhausted)")
	}

	if !l.Allow(k2) {
		t.Fatalf("expected allow for k2 to succeed (independent bucket)")
	}
}
