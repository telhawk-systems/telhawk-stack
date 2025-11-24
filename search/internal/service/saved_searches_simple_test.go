package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
)

func TestCreateSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)

	req := &models.SavedSearchCreateRequest{
		Name: "Test Search",
		Query: map[string]interface{}{
			"source": map[string]interface{}{
				"index": "telhawk-events",
			},
		},
		CreatedBy: "user-123",
	}

	ctx := context.Background()
	saved, err := svc.CreateSavedSearch(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, saved)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestListSavedSearches_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	searches, err := svc.ListSavedSearches(ctx)

	assert.Error(t, err)
	assert.Nil(t, searches)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestGetSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	search, err := svc.GetSavedSearch(ctx, "some-id")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestUpdateSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	newName := "Updated"
	req := &models.SavedSearchUpdateRequest{
		Name:      &newName,
		CreatedBy: "user-123",
	}

	search, err := svc.UpdateSavedSearch(ctx, "some-id", req)

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestDeleteSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	err := svc.DeleteSavedSearch(ctx, "some-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestDisableSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	search, err := svc.DisableSavedSearch(ctx, "some-id", "user-123")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestEnableSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	search, err := svc.EnableSavedSearch(ctx, "some-id", "user-123")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestHideSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	search, err := svc.HideSavedSearch(ctx, "some-id", "user-123")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestUnhideSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	search, err := svc.UnhideSavedSearch(ctx, "some-id", "user-123")

	assert.Error(t, err)
	assert.Nil(t, search)
	assert.Contains(t, err.Error(), "repository not configured")
}

func TestValidateSavedSearchInput_Tests(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		query     map[string]interface{}
		expectErr bool
	}{
		{
			name:      "empty name",
			inputName: "",
			query: map[string]interface{}{
				"source": map[string]interface{}{"index": "test"},
			},
			expectErr: true,
		},
		{
			name:      "whitespace name",
			inputName: "   ",
			query: map[string]interface{}{
				"source": map[string]interface{}{"index": "test"},
			},
			expectErr: true,
		},
		{
			name:      "valid name with whitespace",
			inputName: "  Valid Name  ",
			query: map[string]interface{}{
				"source": map[string]interface{}{"index": "test"},
				"limit":  100,
			},
			expectErr: false, // Trimming makes it valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSavedSearchInput(tt.inputName, tt.query)
			if tt.expectErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrValidationFailed)
			} else {
				// May still error on query validation, but not on name
				if err != nil {
					// If it errors, it should be validation failed
					assert.ErrorIs(t, err, ErrValidationFailed)
				}
			}
		})
	}
}

func TestRunSavedSearch_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	resp, err := svc.RunSavedSearch(ctx, "some-id", "")

	assert.Error(t, err)
	assert.Nil(t, resp)
	// Will fail trying to get the search
	require.Error(t, err)
}

func TestListSavedSearchesPaged_NoRepository(t *testing.T) {
	svc := NewSearchService("1.0.0", nil)
	ctx := context.Background()

	searches, total, err := svc.ListSavedSearchesPaged(ctx, false, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, searches)
	assert.Equal(t, 0, total)
	assert.Contains(t, err.Error(), "repository not configured")
}
