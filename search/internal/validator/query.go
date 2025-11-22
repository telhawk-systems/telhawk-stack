package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/fields"
	"github.com/telhawk-systems/telhawk-stack/search/pkg/model"
)

// QueryValidator validates canonical Query structures before execution.
type QueryValidator struct {
	maxAggregations       int
	maxResultSize         int
	maxFilterDepth        int
	maxSelectFields       int
	maxSortFields         int
	validateFieldMappings bool // If true, validates fields against OpenSearch mapping
}

// NewQueryValidator creates a new query validator with default limits.
// Field mapping validation is enabled by default.
func NewQueryValidator() *QueryValidator {
	return &QueryValidator{
		maxAggregations:       10,    // Maximum number of aggregations per query
		maxResultSize:         10000, // Maximum number of results without cursor
		maxFilterDepth:        10,    // Maximum nesting depth for compound filters
		maxSelectFields:       100,   // Maximum number of fields in select clause
		maxSortFields:         10,    // Maximum number of sort fields
		validateFieldMappings: true,  // Validate fields against OpenSearch mapping by default
	}
}

// NewQueryValidatorWithoutFieldMapping creates a validator without OpenSearch field mapping validation.
// This is useful for testing or when field validation should be handled elsewhere.
func NewQueryValidatorWithoutFieldMapping() *QueryValidator {
	v := NewQueryValidator()
	v.validateFieldMappings = false
	return v
}

// SetFieldMappingValidation enables or disables OpenSearch field mapping validation.
func (v *QueryValidator) SetFieldMappingValidation(enabled bool) {
	v.validateFieldMappings = enabled
}

// Validate checks if a query is valid and safe to execute.
func (v *QueryValidator) Validate(q *model.Query) error {
	if q == nil {
		return fmt.Errorf("query cannot be nil")
	}

	// Validate that all field paths exist in the OpenSearch mapping.
	// This must be done early to provide clear error messages about invalid fields.
	if v.validateFieldMappings {
		allFields := v.collectFields(q)
		invalidFields := fields.ValidateFields(allFields)
		if len(invalidFields) > 0 {
			return fmt.Errorf("query contains fields not supported by OpenSearch mapping: %v", invalidFields)
		}
	}

	// Validate field projections
	if err := v.validateSelect(q.Select); err != nil {
		return fmt.Errorf("invalid select clause: %w", err)
	}

	// Validate filters
	if q.Filter != nil {
		if err := v.validateFilter(q.Filter); err != nil {
			return fmt.Errorf("invalid filter: %w", err)
		}
	}

	// Validate time range
	if q.TimeRange != nil {
		if err := v.validateTimeRange(q.TimeRange); err != nil {
			return fmt.Errorf("invalid time range: %w", err)
		}
	}

	// Validate aggregations
	if len(q.Aggregations) > 0 {
		if err := v.validateAggregations(q.Aggregations); err != nil {
			return fmt.Errorf("invalid aggregations: %w", err)
		}
	}

	// Validate sort
	if len(q.Sort) > 0 {
		if err := v.validateSort(q.Sort); err != nil {
			return fmt.Errorf("invalid sort: %w", err)
		}
	}

	// Validate pagination
	if err := v.validatePagination(q); err != nil {
		return fmt.Errorf("invalid pagination: %w", err)
	}

	return nil
}

// validateSelect validates field projection list.
func (v *QueryValidator) validateSelect(fields []string) error {
	if len(fields) > v.maxSelectFields {
		return fmt.Errorf("too many select fields: %d (max: %d)", len(fields), v.maxSelectFields)
	}

	for _, field := range fields {
		if err := v.validateFieldPath(field); err != nil {
			return fmt.Errorf("invalid field %s: %w", field, err)
		}
	}
	return nil
}

// validateFilter recursively validates filter expressions.
func (v *QueryValidator) validateFilter(filter *model.FilterExpr) error {
	return v.validateFilterWithDepth(filter, 0)
}

