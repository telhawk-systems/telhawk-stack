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

func TestJoinEvaluator_Evaluate(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch that returns events for left and right queries
	queryCallCount := 0
	baseTime := time.Now().Add(-5 * time.Minute)

	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCallCount++

		var response map[string]interface{}
		if queryCallCount == 1 {
			// Left query: failed authentication
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 2},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event1",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Format(time.RFC3339),
								"class_uid": float64(3002),
								"status_id": float64(2),
								"user": map[string]interface{}{
									"name": "alice",
								},
							},
						},
						map[string]interface{}{
							"_id":    "event2",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Add(1 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(3002),
								"status_id": float64(2),
								"user": map[string]interface{}{
									"name": "bob",
								},
							},
						},
					},
				},
			}
		} else {
			// Right query: file deletions
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event3",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":        baseTime.Add(2 * time.Minute).Format(time.RFC3339),
								"class_uid":   float64(4005),
								"activity_id": float64(5),
								"actor": map[string]interface{}{
									"user": map[string]interface{}{
										"name": "alice", // Matches first failed auth
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
	evaluator := NewJoinEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	t.Run("join on matching user field", func(t *testing.T) {
		schema := &DetectionSchema{
			ID:        "join-rule-001",
			VersionID: "version-001",
			Model: map[string]interface{}{
				"correlation_type": "join",
				"parameters": map[string]interface{}{
					"time_window": "10m",
					"left_query": map[string]interface{}{
						"name": "failed_auth",
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
					"right_query": map[string]interface{}{
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
					"join_conditions": []interface{}{
						map[string]interface{}{
							"left_field":  ".user.name",
							"right_field": ".actor.user.name",
							"operator":    "eq",
						},
					},
					"join_type": "inner",
				},
			},
			View: map[string]interface{}{
				"title":       "Post-Auth-Failure Privilege Action",
				"severity":    "critical",
				"description": "User performed privileged action after failed auth",
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{},
			},
		}

		alerts, err := evaluator.Evaluate(ctx, schema)
		require.NoError(t, err)
		require.Equal(t, 1, len(alerts), "Should generate one alert for alice")

		alert := alerts[0]
		assert.Equal(t, "Post-Auth-Failure Privilege Action", alert.Title)
		assert.Equal(t, "critical", alert.Severity)
		assert.Equal(t, "join", alert.CorrelationType)
		assert.Equal(t, 2, alert.Metadata["event_count"])
		assert.Equal(t, "failed_auth", alert.Metadata["left_query"])
		assert.Equal(t, "file_delete", alert.Metadata["right_query"])
		assert.Equal(t, 2, len(alert.Events))
	})

	t.Run("no matches when fields don't match", func(t *testing.T) {
		queryCallCount = 0 // Reset

		schema := &DetectionSchema{
			ID:        "join-rule-002",
			VersionID: "version-002",
			Model: map[string]interface{}{
				"correlation_type": "join",
				"parameters": map[string]interface{}{
					"time_window": "10m",
					"left_query": map[string]interface{}{
						"name": "failed_auth",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(3002),
							},
						},
					},
					"right_query": map[string]interface{}{
						"name": "file_delete",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(4005),
							},
						},
					},
					"join_conditions": []interface{}{
						map[string]interface{}{
							"left_field":  ".user.name",
							"right_field": ".actor.user.different_field", // Won't match
							"operator":    "eq",
						},
					},
					"join_type": "inner",
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
		assert.Empty(t, alerts, "Should not match when join condition fails")
	})
}

func TestJoinEvaluator_MultipleJoinConditions(t *testing.T) {
	mr, redisClient := setupTestRedis(t)
	defer mr.Close()
	defer redisClient.Close()

	// Mock OpenSearch with events that match on multiple fields
	queryCallCount := 0
	baseTime := time.Now().Add(-5 * time.Minute)

	mockOpenSearch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryCallCount++

		var response map[string]interface{}
		if queryCallCount == 1 {
			// Left query: DNS query
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
								"class_uid": float64(4003),
								"query": map[string]interface{}{
									"hostname": "malicious.com",
								},
								"src_endpoint": map[string]interface{}{
									"ip": "10.0.1.5",
								},
							},
						},
					},
				},
			}
		} else {
			// Right query: Network connection
			response = map[string]interface{}{
				"took": 2,
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": 1},
					"hits": []interface{}{
						map[string]interface{}{
							"_id":    "event2",
							"_index": "telhawk-events-2025.01.01",
							"_source": map[string]interface{}{
								"time":      baseTime.Add(1 * time.Minute).Format(time.RFC3339),
								"class_uid": float64(4001),
								"dst_endpoint": map[string]interface{}{
									"hostname": "malicious.com", // Matches DNS query
								},
								"src_endpoint": map[string]interface{}{
									"ip": "10.0.1.5", // Matches source IP
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
	evaluator := NewJoinEvaluator(queryExecutor, stateManager)

	ctx := context.Background()

	schema := &DetectionSchema{
		ID:        "c2-detection-rule",
		VersionID: "version-001",
		Model: map[string]interface{}{
			"correlation_type": "join",
			"parameters": map[string]interface{}{
				"time_window": "10m",
				"left_query": map[string]interface{}{
					"name": "dns_query",
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    float64(4003),
						},
					},
				},
				"right_query": map[string]interface{}{
					"name": "network_connection",
					"query": map[string]interface{}{
						"filter": map[string]interface{}{
							"field":    ".class_uid",
							"operator": "eq",
							"value":    float64(4001),
						},
					},
				},
				"join_conditions": []interface{}{
					map[string]interface{}{
						"left_field":  ".query.hostname",
						"right_field": ".dst_endpoint.hostname",
						"operator":    "eq",
					},
					map[string]interface{}{
						"left_field":  ".src_endpoint.ip",
						"right_field": ".src_endpoint.ip",
						"operator":    "eq",
					},
				},
				"join_type": "inner",
			},
		},
		View: map[string]interface{}{
			"title":       "C2 Communication Detected",
			"severity":    "critical",
			"description": "DNS query followed by connection to same host",
		},
		Controller: map[string]interface{}{
			"detection": map[string]interface{}{},
		},
	}

	alerts, err := evaluator.Evaluate(ctx, schema)
	require.NoError(t, err)
	require.Equal(t, 1, len(alerts))

	alert := alerts[0]
	assert.Equal(t, "C2 Communication Detected", alert.Title)
	assert.Equal(t, "critical", alert.Severity)
}

func TestJoinEvaluator_ParameterExtraction(t *testing.T) {
	stateManager := NewStateManager(nil, false)
	queryExecutor := &QueryExecutor{}
	evaluator := NewJoinEvaluator(queryExecutor, stateManager)

	t.Run("extract join parameters", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "10m",
					"left_query": map[string]interface{}{
						"name": "failed_auth",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(3002),
							},
						},
					},
					"right_query": map[string]interface{}{
						"name": "file_delete",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(4005),
							},
						},
					},
					"join_conditions": []interface{}{
						map[string]interface{}{
							"left_field":  ".user.name",
							"right_field": ".actor.user.name",
							"operator":    "eq",
						},
					},
					"join_type": "inner",
				},
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, "10m", params.TimeWindow)
		assert.Equal(t, "failed_auth", params.LeftQuery.Name)
		assert.Equal(t, "file_delete", params.RightQuery.Name)
		assert.Equal(t, 1, len(params.JoinConditions))
		assert.Equal(t, ".user.name", params.JoinConditions[0].LeftField)
		assert.Equal(t, ".actor.user.name", params.JoinConditions[0].RightField)
		assert.Equal(t, "eq", params.JoinConditions[0].Operator)
		assert.Equal(t, "inner", params.JoinType)
	})

	t.Run("default join_type to inner", func(t *testing.T) {
		schema := &DetectionSchema{
			Model: map[string]interface{}{
				"parameters": map[string]interface{}{
					"time_window": "10m",
					"left_query": map[string]interface{}{
						"name": "query1",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(3002),
							},
						},
					},
					"right_query": map[string]interface{}{
						"name": "query2",
						"query": map[string]interface{}{
							"filter": map[string]interface{}{
								"field":    ".class_uid",
								"operator": "eq",
								"value":    float64(4001),
							},
						},
					},
					"join_conditions": []interface{}{
						map[string]interface{}{
							"left_field":  ".user.name",
							"right_field": ".user.name",
							"operator":    "eq",
						},
					},
					// join_type omitted
				},
			},
			Controller: map[string]interface{}{
				"detection": map[string]interface{}{},
			},
		}

		params, err := evaluator.extractParameters(schema)
		require.NoError(t, err)
		assert.Equal(t, "inner", params.JoinType)
	})
}

