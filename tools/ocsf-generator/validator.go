package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateValidatorForEvent generates a Validate method for an event class.
func generateValidatorForEvent(class *EventClass) string {
	var buf strings.Builder
	structName := toGoStructName(class.Name)

	buf.WriteString(fmt.Sprintf("// Validate checks that all required fields are properly set\n"))
	buf.WriteString(fmt.Sprintf("func (e *%s) Validate() error {\n", structName))

	// Track if we need to check any fields
	hasValidation := false

	// Check each attribute for required string fields
	for attrName, rawAttr := range class.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			continue
		}

		// Skip optional and recommended fields
		if attr.Requirement == "optional" || attr.Requirement == "recommended" {
			continue
		}

		// Skip base event fields
		if isBaseField(attrName) {
			continue
		}

		// Check if it's a string type that needs validation
		goType := inferGoTypeWithObjects(attrName)
		if goType == "string" {
			hasValidation = true
			fieldName := toGoFieldName(attrName)
			buf.WriteString(fmt.Sprintf("\tif e.%s == \"\" {\n", fieldName))
			buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"required field %s is empty\")\n", attrName))
			buf.WriteString("\t}\n")
		}
	}

	if !hasValidation {
		buf.WriteString("\t// No required string fields to validate\n")
	}

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

// generateValidatorForObject generates a Validate method for an object type.
func generateValidatorForObject(object *ObjectSchema) string {
	return generateValidatorForObjectWithAttrs(object, object.Attributes)
}

// generateValidatorForObjectWithAttrs generates a Validate method with specified attributes.
func generateValidatorForObjectWithAttrs(object *ObjectSchema, attrs map[string]json.RawMessage) string {
	var buf strings.Builder
	structName := toGoStructName(object.Name)

	buf.WriteString(fmt.Sprintf("// Validate checks that all required fields are properly set\n"))
	buf.WriteString(fmt.Sprintf("func (o *%s) Validate() error {\n", structName))

	// Track if we need to check any fields
	hasValidation := false

	// Check each attribute for explicitly required string fields
	for attrName, rawAttr := range attrs {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			continue
		}

		// Only validate fields that are EXPLICITLY marked as "required"
		// Fields with no requirement, "optional", or "recommended" are skipped
		if attr.Requirement != "required" {
			continue
		}

		// Check if it's a string type that needs validation
		goType := inferGoTypeSimple(attrName)
		if goType == "string" {
			hasValidation = true
			fieldName := toGoFieldName(attrName)
			buf.WriteString(fmt.Sprintf("\tif o.%s == \"\" {\n", fieldName))
			buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"required field %s is empty\")\n", attrName))
			buf.WriteString("\t}\n")
		}
	}

	if !hasValidation {
		buf.WriteString("\t// No required string fields to validate\n")
	}

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

// validateNoGenericTypes ensures no interface{} or map[string]interface{} in generated code.
func validateNoGenericTypes(outputDir string) error {
	var violations []string

	// Check events directory
	eventsDir := filepath.Join(outputDir, "events")
	err := filepath.Walk(eventsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Check for forbidden patterns
			if strings.Contains(string(content), "interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains interface{}", path))
			}
			if strings.Contains(string(content), "map[string]interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains map[string]interface{}", path))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Check objects directory
	objectsDir := filepath.Join(outputDir, "objects")
	err = filepath.Walk(objectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Check for forbidden patterns
			if strings.Contains(string(content), "interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains interface{}", path))
			}
			if strings.Contains(string(content), "map[string]interface{}") {
				violations = append(violations, fmt.Sprintf("%s contains map[string]interface{}", path))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(violations) > 0 {
		return fmt.Errorf("type safety violations found:\n%s", strings.Join(violations, "\n"))
	}

	return nil
}
