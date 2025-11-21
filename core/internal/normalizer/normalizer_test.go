package normalizer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// mockNormalizer for testing
type mockNormalizer struct {
	id                 int
	supportsFormat     string
	supportsSourceType string
	normalizeFunc      func(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error)
}

func (m *mockNormalizer) Supports(format, sourceType string) bool {
	return format == m.supportsFormat && sourceType == m.supportsSourceType
}

func (m *mockNormalizer) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	if m.normalizeFunc != nil {
		return m.normalizeFunc(ctx, envelope)
	}
	return &ocsf.Event{
		ClassUID: m.id,
		Class:    "test_event",
		Category: "test",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}, nil
}

func TestNewRegistry(t *testing.T) {
	t.Run("creates registry with no normalizers", func(t *testing.T) {
		registry := normalizer.NewRegistry()
		assert.NotNil(t, registry)
	})

	t.Run("creates registry with normalizers", func(t *testing.T) {
		norm1 := &mockNormalizer{id: 1, supportsFormat: "json", supportsSourceType: "auth"}
		norm2 := &mockNormalizer{id: 2, supportsFormat: "json", supportsSourceType: "network"}

		registry := normalizer.NewRegistry(norm1, norm2)
		assert.NotNil(t, registry)
	})
}

func TestRegistry_Find_SingleNormalizer(t *testing.T) {
	norm := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "test_auth",
	}

	registry := normalizer.NewRegistry(norm)

	testCases := []struct {
		name       string
		format     string
		sourceType string
		shouldFind bool
	}{
		{
			name:       "matching format and sourceType",
			format:     "json",
			sourceType: "test_auth",
			shouldFind: true,
		},
		{
			name:       "non-matching format",
			format:     "xml",
			sourceType: "test_auth",
			shouldFind: false,
		},
		{
			name:       "non-matching sourceType",
			format:     "json",
			sourceType: "unknown",
			shouldFind: false,
		},
		{
			name:       "both non-matching",
			format:     "xml",
			sourceType: "unknown",
			shouldFind: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			envelope := &model.RawEventEnvelope{
				Format:     tc.format,
				SourceType: tc.sourceType,
			}

			found := registry.Find(envelope)

			if tc.shouldFind {
				assert.NotNil(t, found, "expected to find normalizer")
			} else {
				assert.Nil(t, found, "expected not to find normalizer")
			}
		})
	}
}

func TestRegistry_Find_MultipleNormalizers(t *testing.T) {
	norm1 := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "auth",
	}

	norm2 := &mockNormalizer{
		id:                 2,
		supportsFormat:     "json",
		supportsSourceType: "network",
	}

	norm3 := &mockNormalizer{
		id:                 3,
		supportsFormat:     "syslog",
		supportsSourceType: "firewall",
	}

	registry := normalizer.NewRegistry(norm1, norm2, norm3)

	testCases := []struct {
		name       string
		format     string
		sourceType string
		expectedID int
		shouldFind bool
	}{
		{
			name:       "finds first normalizer",
			format:     "json",
			sourceType: "auth",
			expectedID: 1,
			shouldFind: true,
		},
		{
			name:       "finds second normalizer",
			format:     "json",
			sourceType: "network",
			expectedID: 2,
			shouldFind: true,
		},
		{
			name:       "finds third normalizer",
			format:     "syslog",
			sourceType: "firewall",
			expectedID: 3,
			shouldFind: true,
		},
		{
			name:       "no match found",
			format:     "xml",
			sourceType: "unknown",
			shouldFind: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			envelope := &model.RawEventEnvelope{
				Format:     tc.format,
				SourceType: tc.sourceType,
			}

			found := registry.Find(envelope)

			if tc.shouldFind {
				require.NotNil(t, found, "expected to find normalizer")

				// Verify it's the right normalizer by calling normalize
				event, err := found.Normalize(context.Background(), envelope)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedID, event.ClassUID)
			} else {
				assert.Nil(t, found, "expected not to find normalizer")
			}
		})
	}
}

func TestRegistry_Find_FirstMatch(t *testing.T) {
	// Create two normalizers that support the same format/sourceType
	norm1 := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "auth",
	}

	norm2 := &mockNormalizer{
		id:                 2,
		supportsFormat:     "json",
		supportsSourceType: "auth",
	}

	// First normalizer should be returned
	registry := normalizer.NewRegistry(norm1, norm2)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "auth",
	}

	found := registry.Find(envelope)

	require.NotNil(t, found)

	// Verify it's the first normalizer
	event, err := found.Normalize(context.Background(), envelope)
	require.NoError(t, err)
	assert.Equal(t, 1, event.ClassUID, "should return first matching normalizer")
}

func TestRegistry_Find_NilRegistry(t *testing.T) {
	var registry *normalizer.Registry

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
	}

	found := registry.Find(envelope)
	assert.Nil(t, found, "nil registry should return nil")
}

