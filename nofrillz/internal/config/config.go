package config

import (
	"fmt"
	"nofrillz/internal/aiaccounts"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Viper    IViper
	Filename string
}

type ApiConfig struct {
	Address string
}

type MySQLConfig struct {
	DSN string
}

type RedisConfig struct {
	Address  string
	Password string
}

type IDGeneratorConfig struct {
	Region uint64
	Node   uint64
}

type SessionConfig struct {
	AccessTTLSeconds  int
	RefreshTTLSeconds int
	JWTSecret         string
}

type LogConfig struct {
	Level string
}

type AdminConfig struct {
	APIKey string
}

type APNSConfig struct {
	Enabled        bool
	Topic          string
	Environment    string
	AuthKeyPath    string
	AuthKeyBase64  string
	KeyID          string
	TeamID         string
	TimeoutSeconds int
}

type AIPosterConfig struct {
	DevelopmentMode               bool
	DevelopmentMinIntervalSeconds int
	DevelopmentMaxIntervalSeconds int
	PollIntervalSeconds           int
	BatchSize                     int
	StaleRunningAfterMinutes      int
}

type AIToolsConfig struct {
	Provider string
	OpenAI   OpenAIConfig
}

type OpenAIConfig struct {
	AccountID      string
	OrganizationID string
	ProjectID      string
	APIKey         string
	BaseURL        string
	Model          string
	TimeoutSeconds int
}

type BaseURLsConfig struct {
	Avatar string
}

type RateLimitConfig struct {
	Burst        int
	RefillPerSec float64
}

type AuthRateLimits struct {
	IP     RateLimitConfig
	Global RateLimitConfig
}

type ReadRateLimits struct {
	IP     RateLimitConfig
	User   RateLimitConfig
	Global RateLimitConfig
}

type WriteRateLimits struct {
	User   RateLimitConfig
	Global RateLimitConfig
}

type RateLimitsConfig struct {
	Signup AuthRateLimits
	Login  AuthRateLimits
	Read   ReadRateLimits
	Write  WriteRateLimits

	IPTTLSeconds    int
	KeyedTTLSeconds int
}

