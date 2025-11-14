package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDetectionSchema_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		schema     *DetectionSchema
		wantActive bool
	}{
		{
			name: "active schema - no disabled or hidden timestamps",
			schema: &DetectionSchema{
				ID:         "test-id",
				VersionID:  "test-version-id",
				DisabledAt: nil,
				HiddenAt:   nil,
			},
			wantActive: true,
		},
		{
			name: "disabled schema - disabled_at is set",
			schema: &DetectionSchema{
				ID:         "test-id",
				VersionID:  "test-version-id",
				DisabledAt: &now,
				HiddenAt:   nil,
			},
			wantActive: false,
		},
		{
			name: "hidden schema - hidden_at is set",
			schema: &DetectionSchema{
				ID:         "test-id",
				VersionID:  "test-version-id",
				DisabledAt: nil,
				HiddenAt:   &now,
			},
			wantActive: false,
		},
		{
			name: "disabled and hidden - both timestamps set",
			schema: &DetectionSchema{
				ID:         "test-id",
				VersionID:  "test-version-id",
				DisabledAt: &now,
				HiddenAt:   &now,
			},
			wantActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.schema.IsActive()
			assert.Equal(t, tt.wantActive, got)
		})
	}
}
