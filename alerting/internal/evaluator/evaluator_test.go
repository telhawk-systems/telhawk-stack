package evaluator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/alerting/internal/correlation"
)

// mockRulesClient is a mock implementation of RulesClient
type mockRulesClient struct {
	listSchemasFunc func(ctx context.Context) ([]*DetectionSchema, error)
}

func (m *mockRulesClient) ListSchemas(ctx context.Context) ([]*DetectionSchema, error) {
	if m.listSchemasFunc != nil {
		return m.listSchemasFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

// mockStorageClient is a mock implementation of StorageClient
type mockStorageClient struct {
	fetchEventsFunc func(ctx context.Context, since time.Time) ([]*Event, error)
	storeAlertFunc  func(ctx context.Context, alert *Alert) error
}

func (m *mockStorageClient) FetchEvents(ctx context.Context, since time.Time) ([]*Event, error) {
	if m.fetchEventsFunc != nil {
		return m.fetchEventsFunc(ctx, since)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStorageClient) StoreAlert(ctx context.Context, alert *Alert) error {
	if m.storeAlertFunc != nil {
		return m.storeAlertFunc(ctx, alert)
	}
	return errors.New("not implemented")
}

func TestNewEvaluator(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)          // nil redis, disabled
	queryExecutor := correlation.NewQueryExecutor("", "", "", false) // empty config is ok for test

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	require.NotNil(t, eval)
	assert.NotNil(t, eval.rulesClient)
	assert.NotNil(t, eval.storageClient)
	assert.NotNil(t, eval.stateManager)
	assert.NotNil(t, eval.queryExecutor)
	assert.NotNil(t, eval.eventCountEval)
	assert.NotNil(t, eval.valueCountEval)
	assert.NotNil(t, eval.temporalEval)
	assert.NotNil(t, eval.temporalOrderedEval)
	assert.NotNil(t, eval.joinEval)

	// Verify lastEvalTime is set to 5 minutes in the past
	expectedTime := time.Now().Add(-5 * time.Minute)
	timeDiff := eval.lastEvalTime.Sub(expectedTime)
	assert.Less(t, timeDiff.Abs(), 1*time.Second)
}

func TestEvaluate_NoEvents(t *testing.T) {
	mockRules := &mockRulesClient{
		listSchemasFunc: func(ctx context.Context) ([]*DetectionSchema, error) {
			return []*DetectionSchema{}, nil
		},
	}
	mockStorage := &mockStorageClient{
		fetchEventsFunc: func(ctx context.Context, since time.Time) ([]*Event, error) {
			return []*Event{}, nil
		},
	}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)
	initialTime := eval.lastEvalTime

	eval.evaluate(context.Background())

	// lastEvalTime should be updated even with no events
	assert.True(t, eval.lastEvalTime.After(initialTime))
}

func TestEvaluate_FetchEventsError(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{
		fetchEventsFunc: func(ctx context.Context, since time.Time) ([]*Event, error) {
			return nil, errors.New("fetch error")
		},
	}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)
	initialTime := eval.lastEvalTime

	eval.evaluate(context.Background())

	// lastEvalTime should not be updated on error
	assert.Equal(t, initialTime, eval.lastEvalTime)
}

func TestEvaluate_FetchSchemasError(t *testing.T) {
	mockRules := &mockRulesClient{
		listSchemasFunc: func(ctx context.Context) ([]*DetectionSchema, error) {
			return nil, errors.New("schema fetch error")
		},
	}
	mockStorage := &mockStorageClient{
		fetchEventsFunc: func(ctx context.Context, since time.Time) ([]*Event, error) {
			return []*Event{
				{ID: "event-1", Index: "test-index"},
			}, nil
		},
	}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)
	initialTime := eval.lastEvalTime

	eval.evaluate(context.Background())

	// lastEvalTime should not be updated on schema fetch error
	assert.Equal(t, initialTime, eval.lastEvalTime)
}

func TestEvaluate_DisabledSchema(t *testing.T) {
	alertStored := false

	mockRules := &mockRulesClient{
		listSchemasFunc: func(ctx context.Context) ([]*DetectionSchema, error) {
			return []*DetectionSchema{
				{
					ID:        "schema-1",
					VersionID: "version-1",
					Disabled:  true, // Disabled schema
					Model: map[string]interface{}{
						"correlation_type": "event_count",
					},
					View: map[string]interface{}{
						"title":    "Disabled Rule",
						"severity": "high",
					},
					Controller: map[string]interface{}{},
				},
			}, nil
		},
	}
	mockStorage := &mockStorageClient{
		fetchEventsFunc: func(ctx context.Context, since time.Time) ([]*Event, error) {
			return []*Event{
				{ID: "event-1", Index: "test-index"},
			}, nil
		},
		storeAlertFunc: func(ctx context.Context, alert *Alert) error {
			alertStored = true
			return nil
		},
	}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)
	eval.evaluate(context.Background())

	// No alert should be stored for disabled schema
	assert.False(t, alertStored)
}

