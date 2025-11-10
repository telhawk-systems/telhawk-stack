package translator

import (
	"fmt"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// OpenSearchTranslator converts canonical JSON queries to OpenSearch Query DSL.
type OpenSearchTranslator struct{}

// NewOpenSearchTranslator creates a new OpenSearch translator.
func NewOpenSearchTranslator() *OpenSearchTranslator {
	return &OpenSearchTranslator{}
}

// Translate converts a canonical Query to OpenSearch Query DSL.
func (t *OpenSearchTranslator) Translate(q *model.Query) (map[string]interface{}, error) {
	query := make(map[string]interface{})

	// Build the filter/query portion
	if q.Filter != nil || q.TimeRange != nil {
		boolQuery, err := t.buildBoolQuery(q.Filter, q.TimeRange)
		if err != nil {
			return nil, fmt.Errorf("failed to build bool query: %w", err)
		}
		query["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	} else {
		// No filters - match all
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// Add field projection (source filtering)
	if len(q.Select) > 0 {
		sources := make([]string, len(q.Select))
		for i, field := range q.Select {
			sources[i] = t.translateFieldPath(field)
		}
		query["_source"] = sources
	}

	// Add sorting
	if len(q.Sort) > 0 {
		sorts := make([]map[string]interface{}, len(q.Sort))
		for i, s := range q.Sort {
			order := "desc"
			if s.Order != "" {
				order = s.Order
			}
			sorts[i] = map[string]interface{}{
				t.translateFieldPath(s.Field): map[string]interface{}{
					"order": order,
				},
			}
		}
		query["sort"] = sorts
	} else {
		// Default sort by time descending
		query["sort"] = []map[string]interface{}{
			{
				"time": map[string]interface{}{
					"order": "desc",
				},
			},
		}
	}

	// Add pagination
	if q.Limit > 0 {
		query["size"] = q.Limit
	} else {
		query["size"] = 100 // Default limit
	}

	if q.Offset > 0 {
		query["from"] = q.Offset
	}

	// Add cursor-based pagination (search_after)
	if q.Cursor != "" {
		// Cursor would be decoded to search_after values
		// For now, this is a placeholder for future implementation
		// query["search_after"] = decodeCursor(q.Cursor)
	}

	// Add aggregations
	if len(q.Aggregations) > 0 {
		aggs, err := t.buildAggregations(q.Aggregations)
		if err != nil {
			return nil, fmt.Errorf("failed to build aggregations: %w", err)
		}
		query["aggs"] = aggs
	}

	return query, nil
}

// buildBoolQuery constructs the OpenSearch bool query from filter and time range.
func (t *OpenSearchTranslator) buildBoolQuery(filter *model.FilterExpr, timeRange *model.TimeRangeDef) (map[string]interface{}, error) {
	boolQuery := make(map[string]interface{})
	must := []interface{}{}

	// Add filter conditions
	if filter != nil {
		filterQuery, err := t.translateFilter(filter)
		if err != nil {
			return nil, err
		}
		must = append(must, filterQuery)
	}

	// Add time range filter
	if timeRange != nil {
		timeFilter, err := t.buildTimeRangeFilter(timeRange)
		if err != nil {
			return nil, err
		}
		must = append(must, timeFilter)
	}

	if len(must) > 0 {
		boolQuery["must"] = must
	}

	return boolQuery, nil
}

// translateFilter converts a FilterExpr to OpenSearch query format.
func (t *OpenSearchTranslator) translateFilter(filter *model.FilterExpr) (interface{}, error) {
	if filter.IsSimpleCondition() {
		return t.translateSimpleCondition(filter)
	}

	if filter.IsCompoundCondition() {
		return t.translateCompoundCondition(filter)
	}

	return nil, fmt.Errorf("invalid filter expression: neither simple nor compound")
}

// translateSimpleCondition converts a simple field condition to OpenSearch format.
func (t *OpenSearchTranslator) translateSimpleCondition(filter *model.FilterExpr) (map[string]interface{}, error) {
	field := t.translateFieldPath(filter.Field)

	switch filter.Operator {
	case model.OpEq:
		// Use term for exact-match fields (IPs, IDs, numbers), match for text fields
		if t.shouldUseTermQuery(field, filter.Value) {
			return map[string]interface{}{
				"term": map[string]interface{}{
					field: filter.Value,
				},
			}, nil
		}
		return map[string]interface{}{
			"match": map[string]interface{}{
				field: filter.Value,
			},
		}, nil

	case model.OpNe:
		// Use term for exact-match fields (IPs, IDs, numbers), match for text fields
		if t.shouldUseTermQuery(field, filter.Value) {
			return map[string]interface{}{
				"bool": map[string]interface{}{
					"must_not": map[string]interface{}{
						"term": map[string]interface{}{
							field: filter.Value,
						},
					},
				},
			}, nil
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"match": map[string]interface{}{
						field: filter.Value,
					},
				},
			},
		}, nil

	case model.OpGt:
		return map[string]interface{}{
			"range": map[string]interface{}{
				field: map[string]interface{}{
					"gt": filter.Value,
				},
			},
		}, nil

	case model.OpGte:
		return map[string]interface{}{
			"range": map[string]interface{}{
				field: map[string]interface{}{
					"gte": filter.Value,
				},
			},
		}, nil

	case model.OpLt:
		return map[string]interface{}{
			"range": map[string]interface{}{
				field: map[string]interface{}{
					"lt": filter.Value,
				},
			},
		}, nil

	case model.OpLte:
		return map[string]interface{}{
			"range": map[string]interface{}{
				field: map[string]interface{}{
					"lte": filter.Value,
				},
			},
		}, nil

	case model.OpIn:
		return map[string]interface{}{
			"terms": map[string]interface{}{
				field: filter.Value,
			},
		}, nil

	case model.OpContains:
		// Use wildcard query for contains
		valueStr := fmt.Sprintf("*%v*", filter.Value)
		return map[string]interface{}{
			"wildcard": map[string]interface{}{
				field: map[string]interface{}{
					"value": valueStr,
				},
			},
		}, nil

	case model.OpStartsWith:
		valueStr := fmt.Sprintf("%v*", filter.Value)
		return map[string]interface{}{
			"wildcard": map[string]interface{}{
				field: map[string]interface{}{
					"value": valueStr,
				},
			},
		}, nil

	case model.OpEndsWith:
		valueStr := fmt.Sprintf("*%v", filter.Value)
		return map[string]interface{}{
			"wildcard": map[string]interface{}{
				field: map[string]interface{}{
					"value": valueStr,
				},
			},
		}, nil

	case model.OpRegex:
		return map[string]interface{}{
			"regexp": map[string]interface{}{
				field: filter.Value,
			},
		}, nil

	case model.OpExists:
		if filter.Value == true {
			return map[string]interface{}{
				"exists": map[string]interface{}{
					"field": field,
				},
			}, nil
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": map[string]interface{}{
					"exists": map[string]interface{}{
						"field": field,
					},
				},
			},
		}, nil

	case model.OpCIDR:
		// OpenSearch can handle CIDR notation directly in term queries
		return map[string]interface{}{
			"term": map[string]interface{}{
				field: filter.Value,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported operator: %s", filter.Operator)
	}
}

// translateCompoundCondition converts compound (AND/OR/NOT) conditions to OpenSearch format.
func (t *OpenSearchTranslator) translateCompoundCondition(filter *model.FilterExpr) (map[string]interface{}, error) {
	switch filter.Type {
	case model.FilterTypeAnd:
		must := make([]interface{}, len(filter.Conditions))
		for i, cond := range filter.Conditions {
			translated, err := t.translateFilter(&cond)
			if err != nil {
				return nil, err
			}
			must[i] = translated
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		}, nil

	case model.FilterTypeOr:
		should := make([]interface{}, len(filter.Conditions))
		for i, cond := range filter.Conditions {
			translated, err := t.translateFilter(&cond)
			if err != nil {
				return nil, err
			}
			should[i] = translated
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               should,
				"minimum_should_match": 1,
			},
		}, nil

	case model.FilterTypeNot:
		if filter.Condition == nil {
			return nil, fmt.Errorf("NOT filter requires a condition")
		}
		translated, err := t.translateFilter(filter.Condition)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": translated,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported compound filter type: %s", filter.Type)
	}
}