func TestJoinEvaluator_CompareValues(t *testing.T) {
	evaluator := &JoinEvaluator{}

	tests := []struct {
		name     string
		left     interface{}
		right    interface{}
		operator string
		expected bool
	}{
		{"equal strings", "alice", "alice", "eq", true},
		{"not equal strings", "alice", "bob", "eq", false},
		{"not equal operator", "alice", "bob", "ne", true},
		{"equal numbers", float64(42), float64(42), "eq", true},
		{"equal number to string", float64(42), "42", "eq", true},
		{"nil values equal", nil, nil, "ne", false},
		{"nil and value not equal", nil, "alice", "ne", true},
		{"default to eq", "test", "test", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.compareValues(tt.left, tt.right, tt.operator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getFieldValue with nested paths", func(t *testing.T) {
		fields := map[string]interface{}{
			"actor": map[string]interface{}{
				"user": map[string]interface{}{
					"name": "alice",
				},
			},
			"src_endpoint": map[string]interface{}{
				"ip": "10.0.1.5",
			},
		}

		// Test nested field access
		value := getFieldValue(fields, ".actor.user.name")
		assert.Equal(t, "alice", value)

		// Test one level
		srcEndpoint := getFieldValue(fields, "src_endpoint")
		assert.NotNil(t, srcEndpoint)

		// Test missing field
		missing := getFieldValue(fields, ".nonexistent.field")
		assert.Nil(t, missing)
	})

	t.Run("extractGroupKey", func(t *testing.T) {
		fields := map[string]interface{}{
			"actor": map[string]interface{}{
				"user": map[string]interface{}{
					"name": "alice",
				},
			},
			"src_endpoint": map[string]interface{}{
				"ip": "10.0.1.5",
			},
		}

		// Single field
		key1 := extractGroupKey(fields, []string{".actor.user.name"})
		assert.Equal(t, "alice|", key1)

		// Multiple fields
		key2 := extractGroupKey(fields, []string{".actor.user.name", ".src_endpoint.ip"})
		assert.Equal(t, "alice|10.0.1.5|", key2)

		// Empty group by
		key3 := extractGroupKey(fields, []string{})
		assert.Equal(t, "default", key3)
	})

	t.Run("splitFieldPath", func(t *testing.T) {
		tests := []struct {
			path     string
			expected []string
		}{
			{"actor.user.name", []string{"actor", "user", "name"}},
			{"src_endpoint.ip", []string{"src_endpoint", "ip"}},
			{"simple", []string{"simple"}},
			{"", []string{}},
		}

		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				result := splitFieldPath(tt.path)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}
