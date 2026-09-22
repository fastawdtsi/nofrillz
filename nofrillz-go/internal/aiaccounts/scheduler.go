package aiaccounts

import (
	"fmt"
	"math/rand"
	"time"
)

// Schedule's zero value uses configured check intervals. Development timing
// must be explicitly enabled. Next retains legacy daily-rate compatibility.
type Schedule struct {
	Development bool
	MinInterval time.Duration
	MaxInterval time.Duration
}

func (s Schedule) Validate() error {
	if s.Development && (s.MinInterval < time.Minute || s.MaxInterval < s.MinInterval || s.MaxInterval > time.Hour) {
		return fmt.Errorf("development posting intervals must be between 60 and 3600 seconds, with max >= min")
	}
	return nil
}

func (s Schedule) Next(now time.Time, minPosts, maxPosts int) time.Time {
	if s.Development {
		return now.UTC().Add(s.MinInterval + time.Duration(rand.Int63n(int64(s.MaxInterval-s.MinInterval)+1)))
	}
	return ComputeNextGenerateAt(now, minPosts, maxPosts)
}

func (s Schedule) First(now time.Time) time.Time {
	if s.Development {
		return s.Next(now, 1, 1)
	}
	// Stagger new/enabled accounts instead of publishing a whole imported roster at once.
	return now.UTC().Add(time.Minute + time.Duration(rand.Int63n(int64(14*time.Minute))))
}

func ComputeNextGenerateAt(now time.Time, minPostsPerDay int, maxPostsPerDay int) time.Time {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return computeNextGenerateAtWithRand(now, minPostsPerDay, maxPostsPerDay, rng)
}

func computeNextGenerateAtWithRand(now time.Time, minPostsPerDay int, maxPostsPerDay int, rng *rand.Rand) time.Time {
	if minPostsPerDay < 1 {
		minPostsPerDay = 1
	}
	if maxPostsPerDay < minPostsPerDay {
		maxPostsPerDay = minPostsPerDay
	}

	targetPosts := minPostsPerDay
	if maxPostsPerDay > minPostsPerDay {
		targetPosts += rng.Intn(maxPostsPerDay - minPostsPerDay + 1)
	}

	baseInterval := (24 * time.Hour) / time.Duration(targetPosts)
	jitterRange := baseInterval / 3
	jitter := time.Duration(rng.Int63n(int64((2*jitterRange)+1))) - jitterRange
	next := now.UTC().Add(baseInterval + jitter)

	minNext := now.UTC().Add(5 * time.Minute)
	if next.Before(minNext) {
		return minNext
	}

	return next
}

// NextCheck schedules evaluations, never a quota of published posts.
func (s Schedule) NextCheck(now time.Time, intervalSeconds int) time.Time {
	if s.Development {
		return s.Next(now, 1, 1)
	}
	base := time.Duration(max(intervalSeconds, 300)) * time.Second
	return now.UTC().Add(base)
}
