package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ValidationRules holds all validation rules for an event class
type ValidationRules struct {
	ClassName         string
	ClassUID          int
	Category          string
	RequiredFields    []string
	RecommendedFields []string
	EnumFields        map[string][]string // field_name -> valid values
}

// extractValidationRules parses an event class to extract validation rules
func extractValidationRules(class *EventClass) (*ValidationRules, error) {
	rules := &ValidationRules{
		ClassName:         class.Name,
		ClassUID:          class.UID,
		Category:          class.Category, // Set by caller (directory name)
		RequiredFields:    make([]string, 0),
		RecommendedFields: make([]string, 0),
		EnumFields:        make(map[string][]string),
	}

	// Parse each attribute
	for fieldName, rawAttr := range class.Attributes {
		// Skip $include directives
		if fieldName == "$include" {
			continue
		}

		var attr Attribute
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			if *verbose {
				fmt.Printf("Warning: failed to parse attribute %s: %v\n", fieldName, err)
			}
			continue
		}

		// Collect required and recommended fields
		switch attr.Requirement {
		case "required":
			rules.RequiredFields = append(rules.RequiredFields, fieldName)
		case "recommended":
			rules.RecommendedFields = append(rules.RecommendedFields, fieldName)
		}

		// Collect enum constraints
		if len(attr.Enum) > 0 {
			validValues := make([]string, 0, len(attr.Enum))
			for value := range attr.Enum {
				validValues = append(validValues, value)
			}
			sort.Strings(validValues)
			rules.EnumFields[fieldName] = validValues
		}
	}

	sort.Strings(rules.RequiredFields)
	sort.Strings(rules.RecommendedFields)

	return rules, nil
}

// generateValidator generates a validator file for a specific event class
func generateValidator(rules *ValidationRules, outputDir string) error {
	if rules.ClassUID == 0 {
		// Skip base_event - it will have a manually crafted base validator
		return nil
	}

	// Generate validate method body first to determine if fmt is needed
	var validateBuf strings.Builder
	needsFmt := generateValidateMethod(&validateBuf, toGoStructName(rules.ClassName)+"Validator", rules)

	var buf strings.Builder
	structName := toGoStructName(rules.ClassName) + "Validator"

	// Package and imports
	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	if needsFmt {
		buf.WriteString("\t\"fmt\"\n")
	}
	buf.WriteString("\n\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf\"\n")
	buf.WriteString(")\n\n")

	// Struct definition
	buf.WriteString(fmt.Sprintf("// %s validates OCSF %s events (class_uid %d)\n", structName, rules.ClassName, rules.ClassUID))
	buf.WriteString(fmt.Sprintf("type %s struct{}\n\n", structName))

	// Constructor
	buf.WriteString(fmt.Sprintf("// New%s creates a new validator\n", structName))
	buf.WriteString(fmt.Sprintf("func New%s() *%s {\n", structName, structName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{}\n", structName))
	buf.WriteString("}\n\n")

	// Supports method
	buf.WriteString(fmt.Sprintf("// Supports returns true for %s events\n", rules.ClassName))
	buf.WriteString(fmt.Sprintf("func (%s) Supports(class string) bool {\n", structName))
	buf.WriteString(fmt.Sprintf("\treturn class == %q\n", rules.ClassName))
	buf.WriteString("}\n\n")

	// Append validate method
	buf.WriteString(validateBuf.String())

	// Write to file
	filename := filepath.Join(outputDir, rules.ClassName+"_validator.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}

// generateValidateMethod generates the Validate method for a validator
// Returns true if the generated code uses fmt package
func generateValidateMethod(buf *strings.Builder, structName string, rules *ValidationRules) bool {
	buf.WriteString(fmt.Sprintf("// Validate ensures %s events have required fields and valid enum values\n", rules.ClassName))
	buf.WriteString(fmt.Sprintf("func (%s) Validate(ctx context.Context, event *ocsf.Event) error {\n", structName))
	buf.WriteString("\t_ = ctx\n\n")

	// Check class UID
	buf.WriteString(fmt.Sprintf("\tif event.ClassUID != %d {\n", rules.ClassUID))
	buf.WriteString("\t\treturn nil // Not our event class\n")
	buf.WriteString("\t}\n\n")

	needsFmt := false

	// Validate required fields
	if len(rules.RequiredFields) > 0 {
		buf.WriteString("\t// Validate required fields\n")
		for _, field := range rules.RequiredFields {
			if generateFieldValidation(buf, field, rules.ClassName) {
				needsFmt = true
			}
		}
		buf.WriteString("\n")
	}

	// Validate enums (these always use fmt if they exist and are implemented)
	if len(rules.EnumFields) > 0 {
		buf.WriteString("\t// Validate enumerated fields\n")
		for field, validValues := range rules.EnumFields {
			if generateEnumValidation(buf, field, validValues) {
				needsFmt = true
			}
		}
	}

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n\n")

	return needsFmt
}

// generateFieldValidation generates validation code for a required field
// Returns true if it generated code that uses fmt.Errorf
func generateFieldValidation(buf *strings.Builder, fieldName, className string) bool {
	// Map OCSF field names to Go struct paths
	fieldPath := mapFieldToGoPath(fieldName)

	switch fieldName {
	case "user":
		// Special case for authentication events
		buf.WriteString("\tif event.Actor == nil || event.Actor.User == nil {\n")
		buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"%s event missing required field: actor.user\")\n", className))
		buf.WriteString("\t}\n")
		buf.WriteString("\tif event.Actor.User.Name == \"\" {\n")
		buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"%s event missing required field: actor.user.name\")\n", className))
		buf.WriteString("\t}\n")
		return true // Uses fmt
	default:
		// For other fields, add a comment indicating manual validation may be needed
		buf.WriteString(fmt.Sprintf("\t// TODO: Validate required field '%s' (path: %s)\n", fieldName, fieldPath))
		buf.WriteString(fmt.Sprintf("\t// Manual validation may be needed for complex nested structures\n"))
		return false // Just a TODO, doesn't use fmt
	}
}

