package seeder

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
	"github.com/telhawk-systems/telhawk-stack/common/ocsf/objects"
)

// ocsfValidator validates OCSF field paths against the actual schema
type ocsfValidator struct {
	// Cache of struct types for performance
	typeCache map[string]reflect.Type
}

// newOCSFValidator creates a new OCSF validator
func newOCSFValidator() *ocsfValidator {
	return &ocsfValidator{
		typeCache: make(map[string]reflect.Type),
	}
}

// ValidateFieldPath checks if a field path is valid according to OCSF schema
func (v *ocsfValidator) ValidateFieldPath(fieldPath string) error {
	// Remove leading dot if present
	if len(fieldPath) > 0 && fieldPath[0] == '.' {
		fieldPath = fieldPath[1:]
	}

	if fieldPath == "" {
		return fmt.Errorf("empty field path")
	}

	parts := strings.Split(fieldPath, ".")

	// Start with a map[string]interface{} as the root (HEC event structure)
	// This represents the JSON that will be unmarshaled into OCSF events
	currentType := reflect.TypeOf(map[string]interface{}{})

	for i, part := range parts {
		if currentType == nil {
			return fmt.Errorf("invalid field path at '%s': parent is nil", strings.Join(parts[:i], "."))
		}

		// Handle map types (generic event structure)
		if currentType.Kind() == reflect.Map {
			// For maps, we need to check against known OCSF structures
			// Try to determine the struct type based on the field name
			structType, err := v.getStructTypeForField(part, i)
			if err != nil {
				// If we can't determine the type and this is not the last field,
				// it might be a nested map - allow it but warn
				if i < len(parts)-1 {
					continue // Allow nested maps for intermediate fields
				}
				// For the last field in the path, it's okay - it's a value
				return nil
			}
			currentType = structType
			continue
		}

		// Dereference pointers
		if currentType.Kind() == reflect.Ptr {
			currentType = currentType.Elem()
		}

		// Must be a struct at this point
		if currentType.Kind() != reflect.Struct {
			return fmt.Errorf("invalid field path at '%s': expected struct but got %s",
				strings.Join(parts[:i], "."), currentType.Kind())
		}

		// Find the field in the struct
		field, found := v.findFieldByJSONTag(currentType, part)
		if !found {
			return fmt.Errorf("field '%s' not found in OCSF schema at path '%s'",
				part, strings.Join(parts[:i], "."))
		}

		// Check if this is the last part of the path
		if i == len(parts)-1 {
			// Last field - validate it's a simple type (not a struct/map)
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			// Allow: string, int, bool, float, time.Time, []string, etc.
			// Disallow: nested structs (unless it's the full path to the struct)
			if fieldType.Kind() == reflect.Struct && fieldType.Name() != "Time" {
				// This means the field path points to a struct, not a field value
				// This is invalid for filter conditions
				return fmt.Errorf("field path '%s' points to a struct, not a value field (add the specific field name)",
					fieldPath)
			}
		}

		// Move to the next type
		currentType = field.Type
	}

	return nil
}

// getStructTypeForField attempts to determine the struct type for a field name
func (v *ocsfValidator) getStructTypeForField(fieldName string, depth int) (reflect.Type, error) {
	// Check cache first
	if typ, exists := v.typeCache[fieldName]; exists {
		return typ, nil
	}

	// Map common OCSF field names to their struct types
	var structType reflect.Type

	switch fieldName {
	case "metadata":
		structType = reflect.TypeOf(ocsf.Metadata{})
	case "actor":
		structType = reflect.TypeOf(objects.Actor{})
	case "src_endpoint", "dst_endpoint":
		structType = reflect.TypeOf(objects.NetworkEndpoint{})
	case "device":
		structType = reflect.TypeOf(objects.Device{})
	case "process", "parent_process":
		structType = reflect.TypeOf(objects.Process{})
	case "file":
		structType = reflect.TypeOf(objects.File{})
	case "user":
		structType = reflect.TypeOf(objects.User{})
	case "session":
		structType = reflect.TypeOf(objects.Session{})
	case "product":
		// This depends on context - in metadata, it's ocsf.Product
		// In other contexts, it's objects.Product
		if depth == 1 { // Assuming metadata.product
			structType = reflect.TypeOf(ocsf.Product{})
		} else {
			structType = reflect.TypeOf(objects.Product{})
		}
	default:
		return nil, fmt.Errorf("unknown OCSF field: %s", fieldName)
	}

	// Cache the result
	v.typeCache[fieldName] = structType
	return structType, nil
}

// findFieldByJSONTag finds a struct field by its JSON tag
func (v *ocsfValidator) findFieldByJSONTag(structType reflect.Type, jsonName string) (reflect.StructField, bool) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// Parse JSON tag (format: "name,omitempty" or just "name")
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] == jsonName {
			return field, true
		}
	}

	return reflect.StructField{}, false
}

// ValidateFilterFields validates all field paths in a filter
func (v *ocsfValidator) ValidateFilterFields(filter *QueryFilter) []error {
	var errors []error

	if filter.Type != "" {
		// Compound filter - validate all conditions
		for i, cond := range filter.Conditions {
			condErrors := v.ValidateFilterFields(&cond)
			for _, err := range condErrors {
				errors = append(errors, fmt.Errorf("condition[%d]: %w", i, err))
			}
		}
	} else {
		// Simple filter - validate the field
		if err := v.ValidateFieldPath(filter.Field); err != nil {
			errors = append(errors, fmt.Errorf("field '%s': %w", filter.Field, err))
		}
	}

	return errors
}
