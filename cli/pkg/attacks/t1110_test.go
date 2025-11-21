package attacks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestT1110_Name(t *testing.T) {
	attack := &T1110{}
	assert.Equal(t, "T1110.001", attack.Name())
}

func TestT1110_Description(t *testing.T) {
	attack := &T1110{}
	desc := attack.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Brute Force")
	assert.Contains(t, desc, "Password Guessing")
}

func TestT1110_DefaultParams(t *testing.T) {
	attack := &T1110{}
	params := attack.DefaultParams()

	// Verify all expected default parameters exist
	assert.Contains(t, params, "ip-count")
	assert.Contains(t, params, "attempts-per-ip")
	assert.Contains(t, params, "target-user")

	// Verify default values
	assert.Equal(t, 15, params["ip-count"])
	assert.Equal(t, 3, params["attempts-per-ip"])
	assert.Equal(t, "admin", params["target-user"])
}

func TestT1110_Generate(t *testing.T) {
	attack := &T1110{}
	now := time.Now()
	timeSpread := 1 * time.Hour

	t.Run("generate with default parameters", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: timeSpread,
			Params:     attack.DefaultParams(),
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		// Should generate ip-count * attempts-per-ip events
		expectedCount := 15 * 3
		assert.Len(t, events, expectedCount)

		// Verify first event structure
		firstEvent := events[0]
		assert.NotZero(t, firstEvent.Time)
		assert.Equal(t, "ocsf:authentication", firstEvent.SourceType)

		// Verify event content
		eventData := firstEvent.Event
		assert.Equal(t, 3002, eventData["class_uid"]) // Authentication class
		assert.Equal(t, "Authentication", eventData["class_name"])
		assert.Equal(t, 3, eventData["category_uid"])
		assert.Equal(t, 1, eventData["activity_id"]) // Login activity
		assert.Equal(t, 2, eventData["status_id"])   // Failure status
		assert.Equal(t, "Failure", eventData["status"])

		// Verify actor information
		actor, ok := eventData["actor"].(map[string]interface{})
		require.True(t, ok)
		user, ok := actor["user"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "admin", user["name"])
		assert.NotEmpty(t, user["uid"])
		assert.NotEmpty(t, user["email"])

		// Verify source endpoint
		srcEndpoint, ok := eventData["src_endpoint"].(map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, srcEndpoint["ip"])
		assert.NotZero(t, srcEndpoint["port"])
		assert.NotEmpty(t, srcEndpoint["hostname"])

		// Verify metadata
		metadata, ok := eventData["metadata"].(map[string]interface{})
		require.True(t, ok)
		product, ok := metadata["product"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "TelHawk", product["vendor_name"])
		assert.Contains(t, product["name"], "T1110.001")

		tags, ok := metadata["tags"].([]string)
		require.True(t, ok)
		assert.Contains(t, tags, "T1110.001")
		assert.Contains(t, tags, "brute-force")
	})

	t.Run("generate with custom parameters", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: timeSpread,
			Params: map[string]interface{}{
				"ip-count":        5,
				"attempts-per-ip": 10,
				"target-user":     "testuser",
			},
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		expectedCount := 5 * 10
		assert.Len(t, events, expectedCount)

		// Verify custom target user is used
		for _, event := range events {
			actor := event.Event["actor"].(map[string]interface{})
			user := actor["user"].(map[string]interface{})
			assert.Equal(t, "testuser", user["name"])
		}

		// Verify we have events from 5 unique IPs
		uniqueIPs := make(map[string]bool)
		for _, event := range events {
			srcEndpoint := event.Event["src_endpoint"].(map[string]interface{})
			ip := srcEndpoint["ip"].(string)
			uniqueIPs[ip] = true
		}
		assert.Len(t, uniqueIPs, 5)
	})

	t.Run("generate with zero time spread", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: 0,
			Params: map[string]interface{}{
				"ip-count":        2,
				"attempts-per-ip": 2,
				"target-user":     "admin",
			},
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		assert.Len(t, events, 4)

		// All events should have similar timestamps (within same second)
		// when time spread is zero
		firstTime := events[0].Time
		for _, event := range events {
			// Allow small variance due to jitter calculation
			assert.InDelta(t, firstTime, event.Time, 1.0)
		}
	})

	t.Run("generate with large time spread", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: 24 * time.Hour, // 1 day
			Params: map[string]interface{}{
				"ip-count":        10,
				"attempts-per-ip": 5,
				"target-user":     "admin",
			},
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		assert.Len(t, events, 50)

		// Find min and max timestamps
		minTime := events[0].Time
		maxTime := events[0].Time
		for _, event := range events {
			if event.Time < minTime {
				minTime = event.Time
			}
			if event.Time > maxTime {
				maxTime = event.Time
			}
		}

		// Events should be spread across roughly 24 hours
		spreadSeconds := maxTime - minTime
		oneDaySeconds := 24.0 * 60.0 * 60.0
		// Allow some tolerance due to jitter
		assert.Greater(t, spreadSeconds, oneDaySeconds*0.8)
	})

	t.Run("verify unique source IPs", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: 1 * time.Hour,
			Params: map[string]interface{}{
				"ip-count":        20,
				"attempts-per-ip": 3,
				"target-user":     "admin",
			},
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		// Extract all source IPs
		ipCounts := make(map[string]int)
		for _, event := range events {
			srcEndpoint := event.Event["src_endpoint"].(map[string]interface{})
			ip := srcEndpoint["ip"].(string)
			ipCounts[ip]++
		}

		// Should have exactly 20 unique IPs
		assert.Len(t, ipCounts, 20)

		// Each IP should have exactly 3 attempts
		for ip, count := range ipCounts {
			assert.Equal(t, 3, count, "IP %s should have 3 attempts", ip)
		}
	})

	t.Run("verify status details vary", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: 1 * time.Hour,
			Params: map[string]interface{}{
				"ip-count":        10,
				"attempts-per-ip": 10,
				"target-user":     "admin",
			},
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		// Collect all status details
		statusDetails := make(map[string]int)
		for _, event := range events {
			detail, ok := event.Event["status_detail"].(string)
			if ok {
				statusDetails[detail]++
			}
		}

		// Should have multiple different status messages (not all the same)
		assert.Greater(t, len(statusDetails), 1, "Should have variety in status details")

		// Verify status details are one of the expected values
		validStatuses := map[string]bool{
			"Invalid credentials":          true,
			"Account locked":               true,
			"Password expired":             true,
			"Invalid username or password": true,
			"Too many failed attempts":     true,
		}

		for status := range statusDetails {
			assert.True(t, validStatuses[status], "Unexpected status detail: %s", status)
		}
	})

	t.Run("verify auth protocol is set", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: 1 * time.Hour,
			Params: map[string]interface{}{
				"ip-count":        2,
				"attempts-per-ip": 2,
				"target-user":     "admin",
			},
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		for _, event := range events {
			assert.Equal(t, "LDAP", event.Event["auth_protocol"])
			assert.NotEmpty(t, event.Event["message"])
		}
	})

	t.Run("empty config uses defaults", func(t *testing.T) {
		cfg := &Config{
			Now:        now,
			TimeSpread: 1 * time.Hour,
			Params:     nil, // No params provided
		}

		events, err := attack.Generate(cfg)
		require.NoError(t, err)

		// Should use default values from GetIntParam/GetStringParam
		// Default: 15 IPs * 3 attempts = 45 events
		assert.Len(t, events, 45)
	})
}

