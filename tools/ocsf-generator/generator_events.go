package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// generateEventClassFile generates a Go file for an OCSF event class.
func generateEventClassFile(class *EventClass, categoryUID int, outputDir, packageName string) error {
	// Generate Go code
	var buf strings.Builder

	// Check if we need objects import
	needsObjects := false
	for _, rawAttr := range class.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err == nil {
			// Check if any field uses objects package
			for attrName := range class.Attributes {
				if !isBaseField(attrName) {
					goType := inferGoTypeWithObjects(attrName)
					if strings.Contains(goType, "objects.") {
						needsObjects = true
						break
					}
				}
			}
		}
		if needsObjects {
			break
		}
	}

	// Check if we need fmt import (only if there are required string fields)
	needsFmt := false
	for attrName, rawAttr := range class.Attributes {
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err == nil {
			// Skip optional/recommended and base fields
			if attr.Requirement != "optional" && attr.Requirement != "recommended" && !isBaseField(attrName) {
				goType := inferGoTypeWithObjects(attrName)
				if goType == "string" {
					needsFmt = true
					break
				}
			}
		}
	}

	// Package and imports
	buf.WriteString(generateFileHeader())
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	buf.WriteString("import (\n")
	if needsFmt {
		buf.WriteString("\t\"fmt\"\n")
	}
	buf.WriteString("\t\"time\"\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/common/ocsf\"\n")
	if needsObjects {
		buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/common/ocsf/objects\"\n")
	}
	buf.WriteString(")\n\n")

	buf.WriteString(fmt.Sprintf("type %s struct {\n", toGoStructName(class.Name)))
	buf.WriteString("\t// Embed base OCSF event\n")
	buf.WriteString("\tocsf.Event\n\n")

	// Add class-specific fields
	buf.WriteString("\t// Class-specific attributes\n")

	// Sort attributes for consistent output
	attrs := make([]string, 0, len(class.Attributes))
	for name := range class.Attributes {
		attrs = append(attrs, name)
	}
	sort.Strings(attrs)

	for _, attrName := range attrs {
		rawAttr := class.Attributes[attrName]

		// Try to parse as AttributeRef
		var attr AttributeRef
		if err := json.Unmarshal(rawAttr, &attr); err != nil {
			// Skip if we can't parse (might be array or other format)
			if *verbose {
				fmt.Printf("  Warning: skipping attribute %s in %s: %v\n", attrName, class.Name, err)
			}
			continue
		}

		// Skip base event fields we already have
		if isBaseField(attrName) {
			continue
		}

		fieldName := toGoFieldName(attrName)
		goType := inferGoTypeWithObjects(attrName)
		jsonTag := attrName

		omit := ""
		if attr.Requirement == "optional" || attr.Requirement == "recommended" {
			omit = ",omitempty"
		}

		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s%s\"`\n", fieldName, goType, jsonTag, omit))
	}

	// Add extra fields for specific event classes that need them
	// (e.g., Attacks for detection_finding from security_control profile)
	extraFields := getExtraFieldsForClass(class.Name)
	for _, ef := range extraFields {
		buf.WriteString(fmt.Sprintf("\t%s %s `json:\"%s,omitempty\"`\n", ef.FieldName, ef.GoType, ef.JSONTag))
	}

	buf.WriteString("}\n\n")

	// Generate activity enum if present
	if rawActivityAttr, hasEnum := class.Attributes["activity_id"]; hasEnum {
		var activityAttr AttributeRef
		if err := json.Unmarshal(rawActivityAttr, &activityAttr); err == nil && len(activityAttr.Enum) > 0 {
			buf.WriteString(generateActivityEnum(class, activityAttr))
		}
	}

	// Generate constructor helper
	buf.WriteString(generateConstructor(class, categoryUID))

	// Generate Validate method
	buf.WriteString(generateValidatorForEvent(class))

	// Write to file
	outputPath := filepath.Join(outputDir, toGoFileName(class.Name))
	return os.WriteFile(outputPath, []byte(buf.String()), 0644)
}

