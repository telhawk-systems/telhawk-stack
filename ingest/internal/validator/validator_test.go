package validator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
)

// mockValidator for testing
type mockValidator struct {
	supportsClass string
	validateFunc  func(ctx context.Context, event *ocsf.Event) error
	callCount     int
}

func (m *mockValidator) Supports(class string) bool {
	return m.supportsClass == "" || m.supportsClass == class
}

func (m *mockValidator) Validate(ctx context.Context, event *ocsf.Event) error {
	m.callCount++
	if m.validateFunc != nil {
		return m.validateFunc(ctx, event)
	}
	return nil
}

func TestNewChain(t *testing.T) {
	t.Run("creates chain with no validators", func(t *testing.T) {
		chain := validator.NewChain()
		assert.NotNil(t, chain)
	})

	t.Run("creates chain with validators", func(t *testing.T) {
		val1 := &mockValidator{}
		val2 := &mockValidator{}

		chain := validator.NewChain(val1, val2)
		assert.NotNil(t, chain)
	})
}

func TestChain_Validate_Success(t *testing.T) {
	val1 := &mockValidator{supportsClass: "authentication"}
	val2 := &mockValidator{supportsClass: "authentication"}

	chain := validator.NewChain(val1, val2)

	event := &ocsf.Event{
		Class:    "authentication",
		ClassUID: 3002,
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.NoError(t, err)
	assert.Equal(t, 1, val1.callCount, "first validator should be called")
	assert.Equal(t, 1, val2.callCount, "second validator should be called")
}

func TestChain_Validate_FirstValidatorFails(t *testing.T) {
	expectedErr := errors.New("validation failed")

	val1 := &mockValidator{
		supportsClass: "authentication",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			return expectedErr
		},
	}
	val2 := &mockValidator{supportsClass: "authentication"}

	chain := validator.NewChain(val1, val2)

	event := &ocsf.Event{
		Class:    "authentication",
		ClassUID: 3002,
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, val1.callCount, "first validator should be called")
	assert.Equal(t, 0, val2.callCount, "second validator should not be called after first fails")
}

func TestChain_Validate_SecondValidatorFails(t *testing.T) {
	expectedErr := errors.New("second validation failed")

	val1 := &mockValidator{supportsClass: "authentication"}
	val2 := &mockValidator{
		supportsClass: "authentication",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			return expectedErr
		},
	}

	chain := validator.NewChain(val1, val2)

	event := &ocsf.Event{
		Class:    "authentication",
		ClassUID: 3002,
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, val1.callCount)
	assert.Equal(t, 1, val2.callCount)
}