// validateFilterWithDepth validates filter expressions with depth tracking.
func (v *QueryValidator) validateFilterWithDepth(filter *model.FilterExpr, depth int) error {
	if depth > v.maxFilterDepth {
		return fmt.Errorf("filter nesting too deep: %d (max: %d)", depth, v.maxFilterDepth)
	}

	if filter.IsSimpleCondition() {
		return v.validateSimpleCondition(filter)
	}

	if filter.IsCompoundCondition() {
		return v.validateCompoundConditionWithDepth(filter, depth)
	}

	return fmt.Errorf("filter must be either a simple or compound condition")
}

// validateSimpleCondition validates a simple field condition.
func (v *QueryValidator) validateSimpleCondition(filter *model.FilterExpr) error {
	// Validate field path
	if err := v.validateFieldPath(filter.Field); err != nil {
		return fmt.Errorf("invalid field: %w", err)
	}

	// Validate operator
	if !v.isValidOperator(filter.Operator) {
		return fmt.Errorf("unsupported operator: %s", filter.Operator)
	}

	// Validate value
	if filter.Value == nil && filter.Operator != model.OpExists {
		return fmt.Errorf("value cannot be nil for operator %s", filter.Operator)
	}

	// Operator-specific validation
	switch filter.Operator {
	case model.OpIn:
		// Value must be an array
		if _, ok := filter.Value.([]interface{}); !ok {
			return fmt.Errorf("value for 'in' operator must be an array")
		}

	case model.OpExists:
		// Value must be a boolean
		if _, ok := filter.Value.(bool); !ok {
			return fmt.Errorf("value for 'exists' operator must be a boolean")
		}

	case model.OpRegex:
		// Validate regex pattern
		if pattern, ok := filter.Value.(string); ok {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("invalid regex pattern: %w", err)
			}
		} else {
			return fmt.Errorf("regex pattern must be a string")
		}

	case model.OpCIDR:
		// Basic CIDR validation (simple check)
		if cidr, ok := filter.Value.(string); ok {
			if !strings.Contains(cidr, "/") {
				return fmt.Errorf("invalid CIDR notation: must contain /")
			}
		} else {
			return fmt.Errorf("CIDR value must be a string")
		}
	}

	return nil
}

// validateCompoundCondition validates compound filter conditions.
func (v *QueryValidator) validateCompoundCondition(filter *model.FilterExpr) error {
	return v.validateCompoundConditionWithDepth(filter, 0)
}

// validateCompoundConditionWithDepth validates compound filter conditions with depth tracking.
func (v *QueryValidator) validateCompoundConditionWithDepth(filter *model.FilterExpr, depth int) error {
	// Validate type
	if !v.isValidCompoundType(filter.Type) {
		return fmt.Errorf("unsupported compound filter type: %s", filter.Type)
	}

	// Validate based on type
	switch filter.Type {
	case model.FilterTypeAnd, model.FilterTypeOr:
		if len(filter.Conditions) == 0 {
			return fmt.Errorf("%s filter requires at least one condition", filter.Type)
		}
		// Recursively validate all conditions
		for i, cond := range filter.Conditions {
			if err := v.validateFilterWithDepth(&cond, depth+1); err != nil {
				return fmt.Errorf("condition %d: %w", i, err)
			}
		}

	case model.FilterTypeNot:
		if filter.Condition == nil {
			return fmt.Errorf("NOT filter requires a condition")
		}
		if err := v.validateFilterWithDepth(filter.Condition, depth+1); err != nil {
			return fmt.Errorf("NOT condition: %w", err)
		}
	}

	return nil
}

