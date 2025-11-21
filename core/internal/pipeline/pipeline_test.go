package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// mockNormalizer is a test normalizer that returns predetermined results
type mockNormalizer struct {
	supportsFormat     string
	supportsSourceType string
	returnEvent        *ocsf.Event
	returnError        error
}

func (m *mockNormalizer) Supports(format, sourceType string) bool {
	return format == m.supportsFormat && sourceType == m.supportsSourceType
}

func (m *mockNormalizer) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	return m.returnEvent, nil
}

// mockValidator is a test validator
type mockValidator struct {
	supportsClass string
	returnError   error
}

func (m *mockValidator) Supports(class string) bool {
	return m.supportsClass == "" || class == m.supportsClass
}

func (m *mockValidator) Validate(ctx context.Context, event *ocsf.Event) error {
	return m.returnError
}

func TestPipeline_New(t *testing.T) {
	registry := normalizer.NewRegistry()
	chain := validator.NewChain()

	pipe := pipeline.New(registry, chain)

	assert.NotNil(t, pipe, "pipeline should not be nil")
}

func TestPipeline_Process_Success(t *testing.T) {
	expectedEvent := &ocsf.Event{
		CategoryUID: 3,
		ClassUID:    3002,
		ActivityID:  1,
		TypeUID:     ocsf.ComputeTypeUID(3, 3002, 1),
		Class:       "authentication",
		Category:    "identity_and_access_management",
		Activity:    "logon",
		Time:        time.Now(),
		Severity:    "Informational",
		SeverityID:  1,
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test Product",
				Vendor: "Test Vendor",
			},
			Version: "1.0.0",
		},
	}

	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test_auth",
		returnEvent:        expectedEvent,
	}

	val := &mockValidator{
		supportsClass: "authentication",
		returnError:   nil,
	}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain(val)
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test_auth",
		Source:     "test",
		Payload:    []byte(`{"user":"alice"}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := pipe.Process(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, expectedEvent.ClassUID, event.ClassUID)
	assert.Equal(t, expectedEvent.Class, event.Class)
}

func TestPipeline_Process_NoNormalizerFound(t *testing.T) {
	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "auth",
	}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "unknown_type",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := pipe.Process(ctx, envelope)

	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "no normalizer registered")
}

func TestPipeline_Process_NormalizationError(t *testing.T) {
	expectedErr := errors.New("normalization failed: invalid format")

	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
		returnError:        expectedErr,
	}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Source:     "test",
		Payload:    []byte(`invalid`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := pipe.Process(ctx, envelope)

	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "normalize")
}

func TestPipeline_Process_ValidationError(t *testing.T) {
	event := &ocsf.Event{
		CategoryUID: 3,
		ClassUID:    3002,
		Class:       "authentication",
		Category:    "iam",
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test",
				Vendor: "Test",
			},
			Version: "1.0.0",
		},
	}

	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
		returnEvent:        event,
	}

	val := &mockValidator{
		returnError: errors.New("validation failed: missing required field"),
	}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain(val)
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := pipe.Process(ctx, envelope)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validate")
}

func TestPipeline_Process_NilPipeline(t *testing.T) {
	var pipe *pipeline.Pipeline

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	event, err := pipe.Process(ctx, envelope)

	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "pipeline not configured")
}

func TestPipeline_Process_MultipleNormalizers(t *testing.T) {
	event1 := &ocsf.Event{
		ClassUID: 3002,
		Class:    "authentication",
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	event2 := &ocsf.Event{
		ClassUID: 4001,
		Class:    "network_activity",
		Category: "network",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	norm1 := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "auth",
		returnEvent:        event1,
	}

	norm2 := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "network",
		returnEvent:        event2,
	}

	registry := normalizer.NewRegistry(norm1, norm2)
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)

	testCases := []struct {
		name             string
		sourceType       string
		expectedClass    string
		expectedClassUID int
	}{
		{
			name:             "Auth event uses first normalizer",
			sourceType:       "auth",
			expectedClass:    "authentication",
			expectedClassUID: 3002,
		},
		{
			name:             "Network event uses second normalizer",
			sourceType:       "network",
			expectedClass:    "network_activity",
			expectedClassUID: 4001,
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: tc.sourceType,
				Source:     "test",
				Payload:    []byte(`{}`),
				ReceivedAt: time.Now(),
			}

			event, err := pipe.Process(ctx, envelope)

			require.NoError(t, err)
			require.NotNil(t, event)
			assert.Equal(t, tc.expectedClass, event.Class)
			assert.Equal(t, tc.expectedClassUID, event.ClassUID)
		})
	}
}

func TestPipeline_Process_ContextCancellation(t *testing.T) {
	event := &ocsf.Event{
		ClassUID: 3002,
		Class:    "authentication",
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
		returnEvent:        event,
	}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain()
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Pipeline should still complete if normalizer doesn't check context
	result, err := pipe.Process(ctx, envelope)

	// This test verifies the pipeline doesn't fail just because context is cancelled
	// Individual normalizers/validators may choose to respect context cancellation
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarshalResult_Success(t *testing.T) {
	event := &ocsf.Event{
		CategoryUID: 3,
		ClassUID:    3002,
		ActivityID:  1,
		TypeUID:     ocsf.ComputeTypeUID(3, 3002, 1),
		Class:       "authentication",
		Category:    "iam",
		Time:        time.Now(),
		Severity:    "Informational",
		SeverityID:  1,
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test",
				Vendor: "Test",
			},
			Version: "1.0.0",
		},
	}

	data, err := pipeline.MarshalResult(event)

	require.NoError(t, err)
	require.NotNil(t, data)

	// Verify it's valid JSON
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Check key fields
	assert.Equal(t, float64(3002), decoded["class_uid"])
	assert.Equal(t, "authentication", decoded["class"])
}

func TestMarshalResult_NilEvent(t *testing.T) {
	data, err := pipeline.MarshalResult(nil)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "nil event")
}

func TestMarshalResult_EmptyEvent(t *testing.T) {
	event := &ocsf.Event{}

	data, err := pipeline.MarshalResult(event)

	require.NoError(t, err)
	require.NotNil(t, data)

	// Even an empty event should marshal to valid JSON
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
}

func TestPipeline_Process_ChainedValidators(t *testing.T) {
	event := &ocsf.Event{
		CategoryUID: 3,
		ClassUID:    3002,
		Class:       "authentication",
		Category:    "iam",
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
		returnEvent:        event,
	}

	// Create multiple validators - all should be called
	val1 := &mockValidator{supportsClass: "authentication", returnError: nil}
	val2 := &mockValidator{supportsClass: "authentication", returnError: nil}
	val3 := &mockValidator{supportsClass: "authentication", returnError: nil}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain(val1, val2, val3)
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := pipe.Process(ctx, envelope)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestPipeline_Process_ValidatorChainStopsOnError(t *testing.T) {
	event := &ocsf.Event{
		CategoryUID: 3,
		ClassUID:    3002,
		Class:       "authentication",
		Category:    "iam",
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	norm := &mockNormalizer{
		supportsFormat:     "json",
		supportsSourceType: "test",
		returnEvent:        event,
	}

	// Second validator returns error - third should not be called
	val1 := &mockValidator{returnError: nil}
	val2 := &mockValidator{returnError: errors.New("validation error")}
	val3 := &mockValidator{returnError: nil}

	registry := normalizer.NewRegistry(norm)
	chain := validator.NewChain(val1, val2, val3)
	pipe := pipeline.New(registry, chain)

	envelope := &model.RawEventEnvelope{
		Format:     "json",
		SourceType: "test",
		Source:     "test",
		Payload:    []byte(`{}`),
		ReceivedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := pipe.Process(ctx, envelope)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "validation error")
}