// generateActivityEnum generates activity constants for an event class.
func generateActivityEnum(class *EventClass, activityAttr AttributeRef) string {
	var buf strings.Builder

	structName := toGoStructName(class.Name)
	buf.WriteString("const (\n")

	// Sort enum values
	keys := make([]string, 0, len(activityAttr.Enum))
	for k := range activityAttr.Enum {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		enum := activityAttr.Enum[key]
		constName := fmt.Sprintf("%sActivity%s", structName, toGoConstName(enum.Caption))
		buf.WriteString(fmt.Sprintf("\t%s = %s\n", constName, key))
	}

	buf.WriteString(")\n\n")
	return buf.String()
}

// generateConstructor generates a constructor function for an event class.
func generateConstructor(class *EventClass, categoryUID int) string {
	var buf strings.Builder

	structName := toGoStructName(class.Name)
	funcName := "New" + structName

	buf.WriteString(fmt.Sprintf("func %s(activityID int) *%s {\n", funcName, structName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{\n", structName))
	buf.WriteString("\t\tEvent: ocsf.Event{\n")

	buf.WriteString(fmt.Sprintf("\t\t\tCategoryUID: %d,\n", categoryUID))
	buf.WriteString(fmt.Sprintf("\t\t\tClassUID:    %d,\n", class.UID))
	buf.WriteString("\t\t\tActivityID:  activityID,\n")
	buf.WriteString(fmt.Sprintf("\t\t\tTypeUID:     ocsf.ComputeTypeUID(%d, %d, activityID),\n", categoryUID, class.UID))
	buf.WriteString("\t\t\tTime:        time.Now(),\n")
	buf.WriteString("\t\t\tSeverityID:  ocsf.SeverityUnknown,\n")
	buf.WriteString(fmt.Sprintf("\t\t\tCategory:    %q,\n", class.Category))
	buf.WriteString(fmt.Sprintf("\t\t\tClass:       %q,\n", class.Name))
	buf.WriteString("\t\t\tSeverity:    \"Unknown\",\n")
	buf.WriteString("\t\t\tMetadata: ocsf.Metadata{\n")
	buf.WriteString("\t\t\t\tProduct: ocsf.Product{\n")
	buf.WriteString("\t\t\t\t\tName:   \"TelHawk Stack\",\n")
	buf.WriteString("\t\t\t\t\tVendor: \"TelHawk Systems\",\n")
	buf.WriteString("\t\t\t\t},\n")
	buf.WriteString("\t\t\t\tVersion: \"1.1.0\",\n")
	buf.WriteString("\t\t\t},\n")
	buf.WriteString("\t\t},\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

// generateFileHeader generates the standard header for generated files.
func generateFileHeader() string {
	return `// Code generated by TelHawk OCSF Generator. DO NOT EDIT.
//
// Generated from OCSF Schema: https://github.com/ocsf/ocsf-schema
//
// This generated code contains no original authorship and is released into the
// public domain. For convenience, TelHawk Systems provides it under the TelHawk
// Systems Source Available License (TSSAL) v1.0 as-is, without warranty of any kind.

`
}

// ExtraField defines an additional field to add to specific event classes.
// Used for fields from OCSF profiles that aren't automatically inherited.
type ExtraField struct {
	FieldName string // Go field name (e.g., "Attacks")
	GoType    string // Go type (e.g., "[]*objects.Attack")
	JSONTag   string // JSON tag (e.g., "attacks")
}

// getExtraFieldsForClass returns additional fields that should be added to
// specific event classes. This handles fields from OCSF profiles (like
// security_control) that aren't automatically inherited by certain classes.
func getExtraFieldsForClass(className string) []ExtraField {
	// detection_finding should have Attacks from security_control profile
	// but the OCSF schema doesn't include it by default
	switch className {
	case "detection_finding":
		return []ExtraField{
			{
				FieldName: "Attacks",
				GoType:    "[]*objects.Attack",
				JSONTag:   "attacks",
			},
		}
	default:
		return nil
	}
}
