package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

func TestListAlerts_EmptyService(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	resp, err := svc.ListAlerts(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	// Service is initialized with one default alert
	assert.GreaterOrEqual(t, len(resp.Alerts), 1)
}

func TestListAlerts_Sorted(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Add multiple alerts
	alerts := []models.AlertRequest{
		{
			Name:     "Zebra Alert",
			Query:    "severity:high",
			Severity: "high",
			Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		},
		{
			Name:     "Alpha Alert",
			Query:    "severity:critical",
			Severity: "critical",
			Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		},
		{
			Name:     "Bravo Alert",
			Query:    "severity:medium",
			Severity: "medium",
			Schedule: models.AlertSchedule{IntervalMinutes: 10, LookbackMinutes: 30},
		},
	}

	for _, req := range alerts {
		_, _, err := svc.UpsertAlert(ctx, &req)
		require.NoError(t, err)
	}

	// List alerts
	resp, err := svc.ListAlerts(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Check they're sorted by name
	for i := 1; i < len(resp.Alerts); i++ {
		assert.LessOrEqual(t, resp.Alerts[i-1].Name, resp.Alerts[i].Name,
			"alerts should be sorted by name")
	}
}

func TestUpsertAlert_Create(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	req := &models.AlertRequest{
		Name:        "New Alert",
		Description: "Test alert description",
		Query:       "severity:high AND user:admin",
		Severity:    "high",
		Schedule: models.AlertSchedule{
			IntervalMinutes: 5,
			LookbackMinutes: 15,
		},
		Owner: "soc-team@example.com",
	}

	alert, created, err := svc.UpsertAlert(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, alert)
	assert.True(t, created, "should indicate alert was created")
	assert.NotEmpty(t, alert.ID, "ID should be generated")
	assert.Equal(t, "New Alert", alert.Name)
	assert.Equal(t, "Test alert description", alert.Description)
	assert.Equal(t, "high", alert.Severity)
	assert.Equal(t, "active", alert.Status, "status should default to 'active'")
	assert.Nil(t, alert.LastTriggeredAt)
}

func TestUpsertAlert_Update(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	createReq := &models.AlertRequest{
		Name:     "Original Alert",
		Query:    "severity:medium",
		Severity: "medium",
		Schedule: models.AlertSchedule{IntervalMinutes: 10, LookbackMinutes: 30},
	}

	created, _, err := svc.UpsertAlert(ctx, createReq)
	require.NoError(t, err)

	// Update the alert
	updateReq := &models.AlertRequest{
		ID:          &created.ID,
		Name:        "Updated Alert",
		Description: "Updated description",
		Query:       "severity:high",
		Severity:    "high",
		Schedule:    models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		Status:      "paused",
		Owner:       "new-owner@example.com",
	}

	updated, wasCreated, err := svc.UpsertAlert(ctx, updateReq)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.False(t, wasCreated, "should indicate alert was updated, not created")
	assert.Equal(t, created.ID, updated.ID, "ID should remain the same")
	assert.Equal(t, "Updated Alert", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, "high", updated.Severity)
	assert.Equal(t, "paused", updated.Status)
	assert.Equal(t, "new-owner@example.com", updated.Owner)
}

func TestUpsertAlert_PreservesLastTriggered(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	createReq := &models.AlertRequest{
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
	}

	created, _, err := svc.UpsertAlert(ctx, createReq)
	require.NoError(t, err)

	// Set last triggered time
	now := time.Now().UTC()
	err = svc.UpdateLastTriggered(ctx, created.ID, now)
	require.NoError(t, err)

	// Update the alert
	updateReq := &models.AlertRequest{
		ID:       &created.ID,
		Name:     "Updated Alert",
		Query:    "severity:critical",
		Severity: "critical",
		Schedule: models.AlertSchedule{IntervalMinutes: 3, LookbackMinutes: 10},
	}

	updated, _, err := svc.UpsertAlert(ctx, updateReq)
	require.NoError(t, err)

	// LastTriggeredAt should be preserved
	require.NotNil(t, updated.LastTriggeredAt)
	assert.Equal(t, now.Unix(), updated.LastTriggeredAt.Unix())
}

func TestUpsertAlert_DefaultStatus(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	req := &models.AlertRequest{
		Name:     "Alert Without Status",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		// Status not set
	}

	alert, _, err := svc.UpsertAlert(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, alert)
	assert.Equal(t, "active", alert.Status, "status should default to 'active'")
}

func TestGetAlert_Success(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	req := &models.AlertRequest{
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
	}

	created, _, err := svc.UpsertAlert(ctx, req)
	require.NoError(t, err)

	// Get the alert
	retrieved, err := svc.GetAlert(ctx, created.ID)

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
	assert.Equal(t, created.Severity, retrieved.Severity)
}

func TestGetAlert_NotFound(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	alert, err := svc.GetAlert(ctx, "nonexistent-alert-id")

	assert.Error(t, err)
	assert.Nil(t, alert)
	assert.ErrorIs(t, err, ErrAlertNotFound)
}

func TestPatchAlert_Status(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	createReq := &models.AlertRequest{
		Name:     "Active Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		Status:   "active",
	}

	created, _, err := svc.UpsertAlert(ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, "active", created.Status)

	// Patch the status
	patchReq := &models.AlertPatchRequest{
		Status: "paused",
	}

	patched, err := svc.PatchAlert(ctx, created.ID, patchReq)

	require.NoError(t, err)
	require.NotNil(t, patched)
	assert.Equal(t, "paused", patched.Status)
	assert.Equal(t, created.Name, patched.Name, "other fields should remain unchanged")
}

func TestPatchAlert_Owner(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	createReq := &models.AlertRequest{
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		Owner:    "original-owner@example.com",
	}

	created, _, err := svc.UpsertAlert(ctx, createReq)
	require.NoError(t, err)

	// Patch the owner
	patchReq := &models.AlertPatchRequest{
		Owner: "new-owner@example.com",
	}

	patched, err := svc.PatchAlert(ctx, created.ID, patchReq)

	require.NoError(t, err)
	require.NotNil(t, patched)
	assert.Equal(t, "new-owner@example.com", patched.Owner)
}

func TestPatchAlert_BothFields(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	createReq := &models.AlertRequest{
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
		Status:   "active",
		Owner:    "original@example.com",
	}

	created, _, err := svc.UpsertAlert(ctx, createReq)
	require.NoError(t, err)

	// Patch both fields
	patchReq := &models.AlertPatchRequest{
		Status: "paused",
		Owner:  "new@example.com",
	}

	patched, err := svc.PatchAlert(ctx, created.ID, patchReq)

	require.NoError(t, err)
	require.NotNil(t, patched)
	assert.Equal(t, "paused", patched.Status)
	assert.Equal(t, "new@example.com", patched.Owner)
}

func TestPatchAlert_NotFound(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	patchReq := &models.AlertPatchRequest{
		Status: "paused",
	}

	patched, err := svc.PatchAlert(ctx, "nonexistent-id", patchReq)

	assert.Error(t, err)
	assert.Nil(t, patched)
	assert.ErrorIs(t, err, ErrAlertNotFound)
}

func TestUpdateLastTriggered_Success(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	req := &models.AlertRequest{
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
	}

	created, _, err := svc.UpsertAlert(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, created.LastTriggeredAt, "new alert should not have last triggered time")

	// Update last triggered
	now := time.Now().UTC()
	err = svc.UpdateLastTriggered(ctx, created.ID, now)
	require.NoError(t, err)

	// Verify it was updated
	retrieved, err := svc.GetAlert(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.LastTriggeredAt)
	assert.Equal(t, now.Unix(), retrieved.LastTriggeredAt.Unix())
}

func TestUpdateLastTriggered_NotFound(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	now := time.Now().UTC()
	err := svc.UpdateLastTriggered(ctx, "nonexistent-id", now)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAlertNotFound)
}

func TestUpdateLastTriggered_MultipleUpdates(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	req := &models.AlertRequest{
		Name:     "Test Alert",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
	}

	created, _, err := svc.UpsertAlert(ctx, req)
	require.NoError(t, err)

	// Update last triggered multiple times
	time1 := time.Now().UTC().Add(-2 * time.Hour)
	err = svc.UpdateLastTriggered(ctx, created.ID, time1)
	require.NoError(t, err)

	time2 := time.Now().UTC().Add(-1 * time.Hour)
	err = svc.UpdateLastTriggered(ctx, created.ID, time2)
	require.NoError(t, err)

	time3 := time.Now().UTC()
	err = svc.UpdateLastTriggered(ctx, created.ID, time3)
	require.NoError(t, err)

	// Verify the most recent time is set
	retrieved, err := svc.GetAlert(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.LastTriggeredAt)
	assert.Equal(t, time3.Unix(), retrieved.LastTriggeredAt.Unix())
}

func TestAlertSchedule_Validation(t *testing.T) {
	// Test various schedule configurations
	tests := []struct {
		name     string
		schedule models.AlertSchedule
		valid    bool
	}{
		{
			name: "valid schedule",
			schedule: models.AlertSchedule{
				IntervalMinutes: 5,
				LookbackMinutes: 15,
			},
			valid: true,
		},
		{
			name: "zero interval",
			schedule: models.AlertSchedule{
				IntervalMinutes: 0,
				LookbackMinutes: 15,
			},
			valid: true, // Service doesn't enforce validation
		},
		{
			name: "negative values",
			schedule: models.AlertSchedule{
				IntervalMinutes: -5,
				LookbackMinutes: -15,
			},
			valid: true, // Service doesn't enforce validation
		},
	}

	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &models.AlertRequest{
				Name:     "Schedule Test",
				Query:    "severity:high",
				Severity: "high",
				Schedule: tt.schedule,
			}

			alert, _, err := svc.UpsertAlert(ctx, req)
			if tt.valid {
				require.NoError(t, err)
				require.NotNil(t, alert)
				assert.Equal(t, tt.schedule.IntervalMinutes, alert.Schedule.IntervalMinutes)
				assert.Equal(t, tt.schedule.LookbackMinutes, alert.Schedule.LookbackMinutes)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestAlertConcurrency(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create an alert
	req := &models.AlertRequest{
		Name:     "Concurrent Test",
		Query:    "severity:high",
		Severity: "high",
		Schedule: models.AlertSchedule{IntervalMinutes: 5, LookbackMinutes: 15},
	}

	created, _, err := svc.UpsertAlert(ctx, req)
	require.NoError(t, err)

	// Simulate concurrent reads and writes
	done := make(chan bool)
	errors := make(chan error, 10)

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			_, err := svc.GetAlert(ctx, created.ID)
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Concurrent patch operations
	for i := 0; i < 5; i++ {
		go func(n int) {
			status := "active"
			if n%2 == 0 {
				status = "paused"
			}
			_, err := svc.PatchAlert(ctx, created.ID, &models.AlertPatchRequest{
				Status: status,
			})
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}

	// Verify alert is in a valid state
	final, err := svc.GetAlert(ctx, created.ID)
	require.NoError(t, err)
	assert.Contains(t, []string{"active", "paused"}, final.Status)
}
