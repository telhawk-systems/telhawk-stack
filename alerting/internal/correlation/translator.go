package correlation

import (
	"fmt"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// SimpleTranslator converts canonical JSON queries to OpenSearch Query DSL
// This is a simplified version for correlation rules
type SimpleTranslator struct{}

// Translate converts a canonical Query to OpenSearch Query DSL
func (t *SimpleTranslator) Translate(q *model.Query) (map[string]interface{}, error) {
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
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// Add field projection
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
		query["sort"] = []map[string]interface{}{
			{"time": map[string]interface{}{"order": "desc"}},
		}
	}

	// Add pagination
	if q.Limit > 0 {
		query["size"] = q.Limit
	} else {
		query["size"] = 100
	}

	if q.Offset > 0 {
		query["from"] = q.Offset
	}

	// Add aggregations
	if len(q.Aggregations) > 0 {
		aggs := make(map[string]interface{})
		for _, agg := range q.Aggregations {
			aggDef, err := t.translateAggregation(&agg)
			if err != nil {
				return nil, err
			}
			aggs[agg.Name] = aggDef
		}
		query["aggs"] = aggs
	}

	return query, nil
}

func (t *SimpleTranslator) buildBoolQuery(filter *model.FilterExpr, timeRange *model.TimeRangeDef) (map[string]interface{}, error) {
	must := []interface{}{}

	// Add filter conditions
	if filter != nil {
		filterQuery, err := t.translateFilter(filter)
		if err != nil {
			return nil, err
		}
		must = append(must, filterQuery)
	}

	// Add time range
	if timeRange != nil {
		timeFilter, err := t.buildTimeRangeFilter(timeRange)
		if err != nil {
			return nil, err
		}
		must = append(must, timeFilter)
	}

	return map[string]interface{}{
		"must": must,
	}, nil
}

func (t *SimpleTranslator) translateFilter(f *model.FilterExpr) (interface{}, error) {
	if f.IsSimpleCondition() {
		return t.translateSimpleCondition(f)
	}
	return t.translateCompoundCondition(f)
}

func (t *SimpleTranslator) translateSimpleCondition(f *model.FilterExpr) (interface{}, error) {
	field := t.translateFieldPath(f.Field)

	switch f.Operator {
	case model.OpEq:
		return map[string]interface{}{"term": map[string]interface{}{field: f.Value}}, nil
	case model.OpNe:
		return map[string]interface{}{"bool": map[string]interface{}{
			"must_not": map[string]interface{}{"term": map[string]interface{}{field: f.Value}},
		}}, nil
	case model.OpGt:
		return map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"gt": f.Value}}}, nil
	case model.OpGte:
		return map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"gte": f.Value}}}, nil
	case model.OpLt:
		return map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"lt": f.Value}}}, nil
	case model.OpLte:
		return map[string]interface{}{"range": map[string]interface{}{field: map[string]interface{}{"lte": f.Value}}}, nil
	case model.OpIn:
		return map[string]interface{}{"terms": map[string]interface{}{field: f.Value}}, nil
	case model.OpContains:
		return map[string]interface{}{"wildcard": map[string]interface{}{field: fmt.Sprintf("*%v*", f.Value)}}, nil
	case model.OpStartsWith:
		return map[string]interface{}{"prefix": map[string]interface{}{field: f.Value}}, nil
	case model.OpExists:
		return map[string]interface{}{"exists": map[string]interface{}{"field": field}}, nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", f.Operator)
	}
}

func (t *SimpleTranslator) translateCompoundCondition(f *model.FilterExpr) (interface{}, error) {
	switch f.Type {
	case model.FilterTypeAnd:
		must := make([]interface{}, len(f.Conditions))
		for i, cond := range f.Conditions {
			q, err := t.translateFilter(&cond)
			if err != nil {
				return nil, err
			}
			must[i] = q
		}
		return map[string]interface{}{"bool": map[string]interface{}{"must": must}}, nil

	case model.FilterTypeOr:
		should := make([]interface{}, len(f.Conditions))
		for i, cond := range f.Conditions {
			q, err := t.translateFilter(&cond)
			if err != nil {
				return nil, err
			}
			should[i] = q
		}
		return map[string]interface{}{"bool": map[string]interface{}{"should": should, "minimum_should_match": 1}}, nil

	case model.FilterTypeNot:
		if f.Condition == nil {
			return nil, fmt.Errorf("NOT filter requires a condition")
		}
		q, err := t.translateFilter(f.Condition)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"bool": map[string]interface{}{"must_not": q}}, nil

	default:
		return nil, fmt.Errorf("unsupported filter type: %s", f.Type)
	}
}

