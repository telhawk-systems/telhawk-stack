package fields

import (
	"fmt"
	"strings"
)

// ValidationError represents field validation errors for a detection rule
type ValidationError struct {
	InvalidFields []InvalidField
}

// InvalidField contains details about an invalid field reference
type InvalidField struct {
	Path     string // The invalid field path
	Location string // Where in the rule the field was found
}

func (e *ValidationError) Error() string {
	if len(e.InvalidFields) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder
	sb.WriteString("invalid field references in detection rule: ")
	for i, f := range e.InvalidFields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q (in %s)", f.Path, f.Location))
	}
	return sb.String()
}

// HasErrors returns true if there are validation errors
func (e *ValidationError) HasErrors() bool {
	return len(e.InvalidFields) > 0
}

// ValidateRule validates all field references in a detection rule.
// It examines the model, view, and controller sections for field paths.
// Returns nil if all fields are valid, or a ValidationError with all invalid fields.
func ValidateRule(model, view, controller map[string]interface{}) error {
	ve := &ValidationError{}

	// Validate model fields
	validateModel(model, ve)

	// Validate view fields (fields_order uses field paths without leading dots)
	validateView(view, ve)

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// validateModel extracts and validates fields from the model section
func validateModel(model map[string]interface{}, ve *ValidationError) {
	if model == nil {
		return
	}

	// Check parameters
	params, ok := model["parameters"].(map[string]interface{})
	if !ok {
		return
	}

	// Validate group_by fields
	if groupBy, ok := params["group_by"].([]interface{}); ok {
		for _, f := range groupBy {
			if field, ok := f.(string); ok {
				if !IsValidField(field) {
					ve.InvalidFields = append(ve.InvalidFields, InvalidField{
						Path:     field,
						Location: "model.parameters.group_by",
					})
				}
			}
		}
	}

	// Validate field (for value_count)
	if field, ok := params["field"].(string); ok {
		if !IsValidField(field) {
			ve.InvalidFields = append(ve.InvalidFields, InvalidField{
				Path:     field,
				Location: "model.parameters.field",
			})
		}
	}

	// Validate query filter
	if query, ok := params["query"].(map[string]interface{}); ok {
		if filter, ok := query["filter"].(map[string]interface{}); ok {
			validateFilter(filter, "model.parameters.query.filter", ve)
		}
	}

	// Validate sequence (for temporal_ordered rules)
	if sequence, ok := params["sequence"].([]interface{}); ok {
		for i, step := range sequence {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if query, ok := stepMap["query"].(map[string]interface{}); ok {
					if filter, ok := query["filter"].(map[string]interface{}); ok {
						location := fmt.Sprintf("model.parameters.sequence[%d].query.filter", i)
						validateFilter(filter, location, ve)
					}
				}
			}
		}
	}
}

// validateFilter recursively validates fields in a filter structure
func validateFilter(filter map[string]interface{}, location string, ve *ValidationError) {
	// Check if this is a condition with a field
	if field, ok := filter["field"].(string); ok {
		if !IsValidField(field) {
			ve.InvalidFields = append(ve.InvalidFields, InvalidField{
				Path:     field,
				Location: location,
			})
		}
	}

	// Check nested conditions (for "and", "or" compound filters)
	if conditions, ok := filter["conditions"].([]interface{}); ok {
		for i, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				condLocation := fmt.Sprintf("%s.conditions[%d]", location, i)
				validateFilter(condMap, condLocation, ve)
			}
		}
	}
}

// validateView extracts and validates fields from the view section
func validateView(view map[string]interface{}, ve *ValidationError) {
	if view == nil {
		return
	}

	// Validate fields_order (these don't have leading dots)
	if fieldsOrder, ok := view["fields_order"].([]interface{}); ok {
		for _, f := range fieldsOrder {
			if field, ok := f.(string); ok {
				// Skip special fields like event_count, distinct_count
				if isAggregationField(field) {
					continue
				}
				// fields_order uses paths without leading dots
				if !IsValidField("." + field) {
					ve.InvalidFields = append(ve.InvalidFields, InvalidField{
						Path:     field,
						Location: "view.fields_order",
					})
				}
			}
		}
	}
}

// isAggregationField returns true for special aggregation result fields
// that are computed at runtime and not actual OCSF fields
func isAggregationField(field string) bool {
	aggregationFields := map[string]bool{
		"event_count":    true,
		"distinct_count": true,
		"count":          true,
		"sum":            true,
		"avg":            true,
		"min":            true,
		"max":            true,
	}
	return aggregationFields[field]
}

// ExtractFields extracts all field references from a detection rule.
// Useful for analysis or debugging.
func ExtractFields(model, view, controller map[string]interface{}) []string {
	fields := make(map[string]bool)

	// Extract from model
	extractFieldsFromModel(model, fields)

	// Extract from view
	extractFieldsFromView(view, fields)

	result := make([]string, 0, len(fields))
	for f := range fields {
		result = append(result, f)
	}
	return result
}

func extractFieldsFromModel(model map[string]interface{}, fields map[string]bool) {
	if model == nil {
		return
	}

	params, ok := model["parameters"].(map[string]interface{})
	if !ok {
		return
	}

	// group_by
	if groupBy, ok := params["group_by"].([]interface{}); ok {
		for _, f := range groupBy {
			if field, ok := f.(string); ok {
				fields[field] = true
			}
		}
	}

	// field (value_count)
	if field, ok := params["field"].(string); ok {
		fields[field] = true
	}

	// query.filter
	if query, ok := params["query"].(map[string]interface{}); ok {
		if filter, ok := query["filter"].(map[string]interface{}); ok {
			extractFieldsFromFilter(filter, fields)
		}
	}

	// sequence
	if sequence, ok := params["sequence"].([]interface{}); ok {
		for _, step := range sequence {
			if stepMap, ok := step.(map[string]interface{}); ok {
				if query, ok := stepMap["query"].(map[string]interface{}); ok {
					if filter, ok := query["filter"].(map[string]interface{}); ok {
						extractFieldsFromFilter(filter, fields)
					}
				}
			}
		}
	}
}

func extractFieldsFromFilter(filter map[string]interface{}, fields map[string]bool) {
	if field, ok := filter["field"].(string); ok {
		fields[field] = true
	}
	if conditions, ok := filter["conditions"].([]interface{}); ok {
		for _, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				extractFieldsFromFilter(condMap, fields)
			}
		}
	}
}

func extractFieldsFromView(view map[string]interface{}, fields map[string]bool) {
	if view == nil {
		return
	}

	if fieldsOrder, ok := view["fields_order"].([]interface{}); ok {
		for _, f := range fieldsOrder {
			if field, ok := f.(string); ok {
				if !isAggregationField(field) {
					// Normalize to have leading dot for consistency
					if !strings.HasPrefix(field, ".") {
						field = "." + field
					}
					fields[field] = true
				}
			}
		}
	}
}