func TestEvaluateSimpleRule(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	tests := []struct {
		name           string
		schema         *DetectionSchema
		events         []*Event
		expectedAlerts int
	}{
		{
			name: "simple rule with no matches",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Model:     map[string]interface{}{},
				View: map[string]interface{}{
					"title":       "Test Rule",
					"description": "Test Description",
					"severity":    "high",
				},
				Controller: map[string]interface{}{
					"detection": map[string]interface{}{
						"query": "test query",
					},
				},
			},
			events: []*Event{
				{
					ID:    "event-1",
					Index: "test-index",
					Source: map[string]interface{}{
						"field": "value",
					},
				},
			},
			expectedAlerts: 0, // matchesController always returns false for now
		},
		{
			name: "simple rule with invalid controller format",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Model:     map[string]interface{}{},
				View: map[string]interface{}{
					"title":    "Test Rule",
					"severity": "high",
				},
				Controller: map[string]interface{}{
					"detection": "invalid", // Not a map
				},
			},
			events: []*Event{
				{ID: "event-1", Index: "test-index"},
			},
			expectedAlerts: 0,
		},
		{
			name: "simple rule with default severity",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Model:     map[string]interface{}{},
				View: map[string]interface{}{
					"title": "Test Rule",
					// No severity specified
				},
				Controller: map[string]interface{}{
					"detection": map[string]interface{}{},
				},
			},
			events: []*Event{
				{ID: "event-1", Index: "test-index"},
			},
			expectedAlerts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts := eval.evaluateSimpleRule(context.Background(), tt.schema, tt.events)
			assert.Len(t, alerts, tt.expectedAlerts)
		})
	}
}

func TestConvertCorrelationAlerts(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	now := time.Now()
	corrAlerts := []*correlation.Alert{
		{
			RuleID:        "rule-1",
			RuleVersionID: "version-1",
			Severity:      "high",
			Title:         "Test Alert 1",
			Description:   "Test Description 1",
			Time:          now,
			Metadata: map[string]interface{}{
				"key1": "value1",
			},
		},
		{
			RuleID:        "rule-2",
			RuleVersionID: "version-2",
			Severity:      "critical",
			Title:         "Test Alert 2",
			Description:   "Test Description 2",
			Time:          now.Add(1 * time.Minute),
			Metadata: map[string]interface{}{
				"key2": "value2",
			},
		},
	}

	alerts := eval.convertCorrelationAlerts(corrAlerts)

	require.Len(t, alerts, 2)

	assert.Equal(t, "rule-1", alerts[0].DetectionSchemaID)
	assert.Equal(t, "version-1", alerts[0].DetectionSchemaVersionID)
	assert.Equal(t, "high", alerts[0].Severity)
	assert.Equal(t, "Test Alert 1", alerts[0].Title)
	assert.Equal(t, "Test Description 1", alerts[0].Description)
	assert.Equal(t, now, alerts[0].Timestamp)
	assert.Equal(t, "value1", alerts[0].Metadata["key1"])

	assert.Equal(t, "rule-2", alerts[1].DetectionSchemaID)
	assert.Equal(t, "version-2", alerts[1].DetectionSchemaVersionID)
	assert.Equal(t, "critical", alerts[1].Severity)
	assert.Equal(t, "Test Alert 2", alerts[1].Title)
	assert.Equal(t, "Test Description 2", alerts[1].Description)
	assert.Equal(t, now.Add(1*time.Minute), alerts[1].Timestamp)
	assert.Equal(t, "value2", alerts[1].Metadata["key2"])
}