// validateTimeRange validates time range specifications.
func (v *QueryValidator) validateTimeRange(tr *model.TimeRangeDef) error {
	// Must have either absolute or relative time range
	hasAbsolute := tr.Start != nil || tr.End != nil
	hasRelative := tr.Last != ""

	if !hasAbsolute && !hasRelative {
		return fmt.Errorf("time range must specify either start/end or last")
	}

	if hasAbsolute && hasRelative {
		return fmt.Errorf("time range cannot specify both absolute and relative times")
	}

	// Validate relative time format
	if hasRelative {
		if !v.isValidRelativeTime(tr.Last) {
			return fmt.Errorf("invalid relative time format: %s", tr.Last)
		}
	}

	// Validate absolute time range
	if hasAbsolute && tr.Start != nil && tr.End != nil {
		if tr.Start.After(*tr.End) {
			return fmt.Errorf("start time cannot be after end time")
		}
	}

	return nil
}

// validateAggregations validates aggregation specifications.
func (v *QueryValidator) validateAggregations(aggs []model.Aggregation) error {
	if len(aggs) > v.maxAggregations {
		return fmt.Errorf("too many aggregations: %d (max: %d)", len(aggs), v.maxAggregations)
	}

	for i, agg := range aggs {
		if err := v.validateAggregation(&agg); err != nil {
			return fmt.Errorf("aggregation %d (%s): %w", i, agg.Name, err)
		}
	}

	return nil
}

// validateAggregation validates a single aggregation.
func (v *QueryValidator) validateAggregation(agg *model.Aggregation) error {
	// Validate type
	if !v.isValidAggregationType(agg.Type) {
		return fmt.Errorf("unsupported aggregation type: %s", agg.Type)
	}

	// Validate name
	if agg.Name == "" {
		return fmt.Errorf("aggregation name cannot be empty")
	}

	// Validate field (required for most aggregation types)
	if agg.Field != "" {
		if err := v.validateFieldPath(agg.Field); err != nil {
			return fmt.Errorf("invalid field: %w", err)
		}
	}

	// Type-specific validation
	switch agg.Type {
	case model.AggTypeTerms:
		if agg.Field == "" {
			return fmt.Errorf("terms aggregation requires a field")
		}
		if agg.Size <= 0 {
			return fmt.Errorf("terms aggregation size must be > 0")
		}

	case model.AggTypeDateHistogram:
		if agg.Field == "" {
			return fmt.Errorf("date_histogram aggregation requires a field")
		}
		if agg.Interval == "" {
			return fmt.Errorf("date_histogram aggregation requires an interval")
		}
	}

	// Recursively validate nested aggregations
	if len(agg.Aggregations) > 0 {
		if err := v.validateAggregations(agg.Aggregations); err != nil {
			return fmt.Errorf("nested aggregations: %w", err)
		}
	}

	return nil
}

// validateSort validates sort specifications.
func (v *QueryValidator) validateSort(sorts []model.SortSpec) error {
	if len(sorts) > v.maxSortFields {
		return fmt.Errorf("too many sort fields: %d (max: %d)", len(sorts), v.maxSortFields)
	}

	for i, sort := range sorts {
		if err := v.validateFieldPath(sort.Field); err != nil {
			return fmt.Errorf("sort %d: invalid field: %w", i, err)
		}
		if sort.Order != "" && sort.Order != "asc" && sort.Order != "desc" {
			return fmt.Errorf("sort %d: invalid order: %s (must be 'asc' or 'desc')", i, sort.Order)
		}
	}
	return nil
}

// validatePagination validates pagination parameters.
func (v *QueryValidator) validatePagination(q *model.Query) error {
	// Limit validation
	if q.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if q.Limit > v.maxResultSize && q.Cursor == "" {
		return fmt.Errorf("limit %d exceeds maximum %d (use cursor pagination for large result sets)", q.Limit, v.maxResultSize)
	}

	// Offset validation
	if q.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	// Cannot use both offset and cursor
	if q.Offset > 0 && q.Cursor != "" {
		return fmt.Errorf("cannot use both offset and cursor pagination")
	}

	return nil
}