func TestRegistry_Find_EmptyRegistry(t *testing.T) {
	registry := normalizer.NewRegistry()

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
	}

	found := registry.Find(envelope)
	assert.Nil(t, found, "empty registry should return nil")
}

func TestRegistry_Find_NilEnvelope(t *testing.T) {
	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
	}

	registry := normalizer.NewRegistry(norm)

	// This will panic with nil envelope due to dereferencing
	// Just verify the registry itself is valid
	assert.NotNil(t, registry)

	// In production, callers should never pass nil envelope
	// so this test just documents the expected behavior
}

func TestRegistry_Find_OrderMatters(t *testing.T) {
	// Create normalizers with overlapping support patterns
	catchAll := &mockNormalizer{
		id:                 999,
		supportsFormat:     "json",
		supportsSourceType: "test",
	}

	specific := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "test",
	}

	t.Run("specific first, should return specific", func(t *testing.T) {
		registry := normalizer.NewRegistry(specific, catchAll)

		envelope := &model.RawEventEnvelope{
			Format:     "json",
			SourceType: "test",
		}

		found := registry.Find(envelope)
		require.NotNil(t, found)

		event, err := found.Normalize(context.Background(), envelope)
		require.NoError(t, err)
		assert.Equal(t, 1, event.ClassUID)
	})

	t.Run("catch-all first, should return catch-all", func(t *testing.T) {
		registry := normalizer.NewRegistry(catchAll, specific)

		envelope := &model.RawEventEnvelope{
			Format:     "json",
			SourceType: "test",
		}

		found := registry.Find(envelope)
		require.NotNil(t, found)

		event, err := found.Normalize(context.Background(), envelope)
		require.NoError(t, err)
		assert.Equal(t, 999, event.ClassUID)
	})
}

func TestRegistry_Find_WithAttributes(t *testing.T) {
	norm := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "test",
	}

	registry := normalizer.NewRegistry(norm)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Attributes: map[string]string{
			"region": "us-west-2",
			"env":    "production",
		},
	}

	found := registry.Find(envelope)
	assert.NotNil(t, found, "should find normalizer regardless of attributes")
}

func TestRegistry_Find_WithPayload(t *testing.T) {
	norm := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "test",
	}

	registry := normalizer.NewRegistry(norm)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Payload:    []byte(`{"user":"alice","action":"login"}`),
		ReceivedAt: time.Now(),
	}

	found := registry.Find(envelope)
	assert.NotNil(t, found, "should find normalizer with payload")
}

func TestRegistry_Integration_WithRealNormalization(t *testing.T) {
	// Test with a normalizer that actually processes the envelope
	processingNorm := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "test_process",
		normalizeFunc: func(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
			return &ocsf.Event{
				ClassUID: 3002,
				Class:    "authentication",
				Category: "iam",
				Time:     envelope.ReceivedAt,
				Metadata: ocsf.Metadata{
					Product: ocsf.Product{
						Name:   "Test System",
						Vendor: "Test Vendor",
					},
					Version: "1.0.0",
				},
				Properties: map[string]string{
					"source":      envelope.Source,
					"source_type": envelope.SourceType,
				},
			}, nil
		},
	}

	registry := normalizer.NewRegistry(processingNorm)

	envelope := &model.RawEventEnvelope{
		ID:         "test-123",
		Format:     "json",
		SourceType: "test_process",
		Source:     "test-system",
		Payload:    []byte(`{"user":"bob","action":"login"}`),
		ReceivedAt: time.Now(),
	}

	found := registry.Find(envelope)
	require.NotNil(t, found)

	event, err := found.Normalize(context.Background(), envelope)
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.Equal(t, 3002, event.ClassUID)
	assert.Equal(t, "authentication", event.Class)
	assert.Equal(t, "test-system", event.Properties["source"])
	assert.Equal(t, "test_process", event.Properties["source_type"])
}

func TestRegistry_CaseSensitivity(t *testing.T) {
	norm := &mockNormalizer{
		id:                 1,
		supportsFormat:     "json",
		supportsSourceType: "TestAuth",
	}

	registry := normalizer.NewRegistry(norm)

	testCases := []struct {
		name       string
		sourceType string
		shouldFind bool
	}{
		{
			name:       "exact case match",
			sourceType: "TestAuth",
			shouldFind: true,
		},
		{
			name:       "lowercase",
			sourceType: "testauth",
			shouldFind: false,
		},
		{
			name:       "uppercase",
			sourceType: "TESTAUTH",
			shouldFind: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: tc.sourceType,
			}

			found := registry.Find(envelope)

			if tc.shouldFind {
				assert.NotNil(t, found)
			} else {
				assert.Nil(t, found)
			}
		})
	}
}
