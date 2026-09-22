package aiposter

import (
	"math/rand"
	"time"
)

func computeRetryAtWithRand(now time.Time, rng *rand.Rand, consecutiveFailures int) time.Time {
	baseDelay := 15 * time.Minute * time.Duration(1<<min(max(consecutiveFailures, 0), 4))
	jitterRange := baseDelay / 2
	jitter := time.Duration(rng.Int63n(int64(jitterRange) + 1))
	return now.UTC().Add(baseDelay + jitter)
}
