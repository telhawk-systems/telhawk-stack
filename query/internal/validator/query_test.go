package validator

import (
	"strings"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

func TestValidateSimpleQuery(t *testing.T) {
	v := NewQueryValidator()

	query := &model.Query{
		Filter: &model.FilterExpr{
			Field:    ".severity",
			Operator: model.OpEq,
			Value:    "High",
		},
		TimeRange: &model.TimeRangeDef{
			Last: "1h",
		},
		Limit: 100,
	}

	if err := v.Validate(query); err != nil {
		t.Errorf("Valid query should pass validation: %v", err)
	}
}

func TestValidateNilQuery(t *testing.T) {
	v := NewQueryValidator()

	err := v.Validate(nil)
	if err == nil {
		t.Error("Nil query should fail validation")
	}
	if !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("Expected 'cannot be nil' error, got: %v", err)
	}
}

func TestValidateFieldPaths(t *testing.T) {
	v := NewQueryValidator()

	tests := []struct {
		name      string
		field     string
		shouldErr bool
	}{
		{"Valid simple field", ".severity", false},
		{"Valid nested field", ".actor.user.name", false},
		{"Valid deep nested", ".attacks[0].tactic.name", false},
		{"Missing dot prefix", "severity", true},
		{"Empty field", "", true},
		{"Double dots", ".actor..user", true},
		{"Trailing dot", ".actor.", true},
		{"Only dot", ".", false}, // Special case - might be valid for root
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &model.Query{
				Select: []string{tt.field},
			}

			err := v.Validate(query)
			if tt.shouldErr && err == nil {
				t.Errorf("Field '%s' should fail validation", tt.field)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Field '%s' should pass validation, got: %v", tt.field, err)
			}
		})
	}
}

func TestValidateOperators(t *testing.T) {
	v := NewQueryValidator()

	tests := []struct {
		name      string
		operator  string
		value     interface{}
		shouldErr bool
	}{
		{"Valid eq", model.OpEq, "test", false},
		{"Valid ne", model.OpNe, "test", false},
		{"Valid gt", model.OpGt, 100, false},
		{"Valid gte", model.OpGte, 100, false},
		{"Valid lt", model.OpLt, 100, false},
		{"Valid lte", model.OpLte, 100, false},
		{"Valid in", model.OpIn, []interface{}{"a", "b"}, false},
		{"Valid contains", model.OpContains, "test", false},
		{"Valid startsWith", model.OpStartsWith, "test", false},
		{"Valid endsWith", model.OpEndsWith, "test", false},
		{"Valid regex", model.OpRegex, "^test.*", false},
		{"Valid exists", model.OpExists, true, false},
		{"Valid cidr", model.OpCIDR, "192.168.0.0/16", false},
		{"Invalid operator", "invalid_op", "test", true},
		{"Empty operator", "", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &model.Query{
				Filter: &model.FilterExpr{
					Field:    ".test_field",
					Operator: tt.operator,
					Value:    tt.value,
				},
			}

			err := v.Validate(query)
			if tt.shouldErr && err == nil {
				t.Errorf("Operator '%s' should fail validation", tt.operator)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Operator '%s' should pass validation, got: %v", tt.operator, err)
			}
		})
	}
}

