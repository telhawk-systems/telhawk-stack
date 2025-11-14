package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestNoOpRateLimiter(t *testing.T) {
	limiter := &NoOpRateLimiter{}
	ctx := context.Background()

	tests := []struct {
		name string
		key  string
	}{
		{
			name: "Any key should be allowed",
			key:  "test-key-1",
		},
		{
			name: "Multiple calls with same key",
			key:  "test-key-2",
		},
		{
			name: "Empty key",
			key:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call multiple times to ensure it always allows
			for i := 0; i < 10; i++ {
				allowed, err := limiter.Allow(ctx, tt.key)
				if err != nil {
					t.Errorf("Allow() error = %v, want nil", err)
				}
				if !allowed {
					t.Errorf("Allow() = false, want true")
				}
			}
		})
	}
}

func TestNoOpRateLimiter_Close(t *testing.T) {
	limiter := &NoOpRateLimiter{}
	err := limiter.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestNewRedisRateLimiter_Disabled(t *testing.T) {
	limiter, err := NewRedisRateLimiter("", 100, time.Minute, true)
	if err != nil {
		t.Fatalf("NewRedisRateLimiter() error = %v, want nil", err)
	}

	ctx := context.Background()
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Errorf("Allow() error = %v, want nil", err)
	}
	if !allowed {
		t.Errorf("Allow() = false, want true (disabled limiter should allow all)")
	}

	err = limiter.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestNewRedisRateLimiter_InvalidURL(t *testing.T) {
	_, err := NewRedisRateLimiter("not-a-valid-url", 100, time.Minute, false)
	if err == nil {
		t.Error("NewRedisRateLimiter() with invalid URL should return error")
	}
}

func TestNewRedisRateLimiter_ConnectionFailed(t *testing.T) {
	// Try to connect to non-existent Redis server
	_, err := NewRedisRateLimiter("redis://localhost:9999", 100, time.Minute, false)
	if err == nil {
		t.Error("NewRedisRateLimiter() with unreachable Redis should return error")
	}
}

// Integration test - requires Redis to be running
// Skip if Redis is not available
func TestRedisRateLimiter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to connect to Redis - skip if not available
	limiter, err := NewRedisRateLimiter("redis://localhost:6379", 5, time.Second, false)
	if err != nil {
		t.Skipf("Redis not available, skipping integration test: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "test-integration-" + time.Now().Format("20060102150405.000")

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow() request %d error = %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Allow() request %d = false, want true", i+1)
		}
	}

	// 6th request should be rate limited
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow() rate limit check error = %v", err)
	}
	if allowed {
		t.Error("Allow() request 6 = true, want false (should be rate limited)")
	}

	// After window expires, should allow again
	time.Sleep(1100 * time.Millisecond) // Wait for window to expire
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow() after window error = %v", err)
	}
	if !allowed {
		t.Error("Allow() after window = false, want true")
	}
}

func TestRedisRateLimiter_DifferentKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	limiter, err := NewRedisRateLimiter("redis://localhost:6379", 2, time.Second, false)
	if err != nil {
		t.Skipf("Redis not available, skipping integration test: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	timestamp := time.Now().Format("20060102150405.000")
	key1 := "test-key-1-" + timestamp
	key2 := "test-key-2-" + timestamp

	// Each key should have independent limits
	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(ctx, key1)
		if err != nil {
			t.Fatalf("Allow(key1) error = %v", err)
		}
		if !allowed {
			t.Errorf("Allow(key1) request %d = false, want true", i+1)
		}

		allowed, err = limiter.Allow(ctx, key2)
		if err != nil {
			t.Fatalf("Allow(key2) error = %v", err)
		}
		if !allowed {
			t.Errorf("Allow(key2) request %d = false, want true", i+1)
		}
	}

	// Both keys should now be at limit
	allowed, err := limiter.Allow(ctx, key1)
	if err != nil {
		t.Fatalf("Allow(key1) limit check error = %v", err)
	}
	if allowed {
		t.Error("Allow(key1) beyond limit = true, want false")
	}

	allowed, err = limiter.Allow(ctx, key2)
	if err != nil {
		t.Fatalf("Allow(key2) limit check error = %v", err)
	}
	if allowed {
		t.Error("Allow(key2) beyond limit = true, want false")
	}
}

func TestRedisRateLimiter_SlidingWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	limiter, err := NewRedisRateLimiter("redis://localhost:6379", 3, 2*time.Second, false)
	if err != nil {
		t.Skipf("Redis not available, skipping integration test: %v", err)
	}
	defer limiter.Close()

	ctx := context.Background()
	key := "test-sliding-" + time.Now().Format("20060102150405.000")

	// Use 2 of 3 allowed requests
	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow() initial request %d error = %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Allow() initial request %d = false, want true", i+1)
		}
	}

	// Wait 1 second (halfway through window)
	time.Sleep(1100 * time.Millisecond)

	// Should still be able to use the remaining 1 request
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow() mid-window error = %v", err)
	}
	if !allowed {
		t.Error("Allow() mid-window = false, want true (1 request remaining)")
	}

	// Now at limit
	allowed, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow() at limit error = %v", err)
	}
	if allowed {
		t.Error("Allow() at limit = true, want false")
	}

	// Wait another second - first 2 requests should expire
	time.Sleep(1100 * time.Millisecond)

	// Should allow again (2 requests expired from window)
	for i := 0; i < 2; i++ {
		allowed, err = limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow() after partial expiry %d error = %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Allow() after partial expiry %d = false, want true", i+1)
		}
	}
}

func TestRedisRateLimiter_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	limiter, err := NewRedisRateLimiter("redis://localhost:6379", 10, time.Minute, false)
	if err != nil {
		t.Skipf("Redis not available, skipping integration test: %v", err)
	}
	defer limiter.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = limiter.Allow(ctx, "test-cancelled")
	if err == nil {
		t.Error("Allow() with cancelled context should return error")
	}
}