func TestT1110_PatternInterface(t *testing.T) {
	// Verify T1110 implements Pattern interface
	var _ Pattern = (*T1110)(nil)

	attack := &T1110{}
	assert.NotNil(t, attack)
}

func TestSelectRandomStatus(t *testing.T) {
	// Run multiple times to check for variety
	statusCounts := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		status := selectRandomStatus()
		statusCounts[status]++
	}

	// Should get multiple different statuses
	assert.Greater(t, len(statusCounts), 1, "Should have variety in random statuses")

	// All statuses should be valid
	validStatuses := map[string]bool{
		"Invalid credentials":          true,
		"Account locked":               true,
		"Password expired":             true,
		"Invalid username or password": true,
		"Too many failed attempts":     true,
	}

	for status := range statusCounts {
		assert.True(t, validStatuses[status], "Invalid status: %s", status)
	}

	// With 1000 iterations, each of 5 statuses should appear at least once
	// (statistically very likely)
	assert.GreaterOrEqual(t, len(statusCounts), 3, "Should see multiple status types")
}

func TestCalculateJitteredTime(t *testing.T) {
	now := time.Now()

	t.Run("zero time spread returns now", func(t *testing.T) {
		result := calculateJitteredTime(now, 0, 0, 10)
		assert.Equal(t, now, result)
	})

	t.Run("single event", func(t *testing.T) {
		result := calculateJitteredTime(now, 1*time.Hour, 0, 1)
		// Should be somewhere within the hour before now
		assert.True(t, result.Before(now) || result.Equal(now))
		assert.True(t, result.After(now.Add(-1*time.Hour)) || result.Equal(now.Add(-1*time.Hour)))
	})

	t.Run("multiple events are distributed", func(t *testing.T) {
		totalEvents := 10
		timeSpread := 1 * time.Hour
		times := make([]time.Time, totalEvents)

		for i := 0; i < totalEvents; i++ {
			times[i] = calculateJitteredTime(now, timeSpread, i, totalEvents)
		}

		// Verify all times are within the spread window
		earliest := now.Add(-timeSpread)
		for _, eventTime := range times {
			assert.True(t, eventTime.After(earliest) || eventTime.Equal(earliest), "Time %v should be after %v", eventTime, earliest)
			assert.True(t, eventTime.Before(now) || eventTime.Equal(now), "Time %v should be before %v", eventTime, now)
		}

		// Verify times are generally increasing (allowing for jitter)
		// First event should be earlier than last event on average
		assert.True(t, times[0].Before(times[len(times)-1]),
			"First event (%v) should generally be before last event (%v)", times[0], times[len(times)-1])
	})

	t.Run("jitter creates variance", func(t *testing.T) {
		// Generate same index multiple times, should get different results due to jitter
		times := make([]time.Time, 100)
		for i := 0; i < 100; i++ {
			times[i] = calculateJitteredTime(now, 1*time.Hour, 5, 10)
		}

		// Count unique timestamps
		uniqueTimes := make(map[int64]bool)
		for _, t := range times {
			uniqueTimes[t.UnixNano()] = true
		}

		// Should have many unique timestamps due to jitter
		assert.Greater(t, len(uniqueTimes), 50, "Jitter should create timestamp variance")
	})

	t.Run("boundary conditions", func(t *testing.T) {
		timeSpread := 1 * time.Hour

		// First event (index 0)
		firstEvent := calculateJitteredTime(now, timeSpread, 0, 100)
		assert.True(t, firstEvent.After(now.Add(-timeSpread)))
		assert.True(t, firstEvent.Before(now) || firstEvent.Equal(now))

		// Last event (index 99)
		lastEvent := calculateJitteredTime(now, timeSpread, 99, 100)
		assert.True(t, lastEvent.After(now.Add(-timeSpread)) || lastEvent.Equal(now.Add(-timeSpread)))
		assert.True(t, lastEvent.Before(now) || lastEvent.Equal(now))
	})

	t.Run("large number of events", func(t *testing.T) {
		totalEvents := 1000
		timeSpread := 24 * time.Hour

		// Generate all events
		times := make([]time.Time, totalEvents)
		for i := 0; i < totalEvents; i++ {
			times[i] = calculateJitteredTime(now, timeSpread, i, totalEvents)
		}

		// Verify spread coverage
		earliest := times[0]
		latest := times[0]
		for _, t := range times {
			if t.Before(earliest) {
				earliest = t
			}
			if t.After(latest) {
				latest = t
			}
		}

		actualSpread := latest.Sub(earliest)
		// Should use most of the available time spread (allowing for jitter)
		assert.Greater(t, actualSpread, timeSpread*8/10)
	})
}

