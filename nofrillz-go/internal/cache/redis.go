package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"nofrillz/internal/config"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(config *config.RedisConfig) (*Redis, error) {
	c := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, err
	}

	return &Redis{Client: c}, nil
}

func (r *Redis) Close() error {
	if r == nil || r.Client == nil {
		return nil
	}
	return r.Client.Close()
}
