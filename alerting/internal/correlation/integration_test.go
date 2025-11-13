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

// TestEventCountIntegration tests the full event_count correlation flow
func TestEventCountIntegration(t *testing.T) {
	// Setup test infrastructure
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch with realistic event data
	queryCallCount := 0
	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCallCount++

		// Return aggregated failed login events
		response := map[string]interface{}{
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
						map[string]interface{}{
							"key":       "charlie",
							"doc_count": 12,
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockOpenSearch.Close()

	// Initialize components
	stateManager := NewStateManager(redisClient, true)
	queryExecutor := NewQueryExecutor(mockOpenSearch.URL, "admin", "password", true)
	evaluator := NewEventCountEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	// Create a realistic brute force detection rule
	schema := &DetectionSchema{
		ID:        "brute-force-rule-001",
		VersionID: "version-001",
		Model: map[string]interface{}{
			"correlation_type": "event_count",
			"parameters": map[string]interface{}{
				"time_window": "5m",
				"threshold":   10,
				"operator":    "gt",
				"group_by":    []interface{}{".actor.user.name"},
			},
		},
		View: map[string]interface{}{
			"title":       "Brute Force Attack Detected",
			"severity":    "high",
			"description": "User {{actor.user.name}} had {{event_count}} failed login attempts in {{time_window}}",
		},
		Controller: map[string]interface{}{
			"detection": map[string]interface{}{
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
				"threshold": float64(10),
				"operator":  "gt",
			},
			"suppression": map[string]interface{}{
				"enabled":        true,
				"window":         "1h",
				"max_alerts":     1,
				"suppress_on":    []interface{}{".actor.user.name"},
				"group_alerts":   true,
				"reset_on_event": false,
			},
		},
	}

	t.Run("initial evaluation generates alerts", func(t *testing.T) {
		// First evaluation - should generate alerts for alice and charlie (>10 events)
		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)

		// Should have 2 alerts: alice (15) and charlie (12), but not bob (8)
		require.Equal(t, 2, len(alerts), "Expected 2 alerts for users exceeding threshold")

		// Verify alert details
		aliceAlert := findAlertByGroupKey(alerts, "alice")
		require.NotNil(t, aliceAlert, "Expected alert for alice")
		assert.Equal(t, "Brute Force Attack Detected", aliceAlert.Title)
		assert.Equal(t, "high", aliceAlert.Severity)
		assert.Equal(t, int64(15), aliceAlert.Metadata["event_count"])
		assert.Equal(t, "5m", aliceAlert.Metadata["time_window"])

		charlieAlert := findAlertByGroupKey(alerts, "charlie")
		require.NotNil(t, charlieAlert, "Expected alert for charlie")
		assert.Equal(t, int64(12), charlieAlert.Metadata["event_count"])

		// Verify OpenSearch was queried
		assert.Equal(t, 1, queryCallCount, "Expected 1 OpenSearch query")
	})

	t.Run("suppression prevents duplicate alerts", func(t *testing.T) {
		// Record the alerts in state manager (simulating they were actually sent)
		aliceKey := map[string]string{".actor.user.name": "alice"}
		charlieKey := map[string]string{".actor.user.name": "charlie"}

		err := stateManager.RecordAlert(ctx, schema.ID, aliceKey, 1*time.Hour, 1)
		require.NoError(t, err)

		err = stateManager.RecordAlert(ctx, schema.ID, charlieKey, 1*time.Hour, 1)
		require.NoError(t, err)

		// Second evaluation (shortly after) - should NOT generate alerts due to suppression
		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)

		// Filter out suppressed alerts
		nonSuppressedAlerts := []*Alert{}
		for _, alert := range alerts {
			groupKey := alert.Metadata["group_key"].(string)
			suppressionKey := map[string]string{".actor.user.name": groupKey}
			suppressed, err := stateManager.IsSuppressed(ctx, schema.ID, suppressionKey)
			require.NoError(t, err)

			if !suppressed {
				nonSuppressedAlerts = append(nonSuppressedAlerts, alert)
			}
		}

		assert.Empty(t, nonSuppressedAlerts, "All alerts should be suppressed within suppression window")
	})

	t.Run("suppression expires after window", func(t *testing.T) {
		// Fast forward time past suppression window
		mr.FastForward(61 * time.Minute)

		// Third evaluation - suppression should have expired
		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)

		// Should generate alerts again
		assert.Equal(t, 2, len(alerts), "Alerts should be generated after suppression window expires")
	})
}

