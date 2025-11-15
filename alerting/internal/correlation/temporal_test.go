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

func TestTemporalEvaluator_Evaluate(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch that returns events for multiple queries
	queryCallCount := 0
	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCallCount++

		// Return different events for each query
		var response map[string]interface{}
		switch queryCallCount {
		case 1:
			// First query: failed logins
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 10},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event1",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(3002),
								"status_id": float64(2),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "alice",
									},
								},
							},
						},
					},
				},
			}
		case 2:
			// Second query: file deletions
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 5},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event2",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":        time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
								"class_uid":   float64(4005),
								"activity_id": float64(5),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "alice",
									},
								},
							},
						},
					},
				},
			}
		default:
			// Third query: external connections
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 3},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event3",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(4001),
								"dst_endpoint": map[string]interface{}{
									"type": "Internet",
								},
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "alice",
									},
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
	evaluator := NewTemporalEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	t.Run("all queries match - generate alert", func(t *testing.T) {
		schema := &DetectionSchema{
			ID:        "temporal-rule-001",
			VersionID: "version-001",
			Model: map[string]interface{}{
				"correlation_type": "temporal",
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"group_by":    []interface{}{".actor.user.name"},
					"queries": []interface{}{
						map[string]interface{}{
							"name": "failed_login",
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
						map[string]interface{}{
							"name": "file_delete",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"type": "and",
									"conditions": []interface{}{
										map[string]interface{}{
											"field":    ".class_uid",
											"operator": "eq",
											"value":    float64(4005),
										},
										map[string]interface{}{
											"field":    ".activity_id",
											"operator": "eq",
											"value":    float64(5),
										},
									},
								},
							},
						},
						map[string]interface{}{
							"name": "external_conn",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(4001),
								},
							},
						},
					},
				},
			},
			View: map[string]interface{}{
				"title":       "Suspicious Activity Cluster",
				"severity":    "critical",
				"description": "User {{actor.user.name}} exhibited multiple suspicious behaviors",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"min_matches": 3,
				},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		require.Equal(t, 1, len(alerts))

		alert := alerts[0]
		assert.Equal(t, "Suspicious Activity Cluster", alert.Title)
		assert.Equal(t, "critical", alert.Severity)
		assert.Equal(t, "temporal", alert.CorrelationType)
		assert.Equal(t, 3, alert.Metadata["match_count"])
		assert.Equal(t, 3, alert.Metadata["event_count"])
		assert.Equal(t, 3, len(alert.Events))
	})

	t.Run("min_matches threshold", func(t *testing.T) {
		queryCallCount = 0 // Reset counter

		schema := &DetectionSchema{
			ID:        "temporal-rule-002",
			VersionID: "version-002",
			Model: map[string]interface{}{
				"correlation_type": "temporal",
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"group_by":    []interface{}{".actor.user.name"},
					"queries": []interface{}{
						map[string]interface{}{
							"name": "failed_login",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(3002),
								},
							},
						},
						map[string]interface{}{
							"name": "file_delete",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(4005),
								},
							},
						},
						map[string]interface{}{
							"name": "external_conn",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(4001),
								},
							},
						},
					},
				},
			},
			View: map[string]interface{}{
				"title":       "Test Alert",
				"severity":    "medium",
				"description": "Test",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"min_matches": 2, // Only need 2 out of 3
				},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		require.Equal(t, 1, len(alerts))
		assert.Equal(t, 3, alerts[0].Metadata["match_count"]) // Still matched all 3
	})
}

func TestTemporalEvaluator_ParameterExtraction(t *testing.T) {
	stateManager := NewStateManager(nil, false)
	queryExecutor := &QueryExecutor{}
	evaluator := NewTemporalEvaluator(queryExecutor, stateManager)

	t.Run("extract basic parameters", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"group_by":    []interface{}{".actor.user.name"},
					"queries": []interface{}{
						map[string]interface{}{
							"name": "query1",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(3002),
								},
							},
						},
						map[string]interface{}{
							"name": "query2",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(4001),
								},
							},
						},
					},
				},
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"min_matches": 2,
				},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, "5m", params.TimeWindow)
		assert.Equal(t, 2, len(params.Queries))
		assert.Equal(t, "query1", params.Queries[0].Name)
		assert.Equal(t, "query2", params.Queries[1].Name)
		assert.Equal(t, 2, params.MinMatches)
		assert.Equal(t, []string{".actor.user.name"}, params.GroupBy)
	})

	t.Run("default min_matches to all queries", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "5m",
					"queries": []interface{}{
						map[string]interface{}{
							"name": "query1",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(3002),
								},
							},
						},
						map[string]interface{}{
							"name": "query2",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(4001),
								},
							},
						},
						map[string]interface{}{
							"name": "query3",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(4005),
								},
							},
						},
					},
				},
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, 3, params.MinMatches) // Should default to number of queries
	})
}
