package correlation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock OpenSearch server
func setupMockOpenSearch(t *testing.T, response map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestEventCountEvaluator_Evaluate(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch response with aggregation
	mockResponse := map[string]interface{}{
		"took": 5,
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value": 150,
			},
			"hits": []interface{}{},
		},
		"aggregations": map[string]interface{}{
			"groups": map[string]interface{}{
				"buckets": []interface{}{
					map[string]interface{}{
						"key":       "alice",
						"doc_count": 15,
					},
					map[string]interface{}{
						"key":       "bob",
						"doc_count": 8,
					},
				},
			},
		},
	}

	server := setupMockOpenSearch(t, mockResponse)
	defer server.Close()

	stateManager := NewStateManager(redisClient, true)
	queryExecutor := NewQueryExecutor(server.URL, "admin", "password", true)
	evaluator := NewEventCountEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	t.Run("evaluate with threshold exceeded", func(t *testing.T) {
		schema := &DetectionSchema{
			ID:        "test-rule-123",
			VersionID: "version-456",
			Model: map[string]interface{}{
				"correlation_type": "event_count",
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"threshold":   10,
					"operator":    "gt",
					"group_by":    []interface{}{".actor.user.name"},
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"type": "and",
							"conditions": []interface{}{
								map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(3002),
								},
								map[string]interface{}{
									"field":    ".status_id",
									"operator": "eq",
									"value":    float64(2),
								},
							},
						},
					},
				},
			},
			View: map[string]interface{}{
				"title":       "Brute Force Detection",
				"severity":    "high",
				"description": "User {{actor.user.name}} had {{event_count}} failed logins",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"threshold": float64(10),
					"operator":  "gt",
				},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		assert.NotEmpty(t, alerts)

		// Should generate alert for alice (15 > 10)
		// Should not generate alert for bob (8 < 10)
		assert.Equal(t, 1, len(alerts))
		assert.Equal(t, "Brute Force Detection", alerts[0].Title)
		assert.Equal(t, "high", alerts[0].Severity)
		assert.Equal(t, int64(15), alerts[0].Metadata["event_count"])
	})

	t.Run("evaluate with no threshold exceeded", func(t *testing.T) {
		schema := &DetectionSchema{
			ID:        "test-rule-789",
			VersionID: "version-abc",
			Model: map[string]interface{}{
				"correlation_type": "event_count",
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"threshold":   100,
					"operator":    "gt",
					"group_by":    []interface{}{".actor.user.name"},
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    float64(3002),
						},
					},
				},
			},
			View: map[string]interface{}{
				"title":       "High Activity",
				"severity":    "low",
				"description": "Activity detected",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"threshold": float64(100), // Higher than any count
					"operator":  "gt",
				},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		assert.Empty(t, alerts) // No alerts since no count exceeds threshold
	})

	t.Run("evaluate with different operators", func(t *testing.T) {
		tests := []struct {
			name      string
			operator  string
			threshold float64
			expected  int
		}{
			{"gte operator", "gte", 15, 1}, // alice: 15 >= 15
			{"lt operator", "lt", 10, 1},   // bob: 8 < 10
			{"lte operator", "lte", 8, 1},  // bob: 8 <= 8
			{"eq operator", "eq", 15, 1},   // alice: 15 == 15
			{"ne operator", "ne", 15, 1},   // bob: 8 != 15
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				schema := &DetectionSchema{
					ID:        "test-rule-ops",
					VersionID: "version-ops",
					Model: map[string]interface{}{
						"correlation_type": "event_count",
						"parameters": map[string]interface{}{
							"time_window": "5m",
							"threshold":   int(tt.threshold),
							"operator":    tt.operator,
							"group_by":    []interface{}{".actor.user.name"},
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(3002),
								},
							},
						},
					},
					View: map[string]interface{}{
						"title":       "Operator Test",
						"severity":    "medium",
						"description": "Test",
					},
					Controller: map[string]interface{}{
						"detection": map[string]interface{}{
							"threshold": tt.threshold,
							"operator":  tt.operator,
						},
					},
				}

				alerts, err := evaluator.Evaluate(ctx, schema)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, len(alerts))
			})
		}
	})
}