// buildTimeRangeFilter creates an OpenSearch range query for the time field.
func (t *OpenSearchTranslator) buildTimeRangeFilter(tr *model.TimeRangeDef) (map[string]interface{}, error) {
	rangeQuery := make(map[string]interface{})

	// Handle relative time ranges
	if tr.Last != "" {
		rangeQuery["gte"] = fmt.Sprintf("now-%s", tr.Last)
		rangeQuery["lte"] = "now"
		return map[string]interface{}{
			"range": map[string]interface{}{
				"time": rangeQuery,
			},
		}, nil
	}

	// Handle absolute time ranges
	if tr.Start != nil {
		rangeQuery["gte"] = tr.Start.Unix()
	}
	if tr.End != nil {
		rangeQuery["lte"] = tr.End.Unix()
	} else if tr.Start != nil {
		// No end time specified, default to now
		rangeQuery["lte"] = time.Now().Unix()
	}

	return map[string]interface{}{
		"range": map[string]interface{}{
			"time": rangeQuery,
		},
	}, nil
}

// buildAggregations converts aggregation specs to OpenSearch aggregations.
func (t *OpenSearchTranslator) buildAggregations(aggs []model.Aggregation) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, agg := range aggs {
		aggBody, err := t.translateAggregation(&agg)
		if err != nil {
			return nil, err
		}
		result[agg.Name] = aggBody
	}

	return result, nil
}