func NewConfig(filename string) (*Config, error) {
	viperCfg := viper.New()

	viperCfg.SetDefault("api.address", "127.0.0.1:8080")

	viperCfg.SetDefault("mysql.dsn", "")

	viperCfg.SetDefault("redis.address", "127.0.0.1:6379")
	viperCfg.SetDefault("redis.password", "")
	viperCfg.SetDefault("session.access_ttl_seconds", 900)
	viperCfg.SetDefault("session.refresh_ttl_seconds", 31536000)
	viperCfg.SetDefault("session.ttl_seconds", 900)
	viperCfg.SetDefault("session.jwt_secret", "change-me")
	viperCfg.SetDefault("log.level", "info")
	viperCfg.SetDefault("admin.api_key", "")
	viperCfg.SetDefault("apns.enabled", false)
	viperCfg.SetDefault("apns.topic", "")
	viperCfg.SetDefault("apns.environment", "sandbox")
	viperCfg.SetDefault("apns.auth_key_path", "")
	viperCfg.SetDefault("apns.auth_key_base64", "")
	viperCfg.SetDefault("apns.key_id", "")
	viperCfg.SetDefault("apns.team_id", "")
	viperCfg.SetDefault("apns.timeout_seconds", 10)
	viperCfg.SetDefault("ai_poster.development_mode", false)
	viperCfg.SetDefault("ai_poster.development_min_interval_seconds", 60)
	viperCfg.SetDefault("ai_poster.development_max_interval_seconds", 180)
	viperCfg.SetDefault("ai_poster.poll_interval_seconds", 60)
	viperCfg.SetDefault("ai_poster.batch_size", 10)
	viperCfg.SetDefault("ai_poster.stale_running_after_minutes", 15)
	viperCfg.SetDefault("ai_tools.provider", "mock")
	viperCfg.SetDefault("ai_tools.openai.account_id", "")
	viperCfg.SetDefault("ai_tools.openai.organization_id", "")
	viperCfg.SetDefault("ai_tools.openai.project_id", "")
	viperCfg.SetDefault("ai_tools.openai.api_key", "")
	viperCfg.SetDefault("ai_tools.openai.base_url", "https://api.openai.com/v1")
	viperCfg.SetDefault("ai_tools.openai.model", "gpt-4.1")
	viperCfg.SetDefault("ai_tools.openai.timeout_seconds", 30)
	viperCfg.SetDefault("base_urls.avatar", "http:10.10.0.230:3000")

	// IMPORTANT: Do NOT default id_generator.region/node. These must be explicitly set
	// to avoid accidental (region,node) collisions across processes.

	// Sensible, safer defaults
	viperCfg.SetDefault("rate_limits.signup.ip.burst", 5)
	viperCfg.SetDefault("rate_limits.signup.ip.refill_per_sec", 0.2) // 1 per 5s per IP
	viperCfg.SetDefault("rate_limits.signup.global.burst", 50)
	viperCfg.SetDefault("rate_limits.signup.global.refill_per_sec", 10.0)

	viperCfg.SetDefault("rate_limits.login.ip.burst", 20)
	viperCfg.SetDefault("rate_limits.login.ip.refill_per_sec", 1.0) // 1/sec per IP
	viperCfg.SetDefault("rate_limits.login.global.burst", 200)
	viperCfg.SetDefault("rate_limits.login.global.refill_per_sec", 50.0)

	viperCfg.SetDefault("rate_limits.read.ip.burst", 120)
	viperCfg.SetDefault("rate_limits.read.ip.refill_per_sec", 10.0)
	viperCfg.SetDefault("rate_limits.read.user.burst", 300)
	viperCfg.SetDefault("rate_limits.read.user.refill_per_sec", 20.0)
	viperCfg.SetDefault("rate_limits.read.global.burst", 2000)
	viperCfg.SetDefault("rate_limits.read.global.refill_per_sec", 500.0)

	viperCfg.SetDefault("rate_limits.write.user.burst", 10)
	viperCfg.SetDefault("rate_limits.write.user.refill_per_sec", 0.2)
	viperCfg.SetDefault("rate_limits.write.global.burst", 200)
	viperCfg.SetDefault("rate_limits.write.global.refill_per_sec", 50.0)

	viperCfg.SetDefault("rate_limits.ip_ttl_seconds", 600)
	viperCfg.SetDefault("rate_limits.keyed_ttl_seconds", 600)

	viperCfg.SetEnvPrefix("nofrillz")
	viperCfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperCfg.AutomaticEnv()

	if filename != "" {
		viperCfg.SetConfigFile(filename)
		if err := viperCfg.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	if !viperCfg.IsSet("id_generator.region") || !viperCfg.IsSet("id_generator.node") {
		return nil, fmt.Errorf("id_generator.region and id_generator.node must be set (via config or env)")
	}

	syncViper := NewSyncViper(viperCfg)

	cfg := &Config{
		Viper:    syncViper,
		Filename: filename,
	}
	if err := cfg.AIPosterConfig().Schedule().Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) ApiConfig() *ApiConfig {
	return &ApiConfig{
		Address: c.Viper.GetString("api.address"),
	}
}

func (c *Config) MySQLConfig() *MySQLConfig {
	return &MySQLConfig{
		DSN: c.Viper.GetString("mysql.dsn"),
	}
}

func (c *Config) RedisConfig() *RedisConfig {
	return &RedisConfig{
		Address:  c.Viper.GetString("redis.address"),
		Password: c.Viper.GetString("redis.password"),
	}
}

func (c *Config) IDGeneratorConfig() *IDGeneratorConfig {
	return &IDGeneratorConfig{
		Region: c.Viper.GetUint64("id_generator.region"),
		Node:   c.Viper.GetUint64("id_generator.node"),
	}
}

func (c *Config) SessionConfig() *SessionConfig {
	accessTTL := c.Viper.GetInt("session.access_ttl_seconds")
	if accessTTL <= 0 {
		accessTTL = c.Viper.GetInt("session.ttl_seconds")
	}

	return &SessionConfig{
		AccessTTLSeconds:  accessTTL,
		RefreshTTLSeconds: c.Viper.GetInt("session.refresh_ttl_seconds"),
		JWTSecret:         c.Viper.GetString("session.jwt_secret"),
	}
}

func (c *Config) LogConfig() *LogConfig {
	return &LogConfig{
		Level: c.Viper.GetString("log.level"),
	}
}

func (c *Config) AdminConfig() *AdminConfig {
	return &AdminConfig{
		APIKey: c.Viper.GetString("admin.api_key"),
	}
}

func (c *Config) APNSConfig() *APNSConfig {
	return &APNSConfig{
		Enabled:        c.Viper.GetBool("apns.enabled"),
		Topic:          c.Viper.GetString("apns.topic"),
		Environment:    c.Viper.GetString("apns.environment"),
		AuthKeyPath:    c.Viper.GetString("apns.auth_key_path"),
		AuthKeyBase64:  c.Viper.GetString("apns.auth_key_base64"),
		KeyID:          c.Viper.GetString("apns.key_id"),
		TeamID:         c.Viper.GetString("apns.team_id"),
		TimeoutSeconds: c.Viper.GetInt("apns.timeout_seconds"),
	}
}

func (c *Config) AIPosterConfig() *AIPosterConfig {
	return &AIPosterConfig{
		DevelopmentMode:               c.Viper.GetBool("ai_poster.development_mode"),
		DevelopmentMinIntervalSeconds: c.Viper.GetInt("ai_poster.development_min_interval_seconds"),
		DevelopmentMaxIntervalSeconds: c.Viper.GetInt("ai_poster.development_max_interval_seconds"),
		PollIntervalSeconds:           c.Viper.GetInt("ai_poster.poll_interval_seconds"),
		BatchSize:                     c.Viper.GetInt("ai_poster.batch_size"),
		StaleRunningAfterMinutes:      c.Viper.GetInt("ai_poster.stale_running_after_minutes"),
	}
}

func (c *Config) AIToolsConfig() *AIToolsConfig {
	return &AIToolsConfig{
		Provider: c.Viper.GetString("ai_tools.provider"),
		OpenAI: OpenAIConfig{
			AccountID:      c.Viper.GetString("ai_tools.openai.account_id"),
			OrganizationID: c.Viper.GetString("ai_tools.openai.organization_id"),
			ProjectID:      c.Viper.GetString("ai_tools.openai.project_id"),
			APIKey:         c.Viper.GetString("ai_tools.openai.api_key"),
			BaseURL:        c.Viper.GetString("ai_tools.openai.base_url"),
			Model:          c.Viper.GetString("ai_tools.openai.model"),
			TimeoutSeconds: c.Viper.GetInt("ai_tools.openai.timeout_seconds"),
		},
	}
}

func (c *AIToolsConfig) OpenAIConfig() *OpenAIConfig {
	if c == nil {
		return nil
	}

	cfg := c.OpenAI
	return &cfg
}

func (c *Config) BaseURLsConfig() *BaseURLsConfig {
	return &BaseURLsConfig{
		Avatar: c.Viper.GetString("base_urls.avatar"),
	}
}

func (c *Config) RateLimitsConfig() *RateLimitsConfig {
	cfg := &RateLimitsConfig{
		Signup: AuthRateLimits{
			IP: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.signup.ip.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.signup.ip.refill_per_sec"),
			},
			Global: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.signup.global.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.signup.global.refill_per_sec"),
			},
		},
		Login: AuthRateLimits{
			IP: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.login.ip.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.login.ip.refill_per_sec"),
			},
			Global: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.login.global.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.login.global.refill_per_sec"),
			},
		},
		Read: ReadRateLimits{
			IP: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.read.ip.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.read.ip.refill_per_sec"),
			},
			User: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.read.user.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.read.user.refill_per_sec"),
			},
			Global: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.read.global.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.read.global.refill_per_sec"),
			},
		},
		Write: WriteRateLimits{
			User: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.write.user.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.write.user.refill_per_sec"),
			},
			Global: RateLimitConfig{
				Burst:        c.Viper.GetInt("rate_limits.write.global.burst"),
				RefillPerSec: c.Viper.GetFloat64("rate_limits.write.global.refill_per_sec"),
			},
		},
		IPTTLSeconds:    c.Viper.GetInt("rate_limits.ip_ttl_seconds"),
		KeyedTTLSeconds: c.Viper.GetInt("rate_limits.keyed_ttl_seconds"),
	}

	return cfg
}

func (c *AIPosterConfig) Schedule() aiaccounts.Schedule {
	return aiaccounts.Schedule{Development: c.DevelopmentMode, MinInterval: time.Duration(c.DevelopmentMinIntervalSeconds) * time.Second, MaxInterval: time.Duration(c.DevelopmentMaxIntervalSeconds) * time.Second}
}