func TestValidateOperatorValues(t *testing.T) {
	v := NewQueryValidator()

	tests := []struct {
		name      string
		operator  string
		value     interface{}
		shouldErr bool
		errMsg    string
	}{
		{"In with array", model.OpIn, []interface{}{"a", "b"}, false, ""},
		{"In with non-array", model.OpIn, "string", true, "must be an array"},
		{"Exists with bool", model.OpExists, true, false, ""},
		{"Exists with non-bool", model.OpExists, "true", true, "must be a boolean"},
		{"Regex with valid pattern", model.OpRegex, "^test.*", false, ""},
		{"Regex with invalid pattern", model.OpRegex, "[invalid", true, "invalid regex pattern"},
		{"Regex with non-string", model.OpRegex, 123, true, "must be a string"},
		{"CIDR with valid notation", model.OpCIDR, "192.168.0.0/16", false, ""},
		{"CIDR without slash", model.OpCIDR, "192.168.0.0", true, "must contain /"},
		{"CIDR with non-string", model.OpCIDR, 123, true, "must be a string"},
		{"Eq with nil value", model.OpEq, nil, true, "cannot be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &model.Query{
				Filter: &model.FilterExpr{
					Field:    ".test_field",
					Operator: tt.operator,
					Value:    tt.value,
				},
			}

			err := v.Validate(query)
			if tt.shouldErr && err == nil {
				t.Errorf("Should fail validation")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Should pass validation, got: %v", err)
			}
			if tt.shouldErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestValidateCompoundFilters(t *testing.T) {
	v := NewQueryValidator()

	t.Run("Valid AND filter", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type: model.FilterTypeAnd,
				Conditions: []model.FilterExpr{
					{Field: ".severity", Operator: model.OpEq, Value: "High"},
					{Field: ".status", Operator: model.OpEq, Value: "Failed"},
				},
			},
		}

		if err := v.Validate(query); err != nil {
			t.Errorf("Valid AND filter should pass: %v", err)
		}
	})

	t.Run("Valid OR filter", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type: model.FilterTypeOr,
				Conditions: []model.FilterExpr{
					{Field: ".severity", Operator: model.OpEq, Value: "High"},
					{Field: ".severity", Operator: model.OpEq, Value: "Critical"},
				},
			},
		}

		if err := v.Validate(query); err != nil {
			t.Errorf("Valid OR filter should pass: %v", err)
		}
	})

	t.Run("Valid NOT filter", func(t *testing.T) {
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

		if err := v.Validate(query); err != nil {
			t.Errorf("Valid NOT filter should pass: %v", err)
		}
	})

	t.Run("AND with empty conditions", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type:       model.FilterTypeAnd,
				Conditions: []model.FilterExpr{},
			},
		}

		err := v.Validate(query)
		if err == nil {
			t.Error("AND with empty conditions should fail")
		}
	})

	t.Run("NOT with nil condition", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type:      model.FilterTypeNot,
				Condition: nil,
			},
		}

		err := v.Validate(query)
		if err == nil {
			t.Error("NOT with nil condition should fail")
		}
	})

	t.Run("Invalid compound type", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type: "invalid_type",
				Conditions: []model.FilterExpr{
					{Field: ".test", Operator: model.OpEq, Value: "test"},
				},
			},
		}

		err := v.Validate(query)
		if err == nil {
			t.Error("Invalid compound type should fail")
		}
	})
}

func TestValidateNestedFilters(t *testing.T) {
	v := NewQueryValidator()

	t.Run("Nested AND/OR", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type: model.FilterTypeAnd,
				Conditions: []model.FilterExpr{
					{Field: ".class_uid", Operator: model.OpEq, Value: 3002},
					{
						Type: model.FilterTypeOr,
						Conditions: []model.FilterExpr{
							{Field: ".severity", Operator: model.OpEq, Value: "High"},
							{Field: ".severity", Operator: model.OpEq, Value: "Critical"},
						},
					},
				},
			},
		}

		if err := v.Validate(query); err != nil {
			t.Errorf("Valid nested filter should pass: %v", err)
		}
	})

	t.Run("Nested with invalid inner condition", func(t *testing.T) {
		query := &model.Query{
			Filter: &model.FilterExpr{
				Type: model.FilterTypeAnd,
				Conditions: []model.FilterExpr{
					{Field: ".class_uid", Operator: model.OpEq, Value: 3002},
					{
						Type: model.FilterTypeOr,
						Conditions: []model.FilterExpr{
							{Field: "invalid_field", Operator: model.OpEq, Value: "High"}, // Missing dot
						},
					},
				},
			},
		}

		err := v.Validate(query)
		if err == nil {
			t.Error("Nested filter with invalid field should fail")
		}
	})
}