// generateEnumValidation generates validation code for enumerated fields
// Returns true if it generated code that uses fmt.Errorf
func generateEnumValidation(buf *strings.Builder, fieldName string, validValues []string) bool {
	// Common enum fields we can validate
	switch fieldName {
	case "activity_id":
		buf.WriteString("\tif event.ActivityID != 0 {\n")
		buf.WriteString("\t\tswitch event.ActivityID {\n")
		for _, val := range validValues {
			buf.WriteString(fmt.Sprintf("\t\tcase %s:\n", val))
		}
		buf.WriteString("\t\t\t// Valid\n")
		buf.WriteString("\t\tdefault:\n")
		buf.WriteString(fmt.Sprintf("\t\t\treturn fmt.Errorf(\"invalid activity_id: %%d (valid values: %s)\", event.ActivityID)\n", strings.Join(validValues, ", ")))
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}\n")
		return true // Uses fmt
	case "severity_id":
		buf.WriteString("\tif event.SeverityID != 0 {\n")
		buf.WriteString("\t\t// Severity validation - OCSF defines standard severity levels (1-6)\n")
		buf.WriteString("\t\tif event.SeverityID < 0 || event.SeverityID > 99 {\n")
		buf.WriteString(fmt.Sprintf("\t\t\treturn fmt.Errorf(\"invalid severity_id: %%d\", event.SeverityID)\n"))
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}\n")
		return true // Uses fmt
	case "status_id":
		buf.WriteString("\tif event.StatusID != 0 {\n")
		buf.WriteString("\t\t// Status validation - OCSF defines standard status codes\n")
		buf.WriteString("\t\tif event.StatusID < 0 || event.StatusID > 99 {\n")
		buf.WriteString(fmt.Sprintf("\t\t\treturn fmt.Errorf(\"invalid status_id: %%d\", event.StatusID)\n"))
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}\n")
		return true // Uses fmt
	default:
		// For other enums, add a comment
		buf.WriteString(fmt.Sprintf("\t// TODO: Validate enum field '%s' (valid values: %s)\n", fieldName, strings.Join(validValues, ", ")))
		return false // Just a TODO, doesn't use fmt
	}
}

// generateValidatorsRegistry generates a registry file that includes all validators
func generateValidatorsRegistry(allRules []*ValidationRules, outputDir string) error {
	var buf strings.Builder

	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/internal/validator\"\n")
	buf.WriteString(")\n\n")

	buf.WriteString("// AllValidators returns all generated validators\n")
	buf.WriteString("func AllValidators() []validator.Validator {\n")
	buf.WriteString("\treturn []validator.Validator{\n")

	for _, rules := range allRules {
		if rules.ClassUID == 0 {
			continue // Skip base_event
		}
		structName := toGoStructName(rules.ClassName) + "Validator"
		buf.WriteString(fmt.Sprintf("\t\tNew%s(),\n", structName))
	}

	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	filename := filepath.Join(outputDir, "validators_registry.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}
