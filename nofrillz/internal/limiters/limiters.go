package limiters

import (
	"time"

	"nofrillz/internal/config"
)

type Limiters struct {
	SignupIP     *IPLimiter
	SignupGlobal *TokenBucket

	LoginIP     *IPLimiter
	LoginGlobal *TokenBucket

	ReadIP     *IPLimiter
	ReadGlobal *TokenBucket

	ReadUser  *KeyedLimiter
	WriteUser *KeyedLimiter

	WriteGlobal *TokenBucket
}

func NewLimiters(config *config.RateLimitsConfig) *Limiters {
	if config == nil {
		config = defaultRateLimitsConfig()
	}

	ipTTL := time.Duration(config.IPTTLSeconds) * time.Second
	keyedTTL := time.Duration(config.KeyedTTLSeconds) * time.Second
	if ipTTL <= 0 {
		ipTTL = 10 * time.Minute
	}
	if keyedTTL <= 0 {
		keyedTTL = 10 * time.Minute
	}

	signupIP := NewIPLimiter(config.Signup.IP.Burst, config.Signup.IP.RefillPerSec, ipTTL)
	signupGlobal := NewTokenBucket(config.Signup.Global.Burst, config.Signup.Global.RefillPerSec)

	loginIP := NewIPLimiter(config.Login.IP.Burst, config.Login.IP.RefillPerSec, ipTTL)
	loginGlobal := NewTokenBucket(config.Login.Global.Burst, config.Login.Global.RefillPerSec)

	readIP := NewIPLimiter(config.Read.IP.Burst, config.Read.IP.RefillPerSec, ipTTL)
	readUser := NewKeyedLimiter(config.Read.User.Burst, config.Read.User.RefillPerSec, keyedTTL)
	readGlobal := NewTokenBucket(config.Read.Global.Burst, config.Read.Global.RefillPerSec)

	writeUser := NewKeyedLimiter(config.Write.User.Burst, config.Write.User.RefillPerSec, keyedTTL)
	writeGlobal := NewTokenBucket(config.Write.Global.Burst, config.Write.Global.RefillPerSec)

	return &Limiters{
		SignupIP:     signupIP,
		SignupGlobal: signupGlobal,
		LoginIP:      loginIP,
		LoginGlobal:  loginGlobal,
		ReadIP:       readIP,
		ReadUser:     readUser,
		ReadGlobal:   readGlobal,
		WriteUser:    writeUser,
		WriteGlobal:  writeGlobal,
	}
}

func defaultRateLimitsConfig() *config.RateLimitsConfig {
	return &config.RateLimitsConfig{
		Signup: config.AuthRateLimits{
			IP: config.RateLimitConfig{
				Burst:        5,
				RefillPerSec: 0.2,
			},
			Global: config.RateLimitConfig{
				Burst:        50,
				RefillPerSec: 10,
			},
		},
		Login: config.AuthRateLimits{
			IP: config.RateLimitConfig{
				Burst:        20,
				RefillPerSec: 1,
			},
			Global: config.RateLimitConfig{
				Burst:        200,
				RefillPerSec: 50,
			},
		},
		Read: config.ReadRateLimits{
			IP: config.RateLimitConfig{
				Burst:        120,
				RefillPerSec: 10,
			},
			User: config.RateLimitConfig{
				Burst:        300,
				RefillPerSec: 20,
			},
			Global: config.RateLimitConfig{
				Burst:        2000,
				RefillPerSec: 500,
			},
		},
		Write: config.WriteRateLimits{
			User: config.RateLimitConfig{
				Burst:        10,
				RefillPerSec: 0.2,
			},
			Global: config.RateLimitConfig{
				Burst:        200,
				RefillPerSec: 50,
			},
		},
		IPTTLSeconds:    int((10 * time.Minute).Seconds()),
		KeyedTTLSeconds: int((10 * time.Minute).Seconds()),
	}
}
