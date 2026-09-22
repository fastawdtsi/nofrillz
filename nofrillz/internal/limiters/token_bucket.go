package limiters

import (
	"sync"
	"time"
	// "nofrillz/internal/limiters"
)

type TokenBucket struct {
	mu        sync.Mutex
	capacity  float64
	tokens    float64
	refillPer float64
	last      time.Time
}

func NewTokenBucket(capacity int, refillPerSecond float64) *TokenBucket {
	now := time.Now()
	return &TokenBucket{
		capacity:  float64(capacity),
		tokens:    float64(capacity),
		refillPer: refillPerSecond,
		last:      now,
	}
}

func (b *TokenBucket) Allow(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now

	b.tokens += elapsed * b.refillPer
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}

	if b.tokens < n {
		return false
	}

	b.tokens -= n
	return true
}
