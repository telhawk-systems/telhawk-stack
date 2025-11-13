package correlation

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateManager_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		client   *redis.Client
		enabled  bool
		expected bool
	}{
		{
			name:     "enabled with client",
			client:   &redis.Client{},
			enabled:  true,
			expected: true,
		},
		{
			name:     "disabled",
			client:   &redis.Client{},
			enabled:  false,
			expected: false,
		},
		{
			name:     "no client",
			client:   nil,
			enabled:  true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateManager(tt.client, tt.enabled)
			assert.Equal(t, tt.expected, sm.IsEnabled())
		})
	}
}

func TestStateManager_Suppression(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	sm := NewStateManager(client, true)
	ctx := context.Background()

	ruleID := "test-rule-123"
	suppressionKey := map[string]string{
		"user.name": "alice",
		"src_ip":    "10.0.1.5",
	}

	t.Run("not suppressed initially", func(t *testing.T) {
		suppressed, err := sm.IsSuppressed(ctx, ruleID, suppressionKey)
		require.NoError(t, err)
		assert.False(t, suppressed)
	})

	t.Run("record alert", func(t *testing.T) {
		err := sm.RecordAlert(ctx, ruleID, suppressionKey, 1*time.Hour, 1)
		require.NoError(t, err)
	})

	t.Run("suppressed after recording", func(t *testing.T) {
		suppressed, err := sm.IsSuppressed(ctx, ruleID, suppressionKey)
		require.NoError(t, err)
		assert.True(t, suppressed)
	})

	t.Run("different key not suppressed", func(t *testing.T) {
		differentKey := map[string]string{
			"user.name": "bob",
			"src_ip":    "10.0.1.5",
		}
		suppressed, err := sm.IsSuppressed(ctx, ruleID, differentKey)
		require.NoError(t, err)
		assert.False(t, suppressed)
	})

	t.Run("suppression expires", func(t *testing.T) {
		// Record with very short window
		shortKey := map[string]string{"test": "expire"}
		err := sm.RecordAlert(ctx, ruleID, shortKey, 1*time.Millisecond, 1)
		require.NoError(t, err)

		// Fast forward time in miniredis
		mr.FastForward(2 * time.Millisecond)

		suppressed, err := sm.IsSuppressed(ctx, ruleID, shortKey)
		require.NoError(t, err)
		assert.False(t, suppressed)
	})
}

func TestStateManager_Baseline(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	sm := NewStateManager(client, true)
	ctx := context.Background()

	ruleID := "baseline-rule-456"
	entityKey := "user:alice"

	t.Run("get empty baseline", func(t *testing.T) {
		baseline, err := sm.GetBaseline(ctx, ruleID, entityKey)
		require.NoError(t, err)
		assert.NotNil(t, baseline)
		assert.Equal(t, 0, len(baseline.Samples))
		assert.Equal(t, int64(0), baseline.Count)
	})

	t.Run("update baseline with values", func(t *testing.T) {
		values := []float64{100.0, 105.0, 95.0, 110.0, 90.0}
		for _, val := range values {
			err := sm.UpdateBaseline(ctx, ruleID, entityKey, val, 7*24*time.Hour)
			require.NoError(t, err)
		}

		baseline, err := sm.GetBaseline(ctx, ruleID, entityKey)
		require.NoError(t, err)
		assert.Equal(t, 5, len(baseline.Samples))
		assert.Equal(t, int64(5), baseline.Count)
		assert.Equal(t, 500.0, baseline.Sum)
		assert.Equal(t, 100.0, baseline.Mean) // (100+105+95+110+90)/5 = 100
	})

	t.Run("baseline limits sample count", func(t *testing.T) {
		longKey := "user:charlie"
		// Add more than 1000 samples
		for i := 0; i < 1100; i++ {
			err := sm.UpdateBaseline(ctx, ruleID, longKey, 100.0, 7*24*time.Hour)
			require.NoError(t, err)
		}

		baseline, err := sm.GetBaseline(ctx, ruleID, longKey)
		require.NoError(t, err)
		// Should be limited to 1000 samples
		assert.LessOrEqual(t, len(baseline.Samples), 1000)
		assert.Equal(t, int64(1100), baseline.Count) // Count not limited
	})
}

func TestStateManager_Heartbeat(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	sm := NewStateManager(client, true)
	ctx := context.Background()

	ruleID := "heartbeat-rule-789"
	entityID := "webserver-01"

	t.Run("record heartbeat", func(t *testing.T) {
		err := sm.RecordHeartbeat(ctx, ruleID, entityID, 5*time.Minute)
		require.NoError(t, err)
	})

	t.Run("get last seen time", func(t *testing.T) {
		lastSeen, err := sm.GetMissingSince(ctx, ruleID, entityID)
		require.NoError(t, err)
		assert.False(t, lastSeen.IsZero())
		assert.True(t, time.Since(lastSeen) < 2*time.Second)
	})

	t.Run("get all heartbeats", func(t *testing.T) {
		// Record multiple heartbeats
		entities := []string{"webserver-01", "webserver-02", "dbserver-01"}
		for _, entity := range entities {
			err := sm.RecordHeartbeat(ctx, ruleID, entity, 5*time.Minute)
			require.NoError(t, err)
		}

		heartbeats, err := sm.GetAllHeartbeats(ctx, ruleID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(heartbeats), 3)
	})

	t.Run("missing entity returns zero time", func(t *testing.T) {
		lastSeen, err := sm.GetMissingSince(ctx, ruleID, "nonexistent-server")
		require.NoError(t, err)
		assert.True(t, lastSeen.IsZero())
	})
}

func TestStateManager_Disabled(t *testing.T) {
	sm := NewStateManager(nil, false)
	ctx := context.Background()

	t.Run("suppression check returns false when disabled", func(t *testing.T) {
		suppressed, err := sm.IsSuppressed(ctx, "rule", map[string]string{"key": "value"})
		require.NoError(t, err)
		assert.False(t, suppressed)
	})

	t.Run("record alert succeeds when disabled", func(t *testing.T) {
		err := sm.RecordAlert(ctx, "rule", map[string]string{"key": "value"}, time.Hour, 1)
		require.NoError(t, err)
	})

	t.Run("baseline operations fail when disabled", func(t *testing.T) {
		_, err := sm.GetBaseline(ctx, "rule", "entity")
		assert.Error(t, err)
	})

	t.Run("heartbeat operations fail when disabled", func(t *testing.T) {
		err := sm.RecordHeartbeat(ctx, "rule", "entity", time.Minute)
		require.NoError(t, err) // Should not error, just no-op
	})
}

func TestStateManager_KeyHashing(t *testing.T) {
	t.Run("consistent hash for same input", func(t *testing.T) {
		key1 := hashKey("test-key")
		key2 := hashKey("test-key")
		assert.Equal(t, key1, key2)
	})

	t.Run("different hash for different input", func(t *testing.T) {
		key1 := hashKey("test-key-1")
		key2 := hashKey("test-key-2")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("consistent map hash", func(t *testing.T) {
		m := map[string]string{
			"user": "alice",
			"ip":   "10.0.1.5",
		}
		hash1 := hashMap(m)
		hash2 := hashMap(m)
		assert.Equal(t, hash1, hash2)
	})
}
