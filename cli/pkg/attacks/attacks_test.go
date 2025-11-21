package attacks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	// Create a mock pattern
	mockPattern := &MockPattern{
		name:        "TEST001",
		description: "Test pattern",
	}

	// Clear registry for clean test
	Registry = make(map[string]Pattern)

	// Register the pattern
	Register(mockPattern)

	// Verify it was registered
	assert.Contains(t, Registry, "TEST001")
	assert.Equal(t, mockPattern, Registry["TEST001"])
}

func TestGet(t *testing.T) {
	// Setup
	Registry = make(map[string]Pattern)
	mockPattern := &MockPattern{name: "TEST002"}
	Registry["TEST002"] = mockPattern

	t.Run("existing pattern", func(t *testing.T) {
		pattern, ok := Get("TEST002")
		assert.True(t, ok)
		assert.Equal(t, mockPattern, pattern)
	})

	t.Run("non-existent pattern", func(t *testing.T) {
		pattern, ok := Get("NONEXISTENT")
		assert.False(t, ok)
		assert.Nil(t, pattern)
	})
}

func TestList(t *testing.T) {
	// Setup registry with multiple patterns
	Registry = make(map[string]Pattern)
	Registry["T1110.001"] = &MockPattern{name: "T1110.001"}
	Registry["T1078.001"] = &MockPattern{name: "T1078.001"}
	Registry["T1059.001"] = &MockPattern{name: "T1059.001"}

	names := List()

	assert.Len(t, names, 3)
	assert.Contains(t, names, "T1110.001")
	assert.Contains(t, names, "T1078.001")
	assert.Contains(t, names, "T1059.001")
}

func TestList_EmptyRegistry(t *testing.T) {
	Registry = make(map[string]Pattern)

	names := List()

	assert.Empty(t, names)
	assert.NotNil(t, names) // Should return empty slice, not nil
}

func TestGetParam(t *testing.T) {
	t.Run("existing string parameter", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"test-key": "test-value",
			},
		}

		result := GetParam(cfg, "test-key", "default")
		assert.Equal(t, "test-value", result)
	})

	t.Run("missing parameter returns default", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{},
		}

		result := GetParam(cfg, "missing-key", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("nil params returns default", func(t *testing.T) {
		cfg := &Config{
			Params: nil,
		}

		result := GetParam(cfg, "any-key", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("wrong type returns default", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"number": 42,
			},
		}

		// Trying to get as string when it's actually an int
		result := GetParam(cfg, "number", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("integer parameter", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"count": 100,
			},
		}

		result := GetParam(cfg, "count", 0)
		assert.Equal(t, 100, result)
	})

	t.Run("boolean parameter", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"enabled": true,
			},
		}

		result := GetParam(cfg, "enabled", false)
		assert.True(t, result)
	})
}

func TestGetIntParam(t *testing.T) {
	t.Run("existing int parameter", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"count": 42,
			},
		}

		result := GetIntParam(cfg, "count", 10)
		assert.Equal(t, 42, result)
	})

	t.Run("string parameter parsed to int", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"count": "100",
			},
		}

		result := GetIntParam(cfg, "count", 10)
		assert.Equal(t, 100, result)
	})

	t.Run("invalid string returns default", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"count": "not-a-number",
			},
		}

		result := GetIntParam(cfg, "count", 10)
		assert.Equal(t, 10, result)
	})

	t.Run("missing parameter returns default", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{},
		}

		result := GetIntParam(cfg, "missing", 99)
		assert.Equal(t, 99, result)
	})

	t.Run("nil params returns default", func(t *testing.T) {
		cfg := &Config{
			Params: nil,
		}

		result := GetIntParam(cfg, "any-key", 55)
		assert.Equal(t, 55, result)
	})

	t.Run("zero value", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"count": 0,
			},
		}

		result := GetIntParam(cfg, "count", 10)
		assert.Equal(t, 0, result)
	})

	t.Run("negative value", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"count": -5,
			},
		}

		result := GetIntParam(cfg, "count", 10)
		assert.Equal(t, -5, result)
	})
}

