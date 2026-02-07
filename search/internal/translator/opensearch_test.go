package translator

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/search/pkg/model"
)

func TestTranslateSimpleQuery(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Filter: &model.FilterExpr{
			Field:    ".severity",
			Operator: model.OpEq,
			Value:    "High",
		},
		Limit: 100,
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	// Verify query structure
	if result["query"] == nil {
		t.Fatal("Missing query field")
	}

	// Verify bool query with must clause
	queryMap := result["query"].(map[string]interface{})
	boolQuery := queryMap["bool"].(map[string]interface{})
	must := boolQuery["must"].([]interface{})

	if len(must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(must))
	}

	// Verify term query
	termQuery := must[0].(map[string]interface{})["term"].(map[string]interface{})
	if termQuery["severity"] != "High" {
		t.Errorf("Expected severity=High, got %v", termQuery["severity"])
	}
}

func TestTranslateCompoundAnd(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Filter: &model.FilterExpr{
			Type: model.FilterTypeAnd,
			Conditions: []model.FilterExpr{
				{Field: ".class_uid", Operator: model.OpEq, Value: 3002},
				{Field: ".severity_id", Operator: model.OpGte, Value: 4},
			},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	// Get the filter from the query
	queryMap := result["query"].(map[string]interface{})
	boolQuery := queryMap["bool"].(map[string]interface{})
	must := boolQuery["must"].([]interface{})

	// Should have 1 item (the AND filter)
	if len(must) != 1 {
		t.Fatalf("Expected 1 must clause, got %d", len(must))
	}

	// The AND filter itself should be a bool query with 2 must clauses
	andFilter := must[0].(map[string]interface{})
	andBool := andFilter["bool"].(map[string]interface{})
	andMust := andBool["must"].([]interface{})

	if len(andMust) != 2 {
		t.Fatalf("Expected 2 conditions in AND, got %d", len(andMust))
	}
}

func TestTranslateCompoundOr(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Filter: &model.FilterExpr{
			Type: model.FilterTypeOr,
			Conditions: []model.FilterExpr{
				{Field: ".severity", Operator: model.OpEq, Value: "High"},
				{Field: ".severity", Operator: model.OpEq, Value: "Critical"},
			},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	queryMap := result["query"].(map[string]interface{})
	boolQuery := queryMap["bool"].(map[string]interface{})
	must := boolQuery["must"].([]interface{})

	// Get the OR filter
	orFilter := must[0].(map[string]interface{})
	orBool := orFilter["bool"].(map[string]interface{})
	should := orBool["should"].([]interface{})

	if len(should) != 2 {
		t.Fatalf("Expected 2 should clauses, got %d", len(should))
	}

	// Verify minimum_should_match is set
	if orBool["minimum_should_match"] != 1 {
		t.Error("Expected minimum_should_match=1 for OR query")
	}
}

func TestTranslateNotFilter(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Filter: &model.FilterExpr{
			Type: model.FilterTypeNot,
			Condition: &model.FilterExpr{
				Field:    ".actor.user.name",
				Operator: model.OpEq,
				Value:    "system",
			},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	queryMap := result["query"].(map[string]interface{})
	boolQuery := queryMap["bool"].(map[string]interface{})
	must := boolQuery["must"].([]interface{})

	// Get the NOT filter
	notFilter := must[0].(map[string]interface{})
	notBool := notFilter["bool"].(map[string]interface{})
	mustNot := notBool["must_not"]

	if mustNot == nil {
		t.Fatal("Expected must_not clause in NOT filter")
	}
}

func TestTranslateTimeRange(t *testing.T) {
	translator := NewOpenSearchTranslator()

	// Test relative time range
	query := &model.Query{
		TimeRange: &model.TimeRangeDef{
			Last: "1h",
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	queryMap := result["query"].(map[string]interface{})
	boolQuery := queryMap["bool"].(map[string]interface{})
	must := boolQuery["must"].([]interface{})

	// Find the time range filter
	var rangeFilter map[string]interface{}
	for _, item := range must {
		if rf, ok := item.(map[string]interface{})["range"]; ok {
			rangeFilter = rf.(map[string]interface{})
			break
		}
	}

	if rangeFilter == nil {
		t.Fatal("Expected range filter for time")
	}

	timeRange := rangeFilter["time"].(map[string]interface{})
	if timeRange["gte"] != "now-1h" {
		t.Errorf("Expected gte=now-1h, got %v", timeRange["gte"])
	}
}

func TestTranslateAbsoluteTimeRange(t *testing.T) {
	translator := NewOpenSearchTranslator()

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	query := &model.Query{
		TimeRange: &model.TimeRangeDef{
			Start: &start,
			End:   &end,
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	queryMap := result["query"].(map[string]interface{})
	boolQuery := queryMap["bool"].(map[string]interface{})
	must := boolQuery["must"].([]interface{})

	// Find the time range filter
	var rangeFilter map[string]interface{}
	for _, item := range must {
		if rf, ok := item.(map[string]interface{})["range"]; ok {
			rangeFilter = rf.(map[string]interface{})
			break
		}
	}

	timeRange := rangeFilter["time"].(map[string]interface{})
	if int64(timeRange["gte"].(int64)) != start.Unix() {
		t.Errorf("Expected gte=%d, got %v", start.Unix(), timeRange["gte"])
	}
	if int64(timeRange["lte"].(int64)) != end.Unix() {
		t.Errorf("Expected lte=%d, got %v", end.Unix(), timeRange["lte"])
	}
}

func TestTranslateOperators(t *testing.T) {
	translator := NewOpenSearchTranslator()

	tests := []struct {
		name     string
		operator string
		value    interface{}
		field    string // Field to use for test
		expected string // Expected OpenSearch operator/clause type
	}{
		{"Equals", model.OpEq, "test", ".test_id", "term"}, // Use _id suffix for exact match
		{"NotEquals", model.OpNe, "test", ".test_id", "must_not"},
		{"GreaterThan", model.OpGt, 100, ".test_field", "range"},
		{"GreaterThanEqual", model.OpGte, 100, ".test_field", "range"},
		{"LessThan", model.OpLt, 100, ".test_field", "range"},
		{"LessThanEqual", model.OpLte, 100, ".test_field", "range"},
		{"In", model.OpIn, []interface{}{"a", "b"}, ".test_field", "terms"},
		{"Contains", model.OpContains, "test", ".test_field", "wildcard"},
		{"StartsWith", model.OpStartsWith, "test", ".test_field", "wildcard"},
		{"EndsWith", model.OpEndsWith, "test", ".test_field", "wildcard"},
		{"Regex", model.OpRegex, "^test.*", ".test_field", "regexp"},
		{"Exists", model.OpExists, true, ".test_field", "exists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &model.Query{
				Filter: &model.FilterExpr{
					Field:    tt.field,
					Operator: tt.operator,
					Value:    tt.value,
				},
			}

			result, err := translator.Translate(query)
			if err != nil {
				t.Fatalf("Translation failed: %v", err)
			}

			// Verify the operator was translated correctly
			jsonBytes, _ := json.MarshalIndent(result, "", "  ")
			jsonStr := string(jsonBytes)

			if !containsString(jsonStr, tt.expected) {
				t.Errorf("Expected to find '%s' in query, got:\n%s", tt.expected, jsonStr)
			}
		})
	}
}

func TestTranslateFieldProjection(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Select: []string{".time", ".severity", ".actor.user.name"},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	source := result["_source"].([]string)
	if len(source) != 3 {
		t.Fatalf("Expected 3 source fields, got %d", len(source))
	}

	// Verify fields have dots removed
	expected := []string{"time", "severity", "actor.user.name"}
	for i, field := range source {
		if field != expected[i] {
			t.Errorf("Expected field %s, got %s", expected[i], field)
		}
	}
}

func TestTranslateSorting(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Sort: []model.SortSpec{
			{Field: ".severity_id", Order: "desc"},
			{Field: ".time", Order: "desc"},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	sorts := result["sort"].([]map[string]interface{})
	if len(sorts) != 2 {
		t.Fatalf("Expected 2 sort fields, got %d", len(sorts))
	}

	// Verify first sort
	firstSort := sorts[0]["severity_id"].(map[string]interface{})
	if firstSort["order"] != "desc" {
		t.Errorf("Expected desc order, got %v", firstSort["order"])
	}
}

func TestTranslateAggregations(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Aggregations: []model.Aggregation{
			{
				Type:  model.AggTypeTerms,
				Field: ".actor.user.name",
				Name:  "top_users",
				Size:  10,
			},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	aggs := result["aggs"].(map[string]interface{})
	if aggs["top_users"] == nil {
		t.Fatal("Expected 'top_users' aggregation")
	}

	topUsers := aggs["top_users"].(map[string]interface{})
	terms := topUsers["terms"].(map[string]interface{})

	// Terms aggregations should use .keyword suffix for text fields
	if terms["field"] != "actor.user.name.keyword" {
		t.Errorf("Expected field actor.user.name.keyword, got %v", terms["field"])
	}
	if terms["size"] != 10 {
		t.Errorf("Expected size 10, got %v", terms["size"])
	}
}

func TestTranslateNestedAggregations(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Aggregations: []model.Aggregation{
			{
				Type:  model.AggTypeTerms,
				Field: ".severity",
				Name:  "by_severity",
				Size:  5,
				Aggregations: []model.Aggregation{
					{
						Type:  model.AggTypeTerms,
						Field: ".actor.user.name",
						Name:  "top_users_per_severity",
						Size:  3,
					},
				},
			},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	aggs := result["aggs"].(map[string]interface{})
	bySeverity := aggs["by_severity"].(map[string]interface{})

	if bySeverity["aggs"] == nil {
		t.Fatal("Expected nested aggregations")
	}

	nestedAggs := bySeverity["aggs"].(map[string]interface{})
	if nestedAggs["top_users_per_severity"] == nil {
		t.Fatal("Expected nested 'top_users_per_severity' aggregation")
	}
}

func TestTranslatePagination(t *testing.T) {
	translator := NewOpenSearchTranslator()

	query := &model.Query{
		Limit:  100,
		Offset: 200,
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if result["size"] != 100 {
		t.Errorf("Expected size=100, got %v", result["size"])
	}
	if result["from"] != 200 {
		t.Errorf("Expected from=200, got %v", result["from"])
	}
}

func TestTranslateComplexQuery(t *testing.T) {
	translator := NewOpenSearchTranslator()

	// Complex query from the design doc: Failed auth from external IPs
	query := &model.Query{
		Select: []string{".time", ".severity", ".actor.user.name", ".src_endpoint.ip"},
		Filter: &model.FilterExpr{
			Type: model.FilterTypeAnd,
			Conditions: []model.FilterExpr{
				{Field: ".class_uid", Operator: model.OpEq, Value: 3002},
				{Field: ".status", Operator: model.OpEq, Value: "Failed"},
				{Field: ".severity", Operator: model.OpEq, Value: "High"},
				{
					Type: model.FilterTypeNot,
					Condition: &model.FilterExpr{
						Field:    ".src_endpoint.ip",
						Operator: model.OpCIDR,
						Value:    "10.0.0.0/8",
					},
				},
			},
		},
		TimeRange: &model.TimeRangeDef{
			Last: "24h",
		},
		Sort: []model.SortSpec{
			{Field: ".time", Order: "desc"},
		},
		Limit: 100,
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	// Verify all components are present
	if result["query"] == nil {
		t.Error("Missing query")
	}
	if result["_source"] == nil {
		t.Error("Missing _source")
	}
	if result["sort"] == nil {
		t.Error("Missing sort")
	}
	if result["size"] != 100 {
		t.Errorf("Expected size=100, got %v", result["size"])
	}

	// Print the generated query for manual inspection
	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Generated OpenSearch query:\n%s", string(jsonBytes))
}

func TestTranslateTermsAggregationWithOrder(t *testing.T) {
	translator := NewOpenSearchTranslator()

	tests := []struct {
		name     string
		order    map[string]string
		expected map[string]string
	}{
		{
			name:     "ascending count (rare events)",
			order:    map[string]string{"_count": "asc"},
			expected: map[string]string{"_count": "asc"},
		},
		{
			name:     "descending count (common events)",
			order:    map[string]string{"_count": "desc"},
			expected: map[string]string{"_count": "desc"},
		},
		{
			name:     "alphabetical ascending",
			order:    map[string]string{"_key": "asc"},
			expected: map[string]string{"_key": "asc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &model.Query{
				Aggregations: []model.Aggregation{
					{
						Type:  model.AggTypeTerms,
						Field: ".process.name",
						Name:  "processes",
						Size:  10,
						Order: tt.order,
					},
				},
			}

			result, err := translator.Translate(query)
			if err != nil {
				t.Fatalf("Translation failed: %v", err)
			}

			aggs := result["aggs"].(map[string]interface{})
			processes := aggs["processes"].(map[string]interface{})
			terms := processes["terms"].(map[string]interface{})

			if terms["order"] == nil {
				t.Fatal("Expected order in terms aggregation")
			}

			order := terms["order"].(map[string]string)
			for key, val := range tt.expected {
				if order[key] != val {
					t.Errorf("Expected order[%s]=%s, got %s", key, val, order[key])
				}
			}
		})
	}
}

func TestTranslateTopHitsAggregation(t *testing.T) {
	translator := NewOpenSearchTranslator()

	tests := []struct {
		name         string
		agg          model.Aggregation
		expectSize   int
		expectSorted bool
	}{
		{
			name: "top_hits with custom sort",
			agg: model.Aggregation{
				Type:        model.AggTypeTopHits,
				Name:        "first_event",
				TopHitsSize: 1,
				TopHitsSort: []model.SortSpec{
					{Field: ".time", Order: "asc"},
				},
			},
			expectSize:   1,
			expectSorted: true,
		},
		{
			name: "top_hits default sort",
			agg: model.Aggregation{
				Type:        model.AggTypeTopHits,
				Name:        "sample",
				TopHitsSize: 3,
			},
			expectSize:   3,
			expectSorted: true,
		},
		{
			name: "top_hits default size",
			agg: model.Aggregation{
				Type: model.AggTypeTopHits,
				Name: "default",
			},
			expectSize:   1,
			expectSorted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &model.Query{
				Aggregations: []model.Aggregation{tt.agg},
			}

			result, err := translator.Translate(query)
			if err != nil {
				t.Fatalf("Translation failed: %v", err)
			}

			aggs := result["aggs"].(map[string]interface{})
			aggResult := aggs[tt.agg.Name].(map[string]interface{})
			topHits := aggResult["top_hits"].(map[string]interface{})

			if topHits["size"] != tt.expectSize {
				t.Errorf("Expected size=%d, got %v", tt.expectSize, topHits["size"])
			}

			if tt.expectSorted {
				if topHits["sort"] == nil {
					t.Error("Expected sort to be set")
				}
			}
		})
	}
}

func TestTranslateNestedTopHitsAggregation(t *testing.T) {
	translator := NewOpenSearchTranslator()

	// Test nested aggregation: group by user, get first login per user
	query := &model.Query{
		Aggregations: []model.Aggregation{
			{
				Type:  model.AggTypeTerms,
				Field: ".actor.user.name",
				Name:  "users",
				Size:  100,
				Aggregations: []model.Aggregation{
					{
						Type:        model.AggTypeTopHits,
						Name:        "first_login",
						TopHitsSize: 1,
						TopHitsSort: []model.SortSpec{
							{Field: ".time", Order: "asc"},
						},
					},
				},
			},
		},
	}

	result, err := translator.Translate(query)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	aggs := result["aggs"].(map[string]interface{})
	users := aggs["users"].(map[string]interface{})

	// Check nested aggregations exist
	if users["aggs"] == nil {
		t.Fatal("Expected nested aggregations")
	}

	nestedAggs := users["aggs"].(map[string]interface{})
	firstLogin := nestedAggs["first_login"].(map[string]interface{})
	topHits := firstLogin["top_hits"].(map[string]interface{})

	if topHits["size"] != 1 {
		t.Errorf("Expected size=1, got %v", topHits["size"])
	}

	sorts := topHits["sort"].([]map[string]interface{})
	if len(sorts) != 1 {
		t.Fatalf("Expected 1 sort field, got %d", len(sorts))
	}

	timeSort := sorts[0]["time"].(map[string]interface{})
	if timeSort["order"] != "asc" {
		t.Errorf("Expected asc order, got %v", timeSort["order"])
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsString(s[1:], substr)))
}
