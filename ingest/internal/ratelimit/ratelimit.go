package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/metrics"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Close() error
}

type redisRateLimiter struct {
	client   *redis.Client
	limit    int64
	window   time.Duration
	disabled bool
}

func NewRedisRateLimiter(redisURL string, limit int, window time.Duration, disabled bool) (RateLimiter, error) {
	if disabled {
		return &redisRateLimiter{disabled: true}, nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &redisRateLimiter{
		client:   client,
		limit:    int64(limit),
		window:   window,
		disabled: false,
	}, nil
}

// Allow implements sliding window rate limiting using Redis
func (r *redisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if r.disabled {
		return true, nil
	}

	now := time.Now().UnixNano()
	windowStart := now - r.window.Nanoseconds()

	// Redis Lua script for atomic rate limiting
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		
		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)
		
		-- Count current entries
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- Add new entry
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, 60)
			return 1
		else
			return 0
		end
	`

	result, err := r.client.Eval(ctx, script, []string{"ratelimit:" + key}, now, windowStart, r.limit).Int()
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	allowed := result == 1
	if !allowed {
		metrics.RateLimitHits.WithLabelValues(key).Inc()
	}

	return allowed, nil
}

func (r *redisRateLimiter) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// NoOpRateLimiter always allows requests (for testing or disabled rate limiting)
type NoOpRateLimiter struct{}

func (n *NoOpRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func (n *NoOpRateLimiter) Close() error {
	return nil
}