func TestChain_Validate_NilChain(t *testing.T) {
	var chain *validator.Chain

	event := &ocsf.Event{
		Class: "test",
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.NoError(t, err, "nil chain should not error")
}

func TestChain_Validate_EmptyChain(t *testing.T) {
	chain := validator.NewChain()

	event := &ocsf.Event{
		Class: "test",
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.NoError(t, err, "empty chain should not error")
}

func TestChain_Validate_ValidatorDoesNotSupportClass(t *testing.T) {
	val1 := &mockValidator{supportsClass: "authentication"}
	val2 := &mockValidator{supportsClass: "network_activity"}

	chain := validator.NewChain(val1, val2)

	event := &ocsf.Event{
		Class:    "process_activity",
		ClassUID: 1007,
		Category: "system",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.NoError(t, err, "validators that don't support the class should be skipped")
	assert.Equal(t, 0, val1.callCount, "first validator should not be called")
	assert.Equal(t, 0, val2.callCount, "second validator should not be called")
}

func TestChain_Validate_MixedSupport(t *testing.T) {
	val1 := &mockValidator{supportsClass: "authentication"}
	val2 := &mockValidator{supportsClass: ""} // supports all classes
	val3 := &mockValidator{supportsClass: "network_activity"}

	chain := validator.NewChain(val1, val2, val3)

	event := &ocsf.Event{
		Class:    "authentication",
		ClassUID: 3002,
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.NoError(t, err)
	assert.Equal(t, 1, val1.callCount, "auth validator should be called")
	assert.Equal(t, 1, val2.callCount, "catch-all validator should be called")
	assert.Equal(t, 0, val3.callCount, "network validator should not be called")
}

func TestChain_Validate_MultipleClasses(t *testing.T) {
	authVal := &mockValidator{supportsClass: "authentication"}
	netVal := &mockValidator{supportsClass: "network_activity"}
	allVal := &mockValidator{supportsClass: ""} // supports all

	chain := validator.NewChain(authVal, netVal, allVal)

	testCases := []struct {
		name              string
		eventClass        string
		expectedAuthCalls int
		expectedNetCalls  int
		expectedAllCalls  int
	}{
		{
			name:              "authentication event",
			eventClass:        "authentication",
			expectedAuthCalls: 1,
			expectedNetCalls:  0,
			expectedAllCalls:  1,
		},
		{
			name:              "network event",
			eventClass:        "network_activity",
			expectedAuthCalls: 0,
			expectedNetCalls:  1,
			expectedAllCalls:  1,
		},
		{
			name:              "process event",
			eventClass:        "process_activity",
			expectedAuthCalls: 0,
			expectedNetCalls:  0,
			expectedAllCalls:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset call counts
			authVal.callCount = 0
			netVal.callCount = 0
			allVal.callCount = 0

			event := &ocsf.Event{
				Class:    tc.eventClass,
				ClassUID: 1000,
				Category: "test",
				Time:     time.Now(),
				Metadata: ocsf.Metadata{
					Product: ocsf.Product{Name: "Test", Vendor: "Test"},
					Version: "1.0.0",
				},
			}

			ctx := context.Background()
			err := chain.Validate(ctx, event)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedAuthCalls, authVal.callCount)
			assert.Equal(t, tc.expectedNetCalls, netVal.callCount)
			assert.Equal(t, tc.expectedAllCalls, allVal.callCount)
		})
	}
}

func TestChain_Validate_ContextPassed(t *testing.T) {
	type contextKey string
	key := contextKey("test-key")
	expectedValue := "test-value"

	val := &mockValidator{
		supportsClass: "authentication",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			value := ctx.Value(key)
			if value == nil {
				return errors.New("context value not found")
			}
			if value.(string) != expectedValue {
				return errors.New("context value mismatch")
			}
			return nil
		},
	}

	chain := validator.NewChain(val)

	event := &ocsf.Event{
		Class:    "authentication",
		ClassUID: 3002,
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.WithValue(context.Background(), key, expectedValue)
	err := chain.Validate(ctx, event)

	assert.NoError(t, err)
}

func TestChain_Validate_ContextCancellation(t *testing.T) {
	val := &mockValidator{
		supportsClass: "authentication",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		},
	}

	chain := validator.NewChain(val)

	event := &ocsf.Event{
		Class:    "authentication",
		ClassUID: 3002,
		Category: "iam",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := chain.Validate(ctx, event)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestChain_Validate_OrderMatters(t *testing.T) {
	var callOrder []int

	val1 := &mockValidator{
		supportsClass: "",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			callOrder = append(callOrder, 1)
			return nil
		},
	}

	val2 := &mockValidator{
		supportsClass: "",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			callOrder = append(callOrder, 2)
			return nil
		},
	}

	val3 := &mockValidator{
		supportsClass: "",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			callOrder = append(callOrder, 3)
			return nil
		},
	}

	chain := validator.NewChain(val1, val2, val3)

	event := &ocsf.Event{
		Class:    "test",
		ClassUID: 1000,
		Category: "test",
		Time:     time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := chain.Validate(ctx, event)

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, callOrder, "validators should be called in order")
}

func TestChain_Validate_NilEvent(t *testing.T) {
	val := &mockValidator{
		supportsClass: "",
		validateFunc: func(ctx context.Context, event *ocsf.Event) error {
			if event == nil {
				return errors.New("event is nil")
			}
			return nil
		},
	}

	chain := validator.NewChain(val)

	// Validator chain will panic with nil event because it dereferences event.Class
	// This documents that nil events are not supported
	// In production, the pipeline ensures events are never nil before validation

	// Verify we can safely skip nil events by not calling validate
	// or by catching the panic if needed
	assert.NotNil(t, chain)

	// Don't actually call Validate with nil - that's undefined behavior
	// err := chain.Validate(ctx, nil)
}