func TestApplySuppression(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	now := time.Now()

	tests := []struct {
		name            string
		schema          *DetectionSchema
		alerts          []*correlation.Alert
		expectedCount   int
		secondCallCount int
		description     string
	}{
		{
			name: "no suppression configured",
			schema: &DetectionSchema{
				ID:         "schema-1",
				VersionID:  "version-1",
				Controller: map[string]interface{}{},
			},
			alerts: []*correlation.Alert{
				{RuleID: "schema-1", Time: now},
				{RuleID: "schema-1", Time: now.Add(1 * time.Minute)},
			},
			expectedCount:   2,
			secondCallCount: -1,
			description:     "all alerts should pass through",
		},
		{
			name: "suppression disabled",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Controller: map[string]interface{}{
					"suppression": map[string]interface{}{
						"enabled": false,
					},
				},
			},
			alerts: []*correlation.Alert{
				{RuleID: "schema-1", Time: now},
				{RuleID: "schema-1", Time: now.Add(1 * time.Minute)},
			},
			expectedCount:   2,
			secondCallCount: -1,
			description:     "all alerts should pass through when disabled",
		},
		{
			name: "suppression enabled with valid config (no redis)",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Controller: map[string]interface{}{
					"suppression": map[string]interface{}{
						"enabled":    true,
						"window":     "5m",
						"max_alerts": 1,
						"key":        []interface{}{"src_ip"},
					},
				},
			},
			alerts: []*correlation.Alert{
				{
					RuleID: "schema-1",
					Time:   now,
					Metadata: map[string]interface{}{
						"src_ip": "192.168.1.1",
					},
				},
				{
					RuleID: "schema-1",
					Time:   now.Add(1 * time.Minute),
					Metadata: map[string]interface{}{
						"src_ip": "192.168.1.1",
					},
				},
			},
			expectedCount:   2,  // Without redis, suppression doesn't work
			secondCallCount: -1, // Don't test second call
			description:     "suppression requires redis (passes through without it)",
		},
		{
			name: "suppression with different keys",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Controller: map[string]interface{}{
					"suppression": map[string]interface{}{
						"enabled":    true,
						"window":     "5m",
						"max_alerts": 1,
						"key":        []interface{}{"src_ip"},
					},
				},
			},
			alerts: []*correlation.Alert{
				{
					RuleID: "schema-1",
					Time:   now,
					Metadata: map[string]interface{}{
						"src_ip": "192.168.1.1",
					},
				},
				{
					RuleID: "schema-1",
					Time:   now.Add(1 * time.Minute),
					Metadata: map[string]interface{}{
						"src_ip": "192.168.1.2", // Different IP
					},
				},
			},
			expectedCount:   2,
			secondCallCount: -1,
			description:     "alerts with different keys should not be suppressed",
		},
		{
			name: "suppression with invalid window",
			schema: &DetectionSchema{
				ID:        "schema-1",
				VersionID: "version-1",
				Controller: map[string]interface{}{
					"suppression": map[string]interface{}{
						"enabled":    true,
						"window":     "invalid",
						"max_alerts": 1,
						"key":        []interface{}{"src_ip"},
					},
				},
			},
			alerts: []*correlation.Alert{
				{RuleID: "schema-1", Time: now},
				{RuleID: "schema-1", Time: now.Add(1 * time.Minute)},
			},
			expectedCount:   2,
			secondCallCount: -1,
			description:     "all alerts should pass through with invalid window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state manager for each test
			stateManager = correlation.NewStateManager(nil, false)
			eval.stateManager = stateManager

			unsuppressed := eval.applySuppression(context.Background(), tt.schema, tt.alerts)
			assert.Len(t, unsuppressed, tt.expectedCount, tt.description)

			// If secondCallCount is specified (>= 0), test suppression persistence
			// secondCallCount = -1 means skip this test
			if tt.secondCallCount >= 0 {
				secondCall := eval.applySuppression(context.Background(), tt.schema, tt.alerts)
				assert.Len(t, secondCall, tt.secondCallCount, "second call should suppress all alerts")
			}
		})
	}
}

func TestMatchesController(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	tests := []struct {
		name       string
		controller map[string]interface{}
		eventData  map[string]interface{}
		expected   bool
	}{
		{
			name: "valid query",
			controller: map[string]interface{}{
				"query": "test query",
			},
			eventData: map[string]interface{}{
				"field": "value",
			},
			expected: false, // Always false until query engine is implemented
		},
		{
			name: "missing query",
			controller: map[string]interface{}{
				"other": "data",
			},
			eventData: map[string]interface{}{
				"field": "value",
			},
			expected: false,
		},
		{
			name: "invalid query type",
			controller: map[string]interface{}{
				"query": 123, // Not a string
			},
			eventData: map[string]interface{}{
				"field": "value",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.matchesController(tt.controller, tt.eventData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateCorrelationRule_UnsupportedType(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	schema := &DetectionSchema{
		ID:        "schema-1",
		VersionID: "version-1",
		Model: map[string]interface{}{
			"correlation_type": "unsupported_type",
		},
		View: map[string]interface{}{
			"title":    "Test Rule",
			"severity": "high",
		},
		Controller: map[string]interface{}{},
	}

	alerts := eval.evaluateCorrelationRule(context.Background(), schema)

	assert.Empty(t, alerts)
}

func TestEvaluateCorrelationRule_InvalidCorrelationType(t *testing.T) {
	mockRules := &mockRulesClient{}
	mockStorage := &mockStorageClient{}
	stateManager := correlation.NewStateManager(nil, false)
	queryExecutor := correlation.NewQueryExecutor("", "", "", false)

	eval := NewEvaluator(mockRules, mockStorage, stateManager, queryExecutor)

	schema := &DetectionSchema{
		ID:        "schema-1",
		VersionID: "version-1",
		Model: map[string]interface{}{
			"correlation_type": 123, // Not a string
		},
		View: map[string]interface{}{
			"title":    "Test Rule",
			"severity": "high",
		},
		Controller: map[string]interface{}{},
	}

	alerts := eval.evaluateCorrelationRule(context.Background(), schema)

	assert.Empty(t, alerts)
}
