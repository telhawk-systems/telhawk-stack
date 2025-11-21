package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

func TestListDashboards_Default(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	resp, err := svc.ListDashboards(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	// Service is initialized with one default dashboard
	assert.GreaterOrEqual(t, len(resp.Dashboards), 1)
}

func TestListDashboards_Sorted(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Add multiple dashboards manually
	svc.mu.Lock()
	svc.dashboards["zebra-dashboard"] = models.Dashboard{
		ID:          "zebra-dashboard",
		Name:        "Zebra Dashboard",
		Description: "Last in alphabetical order",
		Widgets:     []map[string]interface{}{},
	}
	svc.dashboards["alpha-dashboard"] = models.Dashboard{
		ID:          "alpha-dashboard",
		Name:        "Alpha Dashboard",
		Description: "First in alphabetical order",
		Widgets:     []map[string]interface{}{},
	}
	svc.dashboards["bravo-dashboard"] = models.Dashboard{
		ID:          "bravo-dashboard",
		Name:        "Bravo Dashboard",
		Description: "Second in alphabetical order",
		Widgets:     []map[string]interface{}{},
	}
	svc.mu.Unlock()

	resp, err := svc.ListDashboards(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Check they're sorted by name
	for i := 1; i < len(resp.Dashboards); i++ {
		assert.LessOrEqual(t, resp.Dashboards[i-1].Name, resp.Dashboards[i].Name,
			"dashboards should be sorted by name")
	}
}

func TestGetDashboard_Success(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Add a custom dashboard
	testDashboard := models.Dashboard{
		ID:          "test-dashboard",
		Name:        "Test Dashboard",
		Description: "A dashboard for testing",
		Widgets: []map[string]interface{}{
			{
				"id":    "widget-1",
				"type":  "chart",
				"title": "Test Widget",
			},
		},
	}

	svc.mu.Lock()
	svc.dashboards["test-dashboard"] = testDashboard
	svc.mu.Unlock()

	// Get the dashboard
	retrieved, err := svc.GetDashboard(ctx, "test-dashboard")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "test-dashboard", retrieved.ID)
	assert.Equal(t, "Test Dashboard", retrieved.Name)
	assert.Equal(t, "A dashboard for testing", retrieved.Description)
	assert.Len(t, retrieved.Widgets, 1)
}

func TestGetDashboard_NotFound(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	dashboard, err := svc.GetDashboard(ctx, "nonexistent-dashboard-id")

	assert.Error(t, err)
	assert.Nil(t, dashboard)
	assert.ErrorIs(t, err, ErrDashboardNotFound)
}

func TestGetDashboard_DefaultThreatOverview(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// The default "threat-overview" dashboard should exist
	dashboard, err := svc.GetDashboard(ctx, "threat-overview")

	require.NoError(t, err)
	require.NotNil(t, dashboard)
	assert.Equal(t, "threat-overview", dashboard.ID)
	assert.Equal(t, "Threat Overview", dashboard.Name)
	assert.NotEmpty(t, dashboard.Description)
	assert.Greater(t, len(dashboard.Widgets), 0, "default dashboard should have widgets")
}

func TestDashboardWidgets_Structure(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Create a dashboard with various widget types
	testDashboard := models.Dashboard{
		ID:          "widget-test",
		Name:        "Widget Test Dashboard",
		Description: "Testing widget structures",
		Widgets: []map[string]interface{}{
			{
				"id":      "bar-chart",
				"type":    "bar",
				"title":   "Bar Chart Widget",
				"query":   "index=ocsf stats count by severity",
				"display": map[string]interface{}{"palette": "risk"},
			},
			{
				"id":      "line-chart",
				"type":    "line",
				"title":   "Line Chart Widget",
				"query":   "index=ocsf timechart count",
				"display": map[string]interface{}{"palette": "default"},
			},
			{
				"id":      "table-widget",
				"type":    "table",
				"title":   "Table Widget",
				"query":   "index=ocsf | head 10",
				"columns": []string{"timestamp", "severity", "message"},
			},
			{
				"id":      "metric-widget",
				"type":    "metric",
				"title":   "Metric Widget",
				"query":   "index=ocsf stats count",
				"display": map[string]interface{}{"unit": "events"},
			},
		},
	}

	svc.mu.Lock()
	svc.dashboards["widget-test"] = testDashboard
	svc.mu.Unlock()

	// Retrieve and validate
	retrieved, err := svc.GetDashboard(ctx, "widget-test")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Len(t, retrieved.Widgets, 4)

	// Check each widget type
	widgetTypes := make(map[string]bool)
	for _, widget := range retrieved.Widgets {
		widgetType, ok := widget["type"].(string)
		require.True(t, ok, "widget should have a type field")
		widgetTypes[widgetType] = true

		// All widgets should have id and title
		assert.Contains(t, widget, "id")
		assert.Contains(t, widget, "title")
		assert.Contains(t, widget, "query")
	}

	assert.Contains(t, widgetTypes, "bar")
	assert.Contains(t, widgetTypes, "line")
	assert.Contains(t, widgetTypes, "table")
	assert.Contains(t, widgetTypes, "metric")
}

func TestRequestExport_Success(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	req := &models.ExportRequest{
		Query:               "severity:high",
		Format:              "csv",
		Compression:         "gzip",
		NotificationChannel: "email@example.com",
	}

	resp, err := svc.RequestExport(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.ExportID)
	assert.Equal(t, "pending", resp.Status)
	assert.True(t, resp.ExpiresAt.After(time.Now()))
}

func TestRequestExport_WithTimeRange(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	req := &models.ExportRequest{
		Query: "severity:high",
		TimeRange: &models.TimeRange{
			From: from,
			To:   to,
		},
		Format: "json",
	}

	resp, err := svc.RequestExport(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.ExportID)
	assert.Equal(t, "pending", resp.Status)
}

func TestRequestExport_UniqueIDs(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	req := &models.ExportRequest{
		Query:  "severity:high",
		Format: "csv",
	}

	// Create multiple export requests
	exports := make(map[string]bool)
	for i := 0; i < 10; i++ {
		resp, err := svc.RequestExport(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Check ID is unique
		assert.False(t, exports[resp.ExportID], "export IDs should be unique")
		exports[resp.ExportID] = true
	}

	assert.Len(t, exports, 10)
}

func TestRequestExport_ExpirationTime(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	req := &models.ExportRequest{
		Query:  "severity:high",
		Format: "csv",
	}

	before := time.Now().UTC()
	resp, err := svc.RequestExport(ctx, req)
	after := time.Now().UTC()

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Expiration should be approximately 1 hour from now
	expectedMin := before.Add(55 * time.Minute)
	expectedMax := after.Add(65 * time.Minute)

	assert.True(t, resp.ExpiresAt.After(expectedMin),
		"expiration should be at least 55 minutes from now")
	assert.True(t, resp.ExpiresAt.Before(expectedMax),
		"expiration should be no more than 65 minutes from now")
}

func TestRequestExport_AllFormats(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	formats := []string{"csv", "json", "parquet"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			req := &models.ExportRequest{
				Query:  "severity:high",
				Format: format,
			}

			resp, err := svc.RequestExport(ctx, req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.NotEmpty(t, resp.ExportID)
		})
	}
}

func TestRequestExport_AllCompressions(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	compressions := []string{"", "gzip", "bzip2", "zstd"}

	for _, compression := range compressions {
		t.Run("compression_"+compression, func(t *testing.T) {
			req := &models.ExportRequest{
				Query:       "severity:high",
				Format:      "csv",
				Compression: compression,
			}

			resp, err := svc.RequestExport(ctx, req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.NotEmpty(t, resp.ExportID)
		})
	}
}

func TestDashboardConcurrency(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Add a test dashboard
	testDashboard := models.Dashboard{
		ID:          "concurrent-test",
		Name:        "Concurrent Test Dashboard",
		Description: "Testing concurrent access",
		Widgets:     []map[string]interface{}{},
	}

	svc.mu.Lock()
	svc.dashboards["concurrent-test"] = testDashboard
	svc.mu.Unlock()

	// Simulate concurrent reads
	done := make(chan bool)
	errors := make(chan error, 20)

	// Concurrent GetDashboard calls
	for i := 0; i < 10; i++ {
		go func() {
			_, err := svc.GetDashboard(ctx, "concurrent-test")
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Concurrent ListDashboards calls
	for i := 0; i < 10; i++ {
		go func() {
			_, err := svc.ListDashboards(ctx)
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

func TestHealth_IncludesSchedulerMetrics(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	// Test without scheduler
	health := svc.Health(ctx)
	require.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "1.0.0", health.Version)
	assert.GreaterOrEqual(t, health.UptimeSeconds, int64(0))
	assert.Nil(t, health.Scheduler, "scheduler metrics should be nil when no scheduler")
}

func TestValidateToken_NoAuthClient(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)
	ctx := context.Background()

	userID, err := svc.ValidateToken(ctx, "some-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
	assert.Contains(t, err.Error(), "auth client not configured")
}

func TestNewQueryService_Initialization(t *testing.T) {
	version := "2.0.0-test"
	svc := NewQueryService(version, nil)

	require.NotNil(t, svc)
	assert.Equal(t, version, svc.version)
	assert.NotZero(t, svc.startedAt)
	assert.NotNil(t, svc.alerts)
	assert.NotNil(t, svc.dashboards)

	// Check that default data is loaded
	assert.Greater(t, len(svc.alerts), 0, "should have default alerts")
	assert.Greater(t, len(svc.dashboards), 0, "should have default dashboards")
}

func TestWithDependencies(t *testing.T) {
	svc := NewQueryService("1.0.0", nil)

	assert.Nil(t, svc.repo)
	assert.Nil(t, svc.authClient)

	// Wire dependencies (would need actual repo implementation in real usage)
	// For now, just verify the service was created correctly
	assert.NotNil(t, svc)
}
