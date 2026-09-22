package limiters

import (
	"sync"
	"time"
)

type KeyedLimiter struct {
	mu      sync.Mutex
	buckets map[string]*keyedEntry

	capacity     int
	refillPerSec float64
	ttl          time.Duration
}

type keyedEntry struct {
	b        *TokenBucket
	lastSeen time.Time
}

func NewKeyedLimiter(capacity int, refillPerSec float64, ttl time.Duration) *KeyedLimiter {
	l := &KeyedLimiter{
		buckets:      make(map[string]*keyedEntry),
		capacity:     capacity,
		refillPerSec: refillPerSec,
		ttl:          ttl,
	}
	go l.cleanupLoop()
	return l
}

func (l *KeyedLimiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	e := l.buckets[key]
	if e == nil {
		e = &keyedEntry{
			b:        NewTokenBucket(l.capacity, l.refillPerSec),
			lastSeen: now,
		}
		l.buckets[key] = e
	} else {
		e.lastSeen = now
	}

	return e.b.Allow(1)
}

func (l *KeyedLimiter) cleanupLoop() {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()

	for range t.C {
		now := time.Now()
		l.mu.Lock()
		for k, e := range l.buckets {
			if now.Sub(e.lastSeen) > l.ttl {
				delete(l.buckets, k)
			}
		}
		l.mu.Unlock()
	}
}
