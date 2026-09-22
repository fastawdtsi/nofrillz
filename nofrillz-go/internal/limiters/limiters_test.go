package limiters

import (
	"testing"

	"nofrillz/internal/config"
)

func TestNewLimiters_ReturnsSignupLimiters(t *testing.T) {
	ls := NewLimiters(nil)
	if ls == nil {
		t.Fatalf("expected non-nil Limiters")
	}
	if ls.SignupIP == nil {
		t.Fatalf("expected SignupIP to be non-nil")
	}
	if ls.SignupGlobal == nil {
		t.Fatalf("expected SignupGlobal to be non-nil")
	}
}

func TestNewLimiters_SignupIPDefaultBurstIsEnforced(t *testing.T) {
	ls := NewLimiters(nil)

	ip := "1.1.1.1"
	// NewLimiters currently hardcodes capacity=5 for SignupIP.
	for i := 0; i < 5; i++ {
		if !ls.SignupIP.Allow(ip) {
			t.Fatalf("expected allow #%d to succeed", i+1)
		}
	}
	if ls.SignupIP.Allow(ip) {
		t.Fatalf("expected allow #6 to fail (burst exhausted)")
	}
}

func TestNewLimiters_SignupGlobalDefaultBurstIsEnforced(t *testing.T) {
	ls := NewLimiters(nil)

	// NewLimiters currently hardcodes capacity=50 for SignupGlobal.
	for i := 0; i < 50; i++ {
		if !ls.SignupGlobal.Allow(1) {
			t.Fatalf("expected allow #%d to succeed", i+1)
		}
	}
	if ls.SignupGlobal.Allow(1) {
		t.Fatalf("expected allow #51 to fail (burst exhausted)")
	}
}

func TestNewLimiters_UsesProvidedConfigValues(t *testing.T) {
	cfg := &config.RateLimitsConfig{
		Signup: config.AuthRateLimits{
			IP: config.RateLimitConfig{
				Burst:        2,
				RefillPerSec: 0,
			},
			Global: config.RateLimitConfig{
				Burst:        3,
				RefillPerSec: 0,
			},
		},
		Login: config.AuthRateLimits{
			IP: config.RateLimitConfig{
				Burst:        2,
				RefillPerSec: 0,
			},
			Global: config.RateLimitConfig{
				Burst:        3,
				RefillPerSec: 0,
			},
		},
		Read: config.ReadRateLimits{
			IP: config.RateLimitConfig{
				Burst:        2,
				RefillPerSec: 0,
			},
			User: config.RateLimitConfig{
				Burst:        2,
				RefillPerSec: 0,
			},
			Global: config.RateLimitConfig{
				Burst:        3,
				RefillPerSec: 0,
			},
		},
		Write: config.WriteRateLimits{
			User: config.RateLimitConfig{
				Burst:        2,
				RefillPerSec: 0,
			},
			Global: config.RateLimitConfig{
				Burst:        3,
				RefillPerSec: 0,
			},
		},
		IPTTLSeconds:    600,
		KeyedTTLSeconds: 600,
	}

	ls := NewLimiters(cfg)

	ip := "9.9.9.9"
	if !ls.SignupIP.Allow(ip) || !ls.SignupIP.Allow(ip) {
		t.Fatalf("expected first two allows for SignupIP to succeed")
	}
	if ls.SignupIP.Allow(ip) {
		t.Fatalf("expected third allow for SignupIP to fail")
	}
}