// TestValueCountIntegration tests the full value_count correlation flow
func TestValueCountIntegration(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch with port scanning data
	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"took": 3,
			"hits": map[string]interface{}{
				"total": map[string]interface{}{
					"value": 5000,
				},
			},
			"aggregations": map[string]interface{}{
				"groups": map[string]interface{}{
					"buckets": []interface{}{
						map[string]interface{}{
							"key": "10.0.1.5",
							"distinct_count": map[string]interface{}{
								"value": float64(250), // Scanning 250 different ports
							},
						},
						map[string]interface{}{
							"key": "10.0.1.6",
							"distinct_count": map[string]interface{}{
								"value": float64(50), // Normal behavior
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockOpenSearch.Close()

	stateManager := NewStateManager(redisClient, true)
	queryExecutor := NewQueryExecutor(mockOpenSearch.URL, "admin", "password", true)
	evaluator := NewValueCountEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	schema := &DetectionSchema{
		ID:        "port-scan-rule-001",
		VersionID: "version-001",
		Model: map[string]interface{}{
			"correlation_type": "value_count",
			"parameters": map[string]interface{}{
				"time_window": "10m",
				"field":       ".dst_endpoint.port",
				"threshold":   100,
				"operator":    "gt",
				"group_by":    []interface{}{".src_endpoint.ip"},
			},
		},
		View: map[string]interface{}{
			"title":       "Port Scanning Detected",
			"severity":    "critical",
			"description": "Host {{src_endpoint.ip}} scanned {{distinct_count}} unique ports",
		},
		Controller: map[string]interface{}{
			"detection": map[string]interface{}{
				"query": map[string]interface{}{
					"filter": map[string]interface{}{
						"field":    ".class_uid",
						"operator": "eq",
						"value":    float64(4001),
					},
				},
				"threshold": float64(100),
				"operator":  "gt",
			},
			"suppression": map[string]interface{}{
				"enabled":     true,
				"window":      "30m",
				"max_alerts":  1,
				"suppress_on": []interface{}{".src_endpoint.ip"},
			},
		},
	}

	t.Run("port scan detection", func(t *testing.T) {
		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)

		// Only 10.0.1.5 should generate alert (250 > 100)
		require.Equal(t, 1, len(alerts))

		alert := alerts[0]
		assert.Equal(t, "Port Scanning Detected", alert.Title)
		assert.Equal(t, "critical", alert.Severity)
		assert.Equal(t, int64(250), alert.Metadata["distinct_count"])
		assert.Equal(t, "10.0.1.5", alert.Metadata["group_key"])
	})
}

// TestMultipleCorrelationTypes tests different correlation types working together
func TestMultipleCorrelationTypes(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	requestCount := 0
	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Alternate between event_count and value_count responses
		var response map[string]interface{}
		if requestCount%2 == 1 {
			// First request: event_count response
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 100},
				},
				"aggregations": map[string]interface{}{
					"groups": map[string]interface{}{
						"buckets": []interface{}{
							map[string]interface{}{
								"key":       "testuser",
								"doc_count": 20,
							},
						},
					},
				},
			}
		} else {
			// Second request: value_count (cardinality) response
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 500},
				},
				"aggregations": map[string]interface{}{
					"groups": map[string]interface{}{
						"buckets": []interface{}{
							map[string]interface{}{
								"key": "testhost",
								"distinct_count": map[string]interface{}{
									"value": float64(60),
								},
							},
						},
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockOpenSearch.Close()

	stateManager := NewStateManager(redisClient, true)
	queryExecutor := NewQueryExecutor(mockOpenSearch.URL, "admin", "password", true)

	ctx := context.Background()

	t.Run("event_count and value_count work independently", func(t *testing.T) {
		eventCountEval := NewEventCountEvaluator(queryExecutor, stateManager)
		valueCountEval := NewValueCountEvaluator(queryExecutor, stateManager)

		eventCountSchema := &DetectionSchema{
			ID:        "rule-1",
			VersionID: "v1",
			Model: map[string]interface{}{
				"correlation_type": "event_count",
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"threshold":   10,
					"operator":    "gt",
					"group_by":    []interface{}{".actor.user.name"},
				},
			},
			View: map[string]interface{}{
				"title":       "Test Event Count",
				"severity":    "medium",
				"description": "User {{actor.user.name}} had {{event_count}} events",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    float64(3002),
						},
					},
					"threshold": float64(10),
					"operator":  "gt",
				},
			},
		}

		valueCountSchema := &DetectionSchema{
			ID:        "rule-2",
			VersionID: "v1",
			Model: map[string]interface{}{
				"correlation_type": "value_count",
				"parameters": map[string]interface{}{
					"time_window": "10m",
					"field":       ".dst_endpoint.port",
					"threshold":   50,
					"operator":    "gt",
					"group_by":    []interface{}{".src_endpoint.ip"},
				},
			},
			View: map[string]interface{}{
				"title":       "Test Value Count",
				"severity":    "high",
				"description": "Host scanned {{distinct_count}} ports",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    float64(4001),
						},
					},
					"threshold": float64(50),
					"operator":  "gt",
				},
			},
		}

		// Both should work without interfering
		eventAlerts, err := eventCountEval.Evaluate(ctx, eventCountSchema)
		require.NoError(t, err)
		assert.NotEmpty(t, eventAlerts)

		valueAlerts, err := valueCountEval.Evaluate(ctx, valueCountSchema)
		require.NoError(t, err)
		assert.NotEmpty(t, valueAlerts)

		// Verify state isolation
		eventKey := map[string]string{".actor.user.name": "testuser"}
		err = stateManager.RecordAlert(ctx, "rule-1", eventKey, time.Hour, 1)
		require.NoError(t, err)

		suppressed1, err := stateManager.IsSuppressed(ctx, "rule-1", eventKey)
		require.NoError(t, err)
		assert.True(t, suppressed1)

		// Different rule should not be suppressed
		suppressed2, err := stateManager.IsSuppressed(ctx, "rule-2", eventKey)
		require.NoError(t, err)
		assert.False(t, suppressed2)
	})
}

// Helper function to find alert by group key
func findAlertByGroupKey(alerts []*Alert, groupKey string) *Alert {
	for _, alert := range alerts {
		if gk, ok := alert.Metadata["group_key"].(string); ok && gk == groupKey {
			return alert
		}
	}
	return nil
}