func TestT1110_Registration(t *testing.T) {
	// Re-register T1110 since other tests clear the registry
	Register(&T1110{})

	// T1110 should be registered
	pattern, exists := Get("T1110.001")
	assert.True(t, exists, "T1110 should be registered")
	assert.NotNil(t, pattern)

	// Verify it's the correct type
	t1110, ok := pattern.(*T1110)
	assert.True(t, ok, "Pattern should be *T1110 type")
	assert.NotNil(t, t1110)
}

func BenchmarkT1110_Generate(b *testing.B) {
	attack := &T1110{}
	now := time.Now()
	cfg := &Config{
		Now:        now,
		TimeSpread: 1 * time.Hour,
		Params: map[string]interface{}{
			"ip-count":        15,
			"attempts-per-ip": 3,
			"target-user":     "admin",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attack.Generate(cfg)
	}
}

func BenchmarkCalculateJitteredTime(b *testing.B) {
	now := time.Now()
	timeSpread := 1 * time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateJitteredTime(now, timeSpread, i%100, 100)
	}
}

func BenchmarkSelectRandomStatus(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selectRandomStatus()
	}
}

// Example demonstrates how to use the T1110 attack pattern
func ExampleT1110_Generate() {
	attack := &T1110{}

	cfg := &Config{
		Now:        time.Now(),
		TimeSpread: 1 * time.Hour,
		Params: map[string]interface{}{
			"ip-count":        10,
			"attempts-per-ip": 5,
			"target-user":     "admin",
		},
	}

	events, err := attack.Generate(cfg)
	if err != nil {
		panic(err)
	}

	// Will generate 10 IPs * 5 attempts = 50 failed login events
	_ = events
}