// validateFieldPath validates an OCSF field path syntax.
func (v *QueryValidator) validateFieldPath(field string) error {
	if field == "" {
		return fmt.Errorf("field path cannot be empty")
	}

	// Field paths should start with a dot (jq-style)
	if !strings.HasPrefix(field, ".") {
		return fmt.Errorf("field path must start with '.' (got: %s)", field)
	}

	// Basic validation: no double dots, no trailing dots
	if strings.Contains(field, "..") {
		return fmt.Errorf("field path cannot contain '..'")
	}

	if strings.HasSuffix(field, ".") && field != "." {
		return fmt.Errorf("field path cannot end with '.'")
	}

	return nil
}

// collectFields extracts all field paths from a query for validation.
func (v *QueryValidator) collectFields(q *model.Query) []string {
	fieldSet := make(map[string]struct{})

	// Collect from select clause
	for _, field := range q.Select {
		if field != "" && field != "." {
			fieldSet[field] = struct{}{}
		}
	}

	// Collect from filter
	if q.Filter != nil {
		v.collectFilterFields(q.Filter, fieldSet)
	}

	// Collect from aggregations
	for _, agg := range q.Aggregations {
		v.collectAggregationFields(&agg, fieldSet)
	}

	// Collect from sort
	for _, sort := range q.Sort {
		if sort.Field != "" && sort.Field != "." {
			fieldSet[sort.Field] = struct{}{}
		}
	}

	// Convert set to slice
	result := make([]string, 0, len(fieldSet))
	for field := range fieldSet {
		result = append(result, field)
	}
	return result
}

// collectFilterFields recursively collects field paths from filter expressions.
func (v *QueryValidator) collectFilterFields(filter *model.FilterExpr, fieldSet map[string]struct{}) {
	if filter == nil {
		return
	}

	// Simple condition: collect the field
	if filter.IsSimpleCondition() {
		if filter.Field != "" && filter.Field != "." {
			fieldSet[filter.Field] = struct{}{}
		}
		return
	}

	// Compound condition: recurse into conditions
	if filter.IsCompoundCondition() {
		for i := range filter.Conditions {
			v.collectFilterFields(&filter.Conditions[i], fieldSet)
		}
		if filter.Condition != nil {
			v.collectFilterFields(filter.Condition, fieldSet)
		}
	}
}

// collectAggregationFields recursively collects field paths from aggregations.
func (v *QueryValidator) collectAggregationFields(agg *model.Aggregation, fieldSet map[string]struct{}) {
	if agg.Field != "" && agg.Field != "." {
		fieldSet[agg.Field] = struct{}{}
	}

	// Recurse into nested aggregations
	for i := range agg.Aggregations {
		v.collectAggregationFields(&agg.Aggregations[i], fieldSet)
	}
}

// isValidOperator checks if an operator is supported.
func (v *QueryValidator) isValidOperator(op string) bool {
	validOps := []string{
		model.OpEq, model.OpNe, model.OpGt, model.OpGte,
		model.OpLt, model.OpLte, model.OpIn, model.OpContains,
		model.OpStartsWith, model.OpEndsWith, model.OpRegex,
		model.OpExists, model.OpCIDR,
	}
	for _, valid := range validOps {
		if op == valid {
			return true
		}
	}
	return false
}

// isValidCompoundType checks if a compound filter type is valid.
func (v *QueryValidator) isValidCompoundType(t string) bool {
	return t == model.FilterTypeAnd || t == model.FilterTypeOr || t == model.FilterTypeNot
}

// isValidAggregationType checks if an aggregation type is valid.
func (v *QueryValidator) isValidAggregationType(t string) bool {
	validTypes := []string{
		model.AggTypeTerms, model.AggTypeDateHistogram,
		model.AggTypeAvg, model.AggTypeSum, model.AggTypeMin,
		model.AggTypeMax, model.AggTypeStats, model.AggTypeCardinality,
	}
	for _, valid := range validTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// isValidRelativeTime checks if a relative time string is valid.
func (v *QueryValidator) isValidRelativeTime(rel string) bool {
	// Match patterns like: 15m, 1h, 24h, 7d, 30d
	matched, _ := regexp.MatchString(`^\d+(m|h|d)$`, rel)
	return matched
}