func TestValidateTimeRange(t *testing.T) {
	v := NewQueryValidator()

	t.Run("Valid relative time", func(t *testing.T) {
		validTimes := []string{"15m", "1h", "24h", "7d", "30d", "90d"}
		for _, tr := range validTimes {
			query := &model.Query{
				TimeRange: &model.TimeRangeDef{Last: tr},
			}
			if err := v.Validate(query); err != nil {
				t.Errorf("Time range '%s' should be valid: %v", tr, err)
			}
		}
	})

	t.Run("Invalid relative time", func(t *testing.T) {
		invalidTimes := []string{"1", "1x", "m", "1hour", "1 hour", "-1h"}
		for _, tr := range invalidTimes {
			query := &model.Query{
				TimeRange: &model.TimeRangeDef{Last: tr},
			}
			err := v.Validate(query)
			if err == nil {
				t.Errorf("Time range '%s' should be invalid", tr)
			}
		}
	})

	t.Run("Valid absolute time", func(t *testing.T) {
		start := time.Now().Add(-24 * time.Hour)
		end := time.Now()
		query := &model.Query{
			TimeRange: &model.TimeRangeDef{
				Start: &start,
				End:   &end,
			},
		}
		if err := v.Validate(query); err != nil {
			t.Errorf("Valid absolute time range should pass: %v", err)
		}
	})

	t.Run("Start after end", func(t *testing.T) {
		start := time.Now()
		end := time.Now().Add(-24 * time.Hour)
		query := &model.Query{
			TimeRange: &model.TimeRangeDef{
				Start: &start,
				End:   &end,
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Start after end should fail validation")
		}
	})

	t.Run("Both absolute and relative", func(t *testing.T) {
		start := time.Now()
		query := &model.Query{
			TimeRange: &model.TimeRangeDef{
				Start: &start,
				Last:  "1h",
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Both absolute and relative time should fail")
		}
	})

	t.Run("Neither absolute nor relative", func(t *testing.T) {
		query := &model.Query{
			TimeRange: &model.TimeRangeDef{},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Empty time range should fail")
		}
	})
}

func TestValidateAggregations(t *testing.T) {
	v := NewQueryValidator()

	t.Run("Valid terms aggregation", func(t *testing.T) {
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
		if err := v.Validate(query); err != nil {
			t.Errorf("Valid terms aggregation should pass: %v", err)
		}
	})

	t.Run("Valid date histogram", func(t *testing.T) {
		query := &model.Query{
			Aggregations: []model.Aggregation{
				{
					Type:     model.AggTypeDateHistogram,
					Field:    ".time",
					Name:     "events_over_time",
					Interval: "1h",
				},
			},
		}
		if err := v.Validate(query); err != nil {
			t.Errorf("Valid date histogram should pass: %v", err)
		}
	})

	t.Run("Terms without field", func(t *testing.T) {
		query := &model.Query{
			Aggregations: []model.Aggregation{
				{
					Type: model.AggTypeTerms,
					Name: "top_users",
					Size: 10,
				},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Terms aggregation without field should fail")
		}
	})

	t.Run("Terms with invalid size", func(t *testing.T) {
		query := &model.Query{
			Aggregations: []model.Aggregation{
				{
					Type:  model.AggTypeTerms,
					Field: ".user",
					Name:  "top_users",
					Size:  0, // Invalid
				},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Terms aggregation with size 0 should fail")
		}
	})

	t.Run("Date histogram without interval", func(t *testing.T) {
		query := &model.Query{
			Aggregations: []model.Aggregation{
				{
					Type:  model.AggTypeDateHistogram,
					Field: ".time",
					Name:  "events_over_time",
				},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Date histogram without interval should fail")
		}
	})

	t.Run("Aggregation without name", func(t *testing.T) {
		query := &model.Query{
			Aggregations: []model.Aggregation{
				{
					Type:  model.AggTypeAvg,
					Field: ".risk_score",
				},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Aggregation without name should fail")
		}
	})

	t.Run("Invalid aggregation type", func(t *testing.T) {
		query := &model.Query{
			Aggregations: []model.Aggregation{
				{
					Type:  "invalid_type",
					Field: ".field",
					Name:  "agg",
				},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Invalid aggregation type should fail")
		}
	})

	t.Run("Too many aggregations", func(t *testing.T) {
		aggs := make([]model.Aggregation, 11) // Max is 10
		for i := 0; i < 11; i++ {
			aggs[i] = model.Aggregation{
				Type:  model.AggTypeAvg,
				Field: ".field",
				Name:  "agg",
			}
		}
		query := &model.Query{
			Aggregations: aggs,
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Too many aggregations should fail")
		}
	})

	t.Run("Valid nested aggregations", func(t *testing.T) {
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
							Field: ".user",
							Name:  "top_users",
							Size:  3,
						},
					},
				},
			},
		}
		if err := v.Validate(query); err != nil {
			t.Errorf("Valid nested aggregation should pass: %v", err)
		}
	})
}

func TestValidateSort(t *testing.T) {
	v := NewQueryValidator()

	t.Run("Valid sort", func(t *testing.T) {
		query := &model.Query{
			Sort: []model.SortSpec{
				{Field: ".time", Order: "desc"},
				{Field: ".severity_id", Order: "asc"},
			},
		}
		if err := v.Validate(query); err != nil {
			t.Errorf("Valid sort should pass: %v", err)
		}
	})

	t.Run("Sort with invalid field", func(t *testing.T) {
		query := &model.Query{
			Sort: []model.SortSpec{
				{Field: "invalid_field", Order: "desc"},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Sort with invalid field should fail")
		}
	})

	t.Run("Sort with invalid order", func(t *testing.T) {
		query := &model.Query{
			Sort: []model.SortSpec{
				{Field: ".time", Order: "invalid"},
			},
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Sort with invalid order should fail")
		}
	})
}

func TestValidatePagination(t *testing.T) {
	v := NewQueryValidator()

	t.Run("Valid limit", func(t *testing.T) {
		query := &model.Query{
			Limit: 100,
		}
		if err := v.Validate(query); err != nil {
			t.Errorf("Valid limit should pass: %v", err)
		}
	})

	t.Run("Negative limit", func(t *testing.T) {
		query := &model.Query{
			Limit: -1,
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Negative limit should fail")
		}
	})

	t.Run("Negative offset", func(t *testing.T) {
		query := &model.Query{
			Offset: -1,
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Negative offset should fail")
		}
	})

	t.Run("Limit exceeds max without cursor", func(t *testing.T) {
		query := &model.Query{
			Limit: 20000, // Max is 10,000 without cursor
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Limit exceeding max without cursor should fail")
		}
	})

	t.Run("Large limit with cursor is allowed", func(t *testing.T) {
		query := &model.Query{
			Limit:  20000,
			Cursor: "some_cursor_token",
		}
		// With cursor, larger limits are allowed for deep pagination
		err := v.Validate(query)
		if err != nil {
			t.Errorf("Large limit with cursor should be allowed: %v", err)
		}
	})

	t.Run("Both offset and cursor", func(t *testing.T) {
		query := &model.Query{
			Offset: 100,
			Cursor: "cursor_token",
		}
		err := v.Validate(query)
		if err == nil {
			t.Error("Using both offset and cursor should fail")
		}
	})
}

func TestValidateComplexQuery(t *testing.T) {
	v := NewQueryValidator()

	// Complex query from the design doc
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

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
			Start: &start,
			End:   &end,
		},
		Sort: []model.SortSpec{
			{Field: ".time", Order: "desc"},
		},
		Limit: 100,
	}

	if err := v.Validate(query); err != nil {
		t.Errorf("Complex valid query should pass: %v", err)
	}
}

func TestValidationErrorMessages(t *testing.T) {
	v := NewQueryValidator()

	tests := []struct {
		name      string
		query     *model.Query
		errSubstr string
	}{
		{
			name: "Invalid field path",
			query: &model.Query{
				Select: []string{"no_dot_prefix"},
			},
			errSubstr: "must start with '.'",
		},
		{
			name: "Invalid operator",
			query: &model.Query{
				Filter: &model.FilterExpr{
					Field:    ".test",
					Operator: "bad_op",
					Value:    "test",
				},
			},
			errSubstr: "unsupported operator",
		},
		{
			name: "Invalid time range",
			query: &model.Query{
				TimeRange: &model.TimeRangeDef{
					Last: "bad_time",
				},
			},
			errSubstr: "invalid relative time",
		},
		{
			name: "Negative limit",
			query: &model.Query{
				Limit: -5,
			},
			errSubstr: "cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.query)
			if err == nil {
				t.Errorf("Expected validation error")
				return
			}
			if !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errSubstr, err)
			}
		})
	}
}
