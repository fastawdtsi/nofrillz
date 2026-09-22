package limiters

import (
	"testing"
	"time"
)

func TestIPLimiter_PerIPIsolationAndBurst(t *testing.T) {
	l := NewIPLimiter(2, 0, 10*time.Minute)

	ip1 := "1.2.3.4"
	ip2 := "5.6.7.8"

	if !l.Allow(ip1) || !l.Allow(ip1) {
		t.Fatalf("expected first two allows for ip1 to succeed")
	}
	if l.Allow(ip1) {
		t.Fatalf("expected third allow for ip1 to fail (burst exhausted)")
	}

	if !l.Allow(ip2) {
		t.Fatalf("expected allow for ip2 to succeed (independent bucket)")
	}
}