// translateAggregation converts a single aggregation to OpenSearch format.
func (t *OpenSearchTranslator) translateAggregation(agg *model.Aggregation) (map[string]interface{}, error) {
	field := t.translateFieldPath(agg.Field)

	var aggBody map[string]interface{}

	switch agg.Type {
	case model.AggTypeTerms:
		aggBody = map[string]interface{}{
			"terms": map[string]interface{}{
				"field": field,
				"size":  agg.Size,
			},
		}

	case model.AggTypeDateHistogram:
		aggBody = map[string]interface{}{
			"date_histogram": map[string]interface{}{
				"field":    field,
				"interval": agg.Interval,
			},
		}

	case model.AggTypeAvg:
		aggBody = map[string]interface{}{
			"avg": map[string]interface{}{
				"field": field,
			},
		}

	case model.AggTypeSum:
		aggBody = map[string]interface{}{
			"sum": map[string]interface{}{
				"field": field,
			},
		}

	case model.AggTypeMin:
		aggBody = map[string]interface{}{
			"min": map[string]interface{}{
				"field": field,
			},
		}

	case model.AggTypeMax:
		aggBody = map[string]interface{}{
			"max": map[string]interface{}{
				"field": field,
			},
		}

	case model.AggTypeStats:
		aggBody = map[string]interface{}{
			"stats": map[string]interface{}{
				"field": field,
			},
		}

	case model.AggTypeCardinality:
		aggBody = map[string]interface{}{
			"cardinality": map[string]interface{}{
				"field": field,
			},
		}

	default:
		return nil, fmt.Errorf("unsupported aggregation type: %s", agg.Type)
	}

	// Handle nested aggregations
	if len(agg.Aggregations) > 0 {
		nestedAggs, err := t.buildAggregations(agg.Aggregations)
		if err != nil {
			return nil, err
		}
		aggBody["aggs"] = nestedAggs
	}

	return aggBody, nil
}

// translateFieldPath converts OCSF field paths (with leading dot) to OpenSearch field names.
// Example: ".actor.user.name" -> "actor.user.name"
func (t *OpenSearchTranslator) translateFieldPath(field string) string {
	// Remove leading dot if present
	if strings.HasPrefix(field, ".") {
		return field[1:]
	}
	return field
}

// shouldUseTermQuery determines if a field should use term (exact match) vs match (analyzed text) queries.
// Returns true for fields that should use term queries (IPs, IDs, numeric values, boolean values).
func (t *OpenSearchTranslator) shouldUseTermQuery(field string, value interface{}) bool {
	// Numeric and boolean values always use term queries
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return true
	}

	// Check field name patterns for exact-match fields
	// IP addresses
	if strings.HasSuffix(field, ".ip") || strings.HasSuffix(field, "_ip") {
		return true
	}

	// IDs (uid, id)
	if strings.HasSuffix(field, ".uid") || strings.HasSuffix(field, "_uid") ||
		strings.HasSuffix(field, ".id") || strings.HasSuffix(field, "_id") {
		return true
	}

	// Port numbers
	if strings.HasSuffix(field, ".port") || strings.HasSuffix(field, "_port") {
		return true
	}

	// Status codes, class UIDs, category UIDs, etc.
	if strings.Contains(field, "_uid") || strings.Contains(field, "_id") ||
		strings.HasSuffix(field, ".code") || strings.HasSuffix(field, "_code") {
		return true
	}

	// Specific OCSF fields that should use exact matching
	exactMatchFields := map[string]bool{
		"class_uid":    true,
		"category_uid": true,
		"activity_id":  true,
		"type_uid":     true,
		"severity_id":  true,
		"status_id":    true,
		"observables":  true,
	}
	if exactMatchFields[field] {
		return true
	}

	// Default to match query for text fields (names, messages, etc.)
	return false
}