func TestEventCountEvaluator_ParameterExtraction(t *testing.T) {
	stateManager := NewStateManager(nil, false)
	queryExecutor := &QueryExecutor{}
	evaluator := NewEventCountEvaluator(queryExecutor, stateManager)

	t.Run("extract basic parameters", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"group_by":    []interface{}{".actor.user.name"},
				},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, "5m", params.TimeWindow)
		assert.Equal(t, []string{".actor.user.name"}, params.GroupBy)
	})

	t.Run("extract with parameter set override", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"group_by":    []interface{}{".actor.user.name"},
				},
				"active_parameter_set": "prod",
				"parameter_sets": []interface{}{
					map[string]interface{}{
						"name": "dev",
						"parameters": map[string]interface{}{
							"time_window": "10m",
						},
					},
					map[string]interface{}{
						"name": "prod",
						"parameters": map[string]interface{}{
							"time_window": "3m",
						},
					},
				},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		// Should use prod parameter set
		assert.Equal(t, "3m", params.TimeWindow)
		assert.Equal(t, []string{".actor.user.name"}, params.GroupBy) // Not overridden
	})
}

func TestValueCountEvaluator_Evaluate(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock cardinality aggregation response
	mockResponse := map[string]interface{}{
		"took": 3,
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value": 500,
			},
		},
		"aggregations": map[string]interface{}{
			"groups": map[string]interface{}{
				"buckets": []interface{}{
					map[string]interface{}{
						"key": "10.0.1.5",
						"distinct_count": map[string]interface{}{
							"value": float64(150),
						},
					},
					map[string]interface{}{
						"key": "10.0.1.6",
						"distinct_count": map[string]interface{}{
							"value": float64(50),
						},
					},
				},
			},
		},
	}

	server := setupMockOpenSearch(t, mockResponse)
	defer server.Close()

	stateManager := NewStateManager(redisClient, true)
	queryExecutor := NewQueryExecutor(server.URL, "admin", "password", true)
	evaluator := NewValueCountEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	t.Run("port scanning detection", func(t *testing.T) {
		schema := &DetectionSchema{
			ID:        "port-scan-rule",
			VersionID: "version-123",
			Model: map[string]interface{}{
				"correlation_type": "value_count",
				"parameters": map[string]interface{}{
					"time_window": "10m",
					"field":       ".dst_endpoint.port",
					"threshold":   100,
					"operator":    "gt",
					"group_by":    []interface{}{".src_endpoint.ip"},
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    float64(4001),
						},
					},
				},
			},
			View: map[string]interface{}{
				"title":       "Port Scanning Detected",
				"severity":    "high",
				"description": "{{src_endpoint.ip}} scanned {{distinct_count}} ports",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"threshold": float64(100),
					"operator":  "gt",
				},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		assert.Equal(t, 1, len(alerts)) // Only 10.0.1.5 with 150 > 100
		assert.Equal(t, "Port Scanning Detected", alerts[0].Title)
		assert.Equal(t, int64(150), alerts[0].Metadata["distinct_count"])
	})
}

func TestEventCountEvaluator_ThresholdLogic(t *testing.T) {
	tests := []struct {
		name      string
		count     int64
		threshold int64
		operator  string
		expected  bool
	}{
		{"gt true", 15, 10, "gt", true},
		{"gt false", 10, 10, "gt", false},
		{"gte true equal", 10, 10, "gte", true},
		{"gte true greater", 15, 10, "gte", true},
		{"gte false", 9, 10, "gte", false},
		{"lt true", 5, 10, "lt", true},
		{"lt false", 10, 10, "lt", false},
		{"lte true equal", 10, 10, "lte", true},
		{"lte true less", 5, 10, "lte", true},
		{"lte false", 11, 10, "lte", false},
		{"eq true", 10, 10, "eq", true},
		{"eq false", 11, 10, "eq", false},
		{"ne true", 11, 10, "ne", true},
		{"ne false", 10, 10, "ne", false},
		{"default gt", 15, 10, "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := meetsThreshold(tt.count, tt.threshold, tt.operator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateAlert(t *testing.T) {
	evaluator := &EventCountEvaluator{}

	schema := &DetectionSchema{
		ID:        "rule-123",
		VersionID: "version-456",
		View: map[string]interface{}{
			"title":       "Test Alert",
			"severity":    "critical",
			"description": "Test description with {{event_count}} events in {{time_window}}",
		},
	}

	params := &EventCountParams{
		TimeWindow: "5m",
		Threshold:  10,
		GroupBy:    []string{".actor.user.name"},
	}

	alert := evaluator.createAlert(schema, params, "alice", 25, 5*time.Minute)

	assert.Equal(t, "rule-123", alert.RuleID)
	assert.Equal(t, "version-456", alert.RuleVersionID)
	assert.Equal(t, "Test Alert", alert.Title)
	assert.Equal(t, "critical", alert.Severity)
	assert.Equal(t, "event_count", alert.CorrelationType)
	assert.Equal(t, int64(25), alert.Metadata["event_count"])
	assert.Equal(t, "5m", alert.Metadata["time_window"])
	assert.Equal(t, "alice", alert.Metadata["group_key"])
}