func TestGetStringParam(t *testing.T) {
	t.Run("existing string parameter", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"username": "admin",
			},
		}

		result := GetStringParam(cfg, "username", "default")
		assert.Equal(t, "admin", result)
	})

	t.Run("missing parameter returns default", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{},
		}

		result := GetStringParam(cfg, "missing", "default-value")
		assert.Equal(t, "default-value", result)
	})

	t.Run("nil params returns default", func(t *testing.T) {
		cfg := &Config{
			Params: nil,
		}

		result := GetStringParam(cfg, "any-key", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("non-string type returns default", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"number": 42,
			},
		}

		result := GetStringParam(cfg, "number", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("empty string value", func(t *testing.T) {
		cfg := &Config{
			Params: map[string]interface{}{
				"empty": "",
			},
		}

		result := GetStringParam(cfg, "empty", "default")
		assert.Equal(t, "", result) // Should return empty string, not default
	})
}

func TestHECEvent(t *testing.T) {
	t.Run("create HEC event", func(t *testing.T) {
		now := time.Now()
		event := HECEvent{
			Time: float64(now.Unix()) + float64(now.Nanosecond())/1e9,
			Event: map[string]interface{}{
				"class_uid": 3002,
				"message":   "test event",
			},
			SourceType: "ocsf:authentication",
			Index:      "main",
		}

		assert.NotZero(t, event.Time)
		assert.Equal(t, 3002, event.Event["class_uid"])
		assert.Equal(t, "test event", event.Event["message"])
		assert.Equal(t, "ocsf:authentication", event.SourceType)
		assert.Equal(t, "main", event.Index)
	})

	t.Run("HEC event with fractional timestamp", func(t *testing.T) {
		now := time.Unix(1234567890, 123456789)
		timestamp := float64(now.Unix()) + float64(now.Nanosecond())/1e9

		event := HECEvent{
			Time: timestamp,
			Event: map[string]interface{}{
				"test": "data",
			},
		}

		// Verify timestamp has fractional seconds
		assert.Greater(t, event.Time, 1234567890.0)
		assert.Less(t, event.Time, 1234567891.0)
	})
}

func TestConfig(t *testing.T) {
	t.Run("create config with all fields", func(t *testing.T) {
		now := time.Now()
		timeSpread := 1 * time.Hour

		cfg := &Config{
			Now:        now,
			TimeSpread: timeSpread,
			Params: map[string]interface{}{
				"target-user": "admin",
				"ip-count":    15,
				"enabled":     true,
			},
		}

		assert.Equal(t, now, cfg.Now)
		assert.Equal(t, timeSpread, cfg.TimeSpread)
		assert.Len(t, cfg.Params, 3)
		assert.Equal(t, "admin", cfg.Params["target-user"])
		assert.Equal(t, 15, cfg.Params["ip-count"])
		assert.True(t, cfg.Params["enabled"].(bool))
	})

	t.Run("config with nil params", func(t *testing.T) {
		cfg := &Config{
			Now:        time.Now(),
			TimeSpread: 5 * time.Minute,
			Params:     nil,
		}

		assert.NotNil(t, cfg)
		assert.Nil(t, cfg.Params)

		// Should handle nil params gracefully
		result := GetStringParam(cfg, "any", "default")
		assert.Equal(t, "default", result)
	})
}

// MockPattern is a test implementation of the Pattern interface
type MockPattern struct {
	name        string
	description string
	events      []HECEvent
	err         error
}

func (m *MockPattern) Name() string {
	return m.name
}

func (m *MockPattern) Description() string {
	return m.description
}

func (m *MockPattern) Generate(cfg *Config) ([]HECEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func (m *MockPattern) DefaultParams() map[string]interface{} {
	return map[string]interface{}{
		"test-param": "test-value",
	}
}

func TestMockPattern(t *testing.T) {
	// Verify mock pattern implements Pattern interface
	var _ Pattern = (*MockPattern)(nil)

	mock := &MockPattern{
		name:        "MOCK001",
		description: "Mock attack pattern",
		events: []HECEvent{
			{
				Time:       float64(time.Now().Unix()),
				Event:      map[string]interface{}{"test": "event"},
				SourceType: "test",
			},
		},
	}

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "MOCK001", mock.Name())
	})

	t.Run("description", func(t *testing.T) {
		assert.Equal(t, "Mock attack pattern", mock.Description())
	})

	t.Run("generate success", func(t *testing.T) {
		cfg := &Config{
			Now:        time.Now(),
			TimeSpread: 1 * time.Hour,
		}

		events, err := mock.Generate(cfg)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "test", events[0].SourceType)
	})

	t.Run("generate error", func(t *testing.T) {
		mockWithError := &MockPattern{
			name: "ERROR",
			err:  assert.AnError,
		}

		cfg := &Config{}
		events, err := mockWithError.Generate(cfg)
		assert.Error(t, err)
		assert.Nil(t, events)
	})

	t.Run("default params", func(t *testing.T) {
		params := mock.DefaultParams()
		assert.Contains(t, params, "test-param")
		assert.Equal(t, "test-value", params["test-param"])
	})
}

func BenchmarkGetParam(b *testing.B) {
	cfg := &Config{
		Params: map[string]interface{}{
			"test-key": "test-value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetParam(cfg, "test-key", "default")
	}
}

func BenchmarkGetIntParam(b *testing.B) {
	cfg := &Config{
		Params: map[string]interface{}{
			"count": 42,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetIntParam(cfg, "count", 10)
	}
}

func BenchmarkGetStringParam(b *testing.B) {
	cfg := &Config{
		Params: map[string]interface{}{
			"username": "admin",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetStringParam(cfg, "username", "default")
	}
}
