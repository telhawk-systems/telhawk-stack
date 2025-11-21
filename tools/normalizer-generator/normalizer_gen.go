package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// generateNormalizer creates a normalizer file for a specific event class
func generateNormalizer(className string, pattern Pattern, outputDir string) error {
	var buf strings.Builder

	// Package and imports
	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"encoding/json\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"strings\"\n\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/ingest/internal/models\"\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/common/ocsf\"\n")

	// Add category-specific imports
	if pattern.Category != "" {
		buf.WriteString(fmt.Sprintf("\t\"github.com/telhawk-systems/telhawk-stack/common/ocsf/events/%s\"\n", pattern.Category))
	}
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/common/ocsf/objects\"\n")
	buf.WriteString(")\n\n")

	structName := toGoStructName(className) + "Normalizer"

	// Struct definition
	buf.WriteString(fmt.Sprintf("// %s normalizes %s events to OCSF format\n", structName, className))
	buf.WriteString(fmt.Sprintf("type %s struct{}\n\n", structName))

	// Constructor
	buf.WriteString(fmt.Sprintf("// New%s creates a new normalizer\n", structName))
	buf.WriteString(fmt.Sprintf("func New%s() *%s {\n", structName, structName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{}\n", structName))
	buf.WriteString("}\n\n")

	// Supports method
	buf.WriteString(generateSupportsMethod(structName, pattern))

	// Normalize method
	buf.WriteString(generateNormalizeMethod(structName, className, pattern))

	// Write to file
	filename := filepath.Join(outputDir, className+"_normalizer.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}

// generateSupportsMethod creates the Supports method for pattern matching
func generateSupportsMethod(structName string, pattern Pattern) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("// Supports checks if this normalizer handles the given event\n"))
	buf.WriteString(fmt.Sprintf("func (n *%s) Supports(format, sourceType string) bool {\n", structName))
	buf.WriteString("\tif format != \"json\" {\n")
	buf.WriteString("\t\treturn false\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\tst := strings.ToLower(sourceType)\n")
	buf.WriteString("\treturn ")

	// Generate pattern checks
	conditions := make([]string, 0)
	for _, p := range pattern.SourceTypePatterns {
		// Convert regex patterns to contains checks (simplified)
		pattern := strings.TrimPrefix(p, "^")
		conditions = append(conditions, fmt.Sprintf("strings.Contains(st, %q)", pattern))
	}

	buf.WriteString(strings.Join(conditions, " || \n\t       "))
	buf.WriteString("\n}\n\n")

	return buf.String()
}

// generateNormalizeMethod creates the Normalize method for event transformation
func generateNormalizeMethod(structName, className string, pattern Pattern) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("// Normalize converts raw event to OCSF %s\n", className))
	buf.WriteString(fmt.Sprintf("func (n *%s) Normalize(ctx context.Context, envelope *models.RawEventEnvelope) (*ocsf.Event, error) {\n", structName))
	buf.WriteString("\tvar payload map[string]interface{}\n")
	buf.WriteString("\tif err := json.Unmarshal(envelope.Payload, &payload); err != nil {\n")
	buf.WriteString("\t\treturn nil, fmt.Errorf(\"decode payload: %w\", err)\n")
	buf.WriteString("\t}\n\n")

	// Determine activity ID if keywords defined
	if len(pattern.ActivityKeywords) > 0 {
		buf.WriteString("\tactivityID := n.determineActivityID(payload)\n")
	} else {
		buf.WriteString("\tactivityID := 0 // Default activity\n")
	}

	// Create event (simplified - assumes constructor exists)
	categoryPkg := pattern.Category
	constructorName := fmt.Sprintf("New%s", toGoStructName(className))
	buf.WriteString(fmt.Sprintf("\tevent := %s.%s(activityID)\n\n", categoryPkg, constructorName))

	// Set timing
	buf.WriteString("\tevent.Time = ExtractTimestamp(payload, envelope.ReceivedAt)\n")
	buf.WriteString("\tevent.ObservedTime = envelope.ReceivedAt\n\n")

	// Extract common fields
	buf.WriteString("\t// Extract common fields\n")
	buf.WriteString("\tif user := ExtractUser(payload); user != nil {\n")
	buf.WriteString("\t\tevent.Actor = &objects.Actor{User: user}\n")
	buf.WriteString("\t}\n\n")

	buf.WriteString("\tevent.StatusID, event.Status = ExtractStatus(payload)\n")
	buf.WriteString("\tevent.SeverityID, event.Severity = ExtractSeverity(payload)\n\n")

	// Set metadata
	buf.WriteString("\tevent.Metadata.LogProvider = envelope.Source\n")
	buf.WriteString("\tevent.Raw = ocsf.RawDescriptor{Format: envelope.Format, Data: payload}\n\n")

	buf.WriteString("\treturn &event.Event, nil\n")
	buf.WriteString("}\n\n")

	// Add activity determination if needed
	if len(pattern.ActivityKeywords) > 0 {
		buf.WriteString(generateActivityDetermination(structName, pattern))
	}

	return buf.String()
}

// generateActivityDetermination creates a method to determine activity ID from keywords
func generateActivityDetermination(structName string, pattern Pattern) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("func (n *%s) determineActivityID(payload map[string]interface{}) int {\n", structName))
	buf.WriteString("\taction := strings.ToLower(ExtractString(payload, \"action\", \"event_type\", \"activity\"))\n\n")

	// Generate checks for each activity
	activityKeys := make([]string, 0, len(pattern.ActivityKeywords))
	for k := range pattern.ActivityKeywords {
		activityKeys = append(activityKeys, k)
	}
	sort.Strings(activityKeys)

	for i, activity := range activityKeys {
		keywords := pattern.ActivityKeywords[activity]
		conditions := make([]string, 0)
		for _, kw := range keywords {
			conditions = append(conditions, fmt.Sprintf("strings.Contains(action, %q)", kw))
		}

		if i == 0 {
			buf.WriteString("\tif ")
		} else {
			buf.WriteString("\t} else if ")
		}
		buf.WriteString(strings.Join(conditions, " || "))
		buf.WriteString(" {\n")
		buf.WriteString(fmt.Sprintf("\t\treturn %d // %s\n", i+1, activity))
	}

	buf.WriteString("\t}\n")
	buf.WriteString("\treturn 0 // Unknown\n")
	buf.WriteString("}\n\n")

	return buf.String()
}

// generateNormalizersRegistry generates a registry file that includes all normalizers
func generateNormalizersRegistry(allClasses []*EventClass, outputDir string) error {
	var buf strings.Builder

	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/ingest/internal/normalizer\"\n")
	buf.WriteString(")\n\n")

	buf.WriteString("// AllNormalizers returns all generated normalizers\n")
	buf.WriteString("func AllNormalizers() []normalizer.Normalizer {\n")
	buf.WriteString("\treturn []normalizer.Normalizer{\n")

	for _, class := range allClasses {
		if class.UID == 0 {
			continue // Skip base_event
		}
		structName := toGoStructName(class.Name) + "Normalizer"
		buf.WriteString(fmt.Sprintf("\t\tNew%s(),\n", structName))
	}

	buf.WriteString("\t}\n")
	buf.WriteString("}\n")

	filename := filepath.Join(outputDir, "normalizers_registry.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}
