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

func TestTemporalOrderedEvaluator_Evaluate(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch that returns events in sequence
	queryCallCount := 0
	baseTime := time.Now().Add(-10 * time.Minute)

	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCallCount++

		var response map[string]interface{}
		if queryCallCount == 1 {
			// Step 1: Recon (network scan)
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event1",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Format(time.RFC3339),
								"class_uid": float64(6003),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "attacker",
									},
								},
							},
						},
					},
				},
			}
		} else if queryCallCount == 2 {
			// Step 2: Exploit
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event2",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Add(3 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(2004),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "attacker",
									},
								},
							},
						},
					},
				},
			}
		} else {
			// Step 3: Persistence
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event3",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Add(7 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(1003),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "attacker",
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
	evaluator := NewTemporalOrderedEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	t.Run("detect attack chain sequence", func(t *testing.T) {
		schema := &DetectionSchema{
			ID:        "attack-chain-rule",
			VersionID: "version-001",
			Model: map[string]interface{}{
				"correlation_type": "temporal_ordered",
				"parameters": map[string]interface{}{
					"time_window": "30m",
					"max_gap":     "10m",
					"group_by":    []interface{}{".actor.user.name"},
					"sequence": []interface{}{
						map[string]interface{}{
							"step": 1,
							"name": "recon",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(6003),
								},
							},
						},
						map[string]interface{}{
							"step": 2,
							"name": "exploit",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(2004),
								},
							},
						},
						map[string]interface{}{
							"step": 3,
							"name": "persistence",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(1003),
								},
							},
						},
					},
				},
			},
			View: map[string]interface{}{
				"title":       "Attack Chain Detected",
				"severity":    "critical",
				"description": "User {{actor.user.name}} executed attack sequence",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"strict_order": true,
				},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		require.Equal(t, 1, len(alerts))

		alert := alerts[0]
		assert.Equal(t, "Attack Chain Detected", alert.Title)
		assert.Equal(t, "critical", alert.Severity)
		assert.Equal(t, "temporal_ordered", alert.CorrelationType)
		assert.Equal(t, 3, alert.Metadata["sequence_length"])
		assert.Equal(t, 3, len(alert.Events))

		// Verify events are in correct temporal order
		assert.True(t, alert.Events[0].Time.Before(alert.Events[1].Time))
		assert.True(t, alert.Events[1].Time.Before(alert.Events[2].Time))

		// Verify correct class_uids in sequence
		assert.Equal(t, float64(6003), alert.Events[0].Fields["class_uid"]) // Recon
		assert.Equal(t, float64(2004), alert.Events[1].Fields["class_uid"]) // Exploit
		assert.Equal(t, float64(1003), alert.Events[2].Fields["class_uid"]) // Persistence
	})
}

func TestTemporalOrderedEvaluator_MaxGap(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch with events that exceed max gap
	queryCallCount := 0
	baseTime := time.Now().Add(-30 * time.Minute)

	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCallCount++

		var response map[string]interface{}
		if queryCallCount == 1 {
			// First event
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event1",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Format(time.RFC3339),
								"class_uid": float64(3002),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "user1",
									},
								},
							},
						},
					},
				},
			}
		} else {
			// Second event - 15 minutes later (exceeds 10m max_gap)
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event2",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Add(15 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(4001),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "user1",
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
	evaluator := NewTemporalOrderedEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	schema := &DetectionSchema{
		ID:        "gap-test-rule",
		VersionID: "version-001",
		Model: map[string]interface{}{
			"correlation_type": "temporal_ordered",
			"parameters": map[string]interface{}{
				"time_window": "30m",
				"max_gap":     "10m", // Events are 15 min apart, should not match
				"group_by":    []interface{}{".actor.user.name"},
				"sequence": []interface{}{
					map[string]interface{}{
						"step": 1,
						"name": "step1",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(3002),
							},
						},
					},
					map[string]interface{}{
						"step": 2,
						"name": "step2",
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
			"title":       "Test",
			"severity":    "medium",
			"description": "Test",
		},
		Controller: map[string]interface{}{
			"detection": map[string]interface{}{},
		},
	}

	alerts, err := evaluator.Evaluate(ctx, schema)
	require.NoError(t, err)
	assert.Empty(t, alerts, "Should not match when gap exceeds max_gap")
}

func TestTemporalOrderedEvaluator_ParameterExtraction(t *testing.T) {
	stateManager := NewStateManager(nil, false)
	queryExecutor := &QueryExecutor{}
	evaluator := NewTemporalOrderedEvaluator(queryExecutor, stateManager)

	t.Run("extract sequence parameters", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "30m",
					"max_gap":     "10m",
					"group_by":    []interface{}{".actor.user.name"},
					"sequence": []interface{}{
						map[string]interface{}{
							"step": 1,
							"name": "recon",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(6003),
								},
							},
						},
						map[string]interface{}{
							"step": 2,
							"name": "exploit",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(2004),
								},
							},
						},
					},
				},
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{
					"strict_order": true,
				},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, "30m", params.TimeWindow)
		assert.Equal(t, "10m", params.MaxGap)
		assert.Equal(t, 2, len(params.Sequence))
		assert.Equal(t, "recon", params.Sequence[0].Name)
		assert.Equal(t, "exploit", params.Sequence[1].Name)
		assert.Equal(t, 1, params.Sequence[0].Step)
		assert.Equal(t, 2, params.Sequence[1].Step)
		assert.True(t, params.StrictOrder)
		assert.Equal(t, []string{".actor.user.name"}, params.GroupBy)
	})

	t.Run("default max_gap to time_window", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "30m",
					"sequence": []interface{}{
						map[string]interface{}{
							"step": 1,
							"name": "step1",
							"query": map[string]interface{}{
								"filter": map[string]interface{}{
									"field":    ".class_uid",
									"operator": "eq",
									"value":    float64(3002),
								},
							},
						},
						map[string]interface{}{
							"step": 2,
							"name": "step2",
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
				"detection": map[string]interface{}{},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, "", params.MaxGap) // Should be empty, will default at evaluation time
	})
}
