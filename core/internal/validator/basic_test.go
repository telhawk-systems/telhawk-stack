package validator_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

func TestBasicValidator_Supports(t *testing.T) {
	val := validator.BasicValidator{}

	testCases := []struct {
		name     string
		class    string
		expected bool
	}{
		{
			name:     "non-empty class",
			class:    "authentication",
			expected: true,
		},
		{
			name:     "another class",
			class:    "network_activity",
			expected: true,
		},
		{
			name:     "empty class",
			class:    "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := val.Supports(tc.class)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBasicValidator_Validate_Success(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test Product",
				Vendor: "Test Vendor",
			},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.NoError(t, err)
}

func TestBasicValidator_Validate_MissingClass(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing class")
}

func TestBasicValidator_Validate_MissingClassUID(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    0,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing class_uid")
}

func TestBasicValidator_Validate_BaseEventWithZeroClassUID(t *testing.T) {
	val := validator.BasicValidator{}

	// base_event is allowed to have ClassUID of 0
	event := &ocsf.Event{
		Class:       "base_event",
		ClassUID:    0,
		Category:    "other",
		CategoryUID: 0,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.NoError(t, err, "base_event should allow ClassUID of 0")
}

func TestBasicValidator_Validate_MissingCategory(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing category")
}

func TestBasicValidator_Validate_MissingTime(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing time")
}

func TestBasicValidator_Validate_MissingProductName(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Vendor: "Test Vendor",
			},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.product.name")
}

func TestBasicValidator_Validate_MissingMetadataVersion(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test Product",
				Vendor: "Test Vendor",
			},
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.version")
}

func TestBasicValidator_Validate_AllFieldsPresent(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		ActivityID:  1,
		TypeUID:     300201,
		Time:        time.Now(),
		SeverityID:  1,
		Severity:    "Informational",
		StatusID:    1,
		Status:      "Success",
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test Product",
				Vendor: "Test Vendor",
			},
			Version: "1.1.0",
		},
		Properties: map[string]string{
			"source": "test",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.NoError(t, err, "all required fields are present")
}

func TestBasicValidator_Validate_MultipleErrors(t *testing.T) {
	val := validator.BasicValidator{}

	// Missing multiple required fields
	event := &ocsf.Event{
		ClassUID: 3002,
		// Missing Class, Category, Time, Metadata
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	// First missing field error should be returned
	assert.NotEmpty(t, err.Error())
}

func TestBasicValidator_Validate_OptionalFieldsMissing(t *testing.T) {
	val := validator.BasicValidator{}

	// Only required fields, no optional fields
	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test",
				Vendor: "Test",
			},
			Version: "1.0.0",
		},
		// Missing optional: ActivityID, TypeUID, SeverityID, StatusID, Properties, etc.
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.NoError(t, err, "optional fields should not be required")
}

func TestBasicValidator_Validate_ZeroTime(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Time{}, // zero time
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing time")
}

func TestBasicValidator_Validate_EmptyStrings(t *testing.T) {
	val := validator.BasicValidator{}

	testCases := []struct {
		name    string
		event   *ocsf.Event
		wantErr string
	}{
		{
			name: "empty class",
			event: &ocsf.Event{
				Class:       "",
				ClassUID:    3002,
				Category:    "iam",
				CategoryUID: 3,
				Time:        time.Now(),
				Metadata: ocsf.Metadata{
					Product: ocsf.Product{Name: "Test", Vendor: "Test"},
					Version: "1.0.0",
				},
			},
			wantErr: "missing class",
		},
		{
			name: "empty category",
			event: &ocsf.Event{
				Class:       "authentication",
				ClassUID:    3002,
				Category:    "",
				CategoryUID: 3,
				Time:        time.Now(),
				Metadata: ocsf.Metadata{
					Product: ocsf.Product{Name: "Test", Vendor: "Test"},
					Version: "1.0.0",
				},
			},
			wantErr: "missing category",
		},
		{
			name: "empty product name",
			event: &ocsf.Event{
				Class:       "authentication",
				ClassUID:    3002,
				Category:    "iam",
				CategoryUID: 3,
				Time:        time.Now(),
				Metadata: ocsf.Metadata{
					Product: ocsf.Product{Name: "", Vendor: "Test"},
					Version: "1.0.0",
				},
			},
			wantErr: "metadata.product.name",
		},
		{
			name: "empty metadata version",
			event: &ocsf.Event{
				Class:       "authentication",
				ClassUID:    3002,
				Category:    "iam",
				CategoryUID: 3,
				Time:        time.Now(),
				Metadata: ocsf.Metadata{
					Product: ocsf.Product{Name: "Test", Vendor: "Test"},
					Version: "",
				},
			},
			wantErr: "metadata.version",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			err := val.Validate(ctx, tc.event)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestBasicValidator_Validate_ContextIgnored(t *testing.T) {
	val := validator.BasicValidator{}

	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{Name: "Test", Vendor: "Test"},
			Version: "1.0.0",
		},
	}

	// BasicValidator doesn't use context, so cancelled context should be ignored
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := val.Validate(ctx, event)

	assert.NoError(t, err, "BasicValidator should not check context")
}

func TestBasicValidator_Validate_VendorOptional(t *testing.T) {
	val := validator.BasicValidator{}

	// Vendor is not required by BasicValidator
	event := &ocsf.Event{
		Class:       "authentication",
		ClassUID:    3002,
		Category:    "iam",
		CategoryUID: 3,
		Time:        time.Now(),
		Metadata: ocsf.Metadata{
			Product: ocsf.Product{
				Name:   "Test Product",
				Vendor: "", // empty vendor
			},
			Version: "1.0.0",
		},
	}

	ctx := context.Background()
	err := val.Validate(ctx, event)

	assert.NoError(t, err, "vendor should be optional")
}
