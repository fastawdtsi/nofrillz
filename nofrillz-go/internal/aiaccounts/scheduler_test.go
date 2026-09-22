package aiaccounts

import (
	"math/rand"
	"testing"
	"time"
)

func TestDailyScheduleRandomizesWithinRateBounds(t *testing.T) {
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(42))
	seen := map[time.Duration]bool{}
	for range 100 {
		delta := computeNextGenerateAtWithRand(now, 2, 6, rng).Sub(now)
		if delta < (24*time.Hour/6)*2/3 || delta > (24*time.Hour/2)*4/3 {
			t.Fatalf("interval outside configured bounds: %s", delta)
		}
		seen[delta] = true
	}
	if len(seen) < 90 {
		t.Fatalf("insufficient randomized timing: %d distinct", len(seen))
	}
}
func TestDevelopmentScheduleIsExplicitAndBounded(t *testing.T) {
	now := time.Now()
	dev := Schedule{Development: true, MinInterval: time.Minute, MaxInterval: 2 * time.Minute}
	if err := dev.Validate(); err != nil {
		t.Fatal(err)
	}
	for range 50 {
		d := dev.Next(now, 1, 1).Sub(now)
		if d < time.Minute || d > 2*time.Minute {
			t.Fatal(d)
		}
	}
	if d := (Schedule{}).Next(now, 1, 1).Sub(now); d < 16*time.Hour {
		t.Fatalf("production accelerated: %s", d)
	}
	for _, s := range []Schedule{{true, time.Second, time.Minute}, {true, 2 * time.Minute, time.Minute}, {true, time.Minute, 2 * time.Hour}} {
		if s.Validate() == nil {
			t.Fatalf("accepted invalid schedule: %+v", s)
		}
	}
}
