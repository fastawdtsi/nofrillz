package limiters

import (
	// "net"
	// "net/http"
	// "strings"
	"sync"
	"time"
)

type IPLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipEntry

	capacity     int
	refillPerSec float64
	ttl          time.Duration
}

type ipEntry struct {
	b        *TokenBucket
	lastSeen time.Time
}

func NewIPLimiter(capacity int, refillPerSec float64, ttl time.Duration) *IPLimiter {
	l := &IPLimiter{
		buckets:      make(map[string]*ipEntry),
		capacity:     capacity,
		refillPerSec: refillPerSec,
		ttl:          ttl,
	}
	go l.cleanupLoop()
	return l
}

func (l *IPLimiter) Allow(ip string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	e := l.buckets[ip]
	if e == nil {
		e = &ipEntry{
			b:        NewTokenBucket(l.capacity, l.refillPerSec),
			lastSeen: now,
		}
		l.buckets[ip] = e
	} else {
		e.lastSeen = now
	}

	return e.b.Allow(1)
}

func (l *IPLimiter) cleanupLoop() {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()

	for range t.C {
		now := time.Now()
		l.mu.Lock()
		for ip, e := range l.buckets {
			if now.Sub(e.lastSeen) > l.ttl {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}