func (t *SimpleTranslator) buildTimeRangeFilter(tr *model.TimeRangeDef) (map[string]interface{}, error) {
	rangeQuery := make(map[string]interface{})

	if tr.Last != "" {
		// Parse duration and calculate absolute time
		duration, err := time.ParseDuration(tr.Last)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %w", err)
		}
		now := time.Now()
		start := now.Add(-duration)
		rangeQuery["gte"] = start.UnixMilli()
		rangeQuery["lte"] = now.UnixMilli()
	} else {
		if tr.Start != nil {
			rangeQuery["gte"] = tr.Start.UnixMilli()
		}
		if tr.End != nil {
			rangeQuery["lte"] = tr.End.UnixMilli()
		}
	}

	return map[string]interface{}{
		"range": map[string]interface{}{
			"time": rangeQuery,
		},
	}, nil
}

func (t *SimpleTranslator) translateAggregation(agg *model.Aggregation) (map[string]interface{}, error) {
	switch agg.Type {
	case model.AggTypeTerms:
		// Use keyword field for terms aggregations
		field := t.translateAggregationFieldPath(agg.Field)
		aggDef := map[string]interface{}{
			"terms": map[string]interface{}{
				"field": field,
				"size":  agg.Size,
			},
		}
		// Add nested aggregations
		if len(agg.Aggregations) > 0 {
			nestedAggs := make(map[string]interface{})
			for _, nested := range agg.Aggregations {
				nestedDef, err := t.translateAggregation(&nested)
				if err != nil {
					return nil, err
				}
				nestedAggs[nested.Name] = nestedDef
			}
			aggDef["aggs"] = nestedAggs
		}
		return aggDef, nil

	case model.AggTypeCardinality:
		// Use keyword field for cardinality aggregations
		field := t.translateAggregationFieldPath(agg.Field)
		return map[string]interface{}{
			"cardinality": map[string]interface{}{
				"field": field,
			},
		}, nil

	case model.AggTypeAvg:
		field := t.translateFieldPath(agg.Field)
		return map[string]interface{}{"avg": map[string]interface{}{"field": field}}, nil
	case model.AggTypeSum:
		field := t.translateFieldPath(agg.Field)
		return map[string]interface{}{"sum": map[string]interface{}{"field": field}}, nil
	case model.AggTypeMin:
		field := t.translateFieldPath(agg.Field)
		return map[string]interface{}{"min": map[string]interface{}{"field": field}}, nil
	case model.AggTypeMax:
		field := t.translateFieldPath(agg.Field)
		return map[string]interface{}{"max": map[string]interface{}{"field": field}}, nil
	case model.AggTypeStats:
		field := t.translateFieldPath(agg.Field)
		return map[string]interface{}{"stats": map[string]interface{}{"field": field}}, nil

	default:
		return nil, fmt.Errorf("unsupported aggregation type: %s", agg.Type)
	}
}

func (t *SimpleTranslator) translateFieldPath(path string) string {
	// Remove leading dot if present
	if strings.HasPrefix(path, ".") {
		return strings.TrimPrefix(path, ".")
	}
	return path
}

// translateAggregationFieldPath translates a field path for use in aggregations
// For terms/cardinality aggregations on text fields, OpenSearch requires using the .keyword subfield
func (t *SimpleTranslator) translateAggregationFieldPath(path string) string {
	field := t.translateFieldPath(path)

	// Add .keyword suffix for aggregatable fields
	// OpenSearch text fields need .keyword for aggregations
	// Numeric and date fields don't need this
	if !strings.HasSuffix(field, ".keyword") && !isNumericOrDateField(field) {
		return field + ".keyword"
	}
	return field
}

// isNumericOrDateField checks if a field is numeric or date type (doesn't need .keyword)
func isNumericOrDateField(field string) bool {
	// Common OCSF numeric/date fields that don't need .keyword
	numericDateFields := []string{
		"time", "time_dt", "timestamp",
		"class_uid", "category_uid", "type_uid", "activity_id", "status_id",
		"severity_id", "confidence_id", "impact_id",
		"count", "port", "pid", "uid",
	}

	// Check if field ends with any of these
	for _, ndf := range numericDateFields {
		if strings.HasSuffix(field, ndf) {
			return true
		}
	}

	return false
}
