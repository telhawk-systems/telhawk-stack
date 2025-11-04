package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Configuration structures
type FieldMappings struct {
	CommonFields map[string]FieldMapping `json:"common_fields"`
}

type FieldMapping struct {
	OCSFField string   `json:"ocsf_field"`
	Variants  []string `json:"variants"`
	Type      string   `json:"type"`
	Parser    string   `json:"parser,omitempty"`
}

type SourceTypePatterns struct {
	Patterns map[string]Pattern `json:"patterns"`
}

type Pattern struct {
	ClassUID            int                 `json:"class_uid"`
	Category            string              `json:"category"`
	SourceTypePatterns  []string            `json:"sourcetype_patterns"`
	ContentPatterns     []string            `json:"content_patterns"`
	Priority            int                 `json:"priority"`
	ActivityKeywords    map[string][]string `json:"activity_keywords,omitempty"`
}

type EventClass struct {
	UID         int                        `json:"uid"`
	Caption     string                     `json:"caption"`
	Name        string                     `json:"name"`
	Category    string                     `json:"category"`
	Description string                     `json:"description"`
	Extends     string                     `json:"extends"`
	Attributes  map[string]json.RawMessage `json:"attributes"`
}

type Categories struct {
	Attributes map[string]Category `json:"attributes"`
}

type Category struct {
	UID         int    `json:"uid"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

var (
	schemaDir  = flag.String("schema", "../../ocsf-schema", "Path to OCSF schema directory")
	outputDir  = flag.String("output", "../../core/internal/normalizer/generated", "Output directory for generated code")
	verbose    = flag.Bool("v", false, "Verbose output")
	
	fieldMappings      FieldMappings
	sourceTypePatterns SourceTypePatterns
	categories         Categories
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	if err := loadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Generate shared helpers file first
	if err := generateHelpersFile(); err != nil {
		return fmt.Errorf("generate helpers: %w", err)
	}

	// Try to scan OCSF schema if available, otherwise use generated event classes
	totalGenerated := 0
	schemaEventsDir := filepath.Join(*schemaDir, "events")
	if _, err := os.Stat(schemaEventsDir); err == nil {
		// Schema available - use it
		if *verbose {
			fmt.Println("Using OCSF schema from:", *schemaDir)
		}
		totalGenerated, err = generateFromSchema(schemaEventsDir)
		if err != nil {
			return fmt.Errorf("generate from schema: %w", err)
		}
	} else {
		// Schema not available - scan generated event classes
		if *verbose {
			fmt.Println("Schema not found, scanning generated event classes...")
		}
		generatedEventsDir := filepath.Join("../../core/pkg/ocsf/events")
		totalGenerated, err = generateFromEventClasses(generatedEventsDir)
		if err != nil {
			return fmt.Errorf("generate from event classes: %w", err)
		}
	}

	fmt.Printf("✅ Generated %d normalizers + helpers in %s/\n", totalGenerated, *outputDir)
	return nil
}

func loadConfig() error {
	// Load field mappings
	mappingsData, err := os.ReadFile("field_mappings.json")
	if err != nil {
		return fmt.Errorf("read field_mappings.json: %w", err)
	}
	if err := json.Unmarshal(mappingsData, &fieldMappings); err != nil {
		return fmt.Errorf("parse field_mappings.json: %w", err)
	}

	// Load sourcetype patterns (for overrides - optional)
	patternsData, err := os.ReadFile("sourcetype_patterns.json")
	if err != nil {
		// OK if doesn't exist - we'll use defaults
		if *verbose {
			fmt.Println("No sourcetype_patterns.json found, using auto-generated patterns")
		}
		sourceTypePatterns.Patterns = make(map[string]Pattern)
	} else {
		if err := json.Unmarshal(patternsData, &sourceTypePatterns); err != nil {
			return fmt.Errorf("parse sourcetype_patterns.json: %w", err)
		}
		if *verbose {
			fmt.Printf("Loaded %d pattern overrides from sourcetype_patterns.json\n", len(sourceTypePatterns.Patterns))
		}
	}

	return nil
}

func loadEventClass(path string) (*EventClass, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var class EventClass
	if err := json.Unmarshal(data, &class); err != nil {
		return nil, err
	}

	return &class, nil
}

func getPatternForClass(class *EventClass) Pattern {
	// Check for explicit override in sourcetype_patterns.json
	if override, exists := sourceTypePatterns.Patterns[class.Name]; exists {
		return override
	}

	// Generate default pattern from class name
	patterns := generateDefaultPatterns(class.Name)
	
	return Pattern{
		ClassUID:           class.UID,
		Category:           class.Category,
		SourceTypePatterns: patterns,
		Priority:           50, // Default priority
	}
}

func generateDefaultPatterns(className string) []string {
	patterns := []string{className}
	
	// Add common variations
	parts := strings.Split(className, "_")
	if len(parts) > 1 {
		// Add first part (e.g., "auth" from "authentication")
		patterns = append(patterns, parts[0])
	}
	
	// Add simplified version without "_activity" suffix
	simplified := strings.TrimSuffix(className, "_activity")
	if simplified != className {
		patterns = append(patterns, simplified)
	}
	
	return patterns
}

func generateFromSchema(schemaEventsDir string) (int, error) {
	totalGenerated := 0
	categoryDirs, err := os.ReadDir(schemaEventsDir)
	if err != nil {
		return 0, err
	}

	for _, categoryDir := range categoryDirs {
		if !categoryDir.IsDir() {
			continue
		}

		categoryName := categoryDir.Name()
		categoryPath := filepath.Join(schemaEventsDir, categoryName)
		
		eventFiles, err := os.ReadDir(categoryPath)
		if err != nil {
			if *verbose {
				fmt.Printf("Warning: cannot read category %s: %v\n", categoryName, err)
			}
			continue
		}

		for _, eventFile := range eventFiles {
			if !strings.HasSuffix(eventFile.Name(), ".json") {
				continue
			}

			// Skip category base files
			if eventFile.Name() == categoryName+".json" {
				continue
			}

			eventPath := filepath.Join(categoryPath, eventFile.Name())
			class, err := loadEventClass(eventPath)
			if err != nil {
				if *verbose {
					fmt.Printf("Warning: failed to load %s: %v\n", eventFile.Name(), err)
				}
				continue
			}

			pattern := getPatternForClass(class)
			if err := generateNormalizer(class.Name, pattern); err != nil {
				if *verbose {
					fmt.Printf("Warning: failed to generate normalizer for %s: %v\n", class.Name, err)
				}
				continue
			}
			
			totalGenerated++
			if *verbose {
				fmt.Printf("Generated: %s_normalizer.go (class_uid=%d, category=%s)\n", class.Name, class.UID, categoryName)
			}
		}
	}

	return totalGenerated, nil
}

func generateFromEventClasses(generatedEventsDir string) (int, error) {
	totalGenerated := 0
	categoryDirs, err := os.ReadDir(generatedEventsDir)
	if err != nil {
		return 0, fmt.Errorf("read generated events dir: %w", err)
	}

	for _, categoryDir := range categoryDirs {
		if !categoryDir.IsDir() {
			continue
		}

		categoryName := categoryDir.Name()
		categoryPath := filepath.Join(generatedEventsDir, categoryName)
		
		eventFiles, err := os.ReadDir(categoryPath)
		if err != nil {
			if *verbose {
				fmt.Printf("Warning: cannot read category %s: %v\n", categoryName, err)
			}
			continue
		}

		for _, eventFile := range eventFiles {
			if !strings.HasSuffix(eventFile.Name(), ".go") {
				continue
			}

			// Extract class name from filename (e.g., "authentication.go" → "authentication")
			className := strings.TrimSuffix(eventFile.Name(), ".go")
			
			// Create a minimal EventClass for pattern generation
			class := &EventClass{
				Name:     className,
				Category: categoryName,
				UID:      0, // Will be filled from override if exists
			}

			pattern := getPatternForClass(class)
			if err := generateNormalizer(class.Name, pattern); err != nil {
				if *verbose {
					fmt.Printf("Warning: failed to generate normalizer for %s: %v\n", class.Name, err)
				}
				continue
			}
			
			totalGenerated++
			if *verbose {
				fmt.Printf("Generated: %s_normalizer.go (category=%s)\n", class.Name, categoryName)
			}
		}
	}

	return totalGenerated, nil
}

func generateNormalizer(className string, pattern Pattern) error {
	var buf strings.Builder

	// Package and imports
	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"encoding/json\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"strings\"\n\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/internal/model\"\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf\"\n")
	
	// Add category-specific imports
	if pattern.Category != "" {
		buf.WriteString(fmt.Sprintf("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf/events/%s\"\n", pattern.Category))
	}
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf/objects\"\n")
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

	// Helper methods
	buf.WriteString(generateHelperMethods())

	// Write to file
	filename := filepath.Join(*outputDir, className+"_normalizer.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}

func generateHeader() string {
	return `// Code generated by normalizer-generator. DO NOT EDIT.
//
// This file was automatically generated from OCSF schema and field mappings.

`
}

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

func generateNormalizeMethod(structName, className string, pattern Pattern) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("// Normalize converts raw event to OCSF %s\n", className))
	buf.WriteString(fmt.Sprintf("func (n *%s) Normalize(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {\n", structName))
	buf.WriteString("\tvar payload map[string]interface{}\n")
	buf.WriteString("\tif err := json.Unmarshal(envelope.Payload, &payload); err != nil {\n")
	buf.WriteString("\t\treturn nil, fmt.Errorf(\"decode payload: %%w\", err)\n")
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

func generateHelperMethods() string {
	// Helpers are now in a separate file
	return ""
}

func toGoStructName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

func generateHelpersFile() error {
	var buf strings.Builder

	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"strings\"\n")
	buf.WriteString("\t\"time\"\n\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf\"\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf/objects\"\n")
	buf.WriteString(")\n\n")

	buf.WriteString(`// Shared helper methods for field extraction
// These are used by all generated normalizers

// ExtractString tries multiple field names and returns the first non-empty string
func ExtractString(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := payload[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

// ExtractUser extracts user information from common field names
func ExtractUser(payload map[string]interface{}) *objects.User {
	name := ExtractString(payload, "user", "username", "user_name", "account", "principal")
	if name == "" {
		return nil
	}
	return &objects.User{
		Name:   name,
		Uid:    ExtractString(payload, "user_id", "uid", "user_uid"),
		Domain: ExtractString(payload, "domain", "user_domain", "realm"),
	}
}

// ExtractTimestamp parses timestamps from various formats
func ExtractTimestamp(payload map[string]interface{}, fallback time.Time) time.Time {
	for _, field := range []string{"timestamp", "time", "@timestamp", "event_time", "datetime"} {
		if val, ok := payload[field]; ok {
			switch v := val.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					return t
				}
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					return t
				}
			case float64:
				return time.Unix(int64(v), 0)
			case int64:
				return time.Unix(v, 0)
			}
		}
	}
	return fallback
}

// ExtractStatus maps status strings to OCSF status codes
func ExtractStatus(payload map[string]interface{}) (int, string) {
	status := strings.ToLower(ExtractString(payload, "status", "result", "outcome"))
	switch {
	case strings.Contains(status, "success") || strings.Contains(status, "ok"):
		return ocsf.StatusSuccess, "Success"
	case strings.Contains(status, "fail") || strings.Contains(status, "error"):
		return ocsf.StatusFailure, "Failure"
	default:
		return ocsf.StatusUnknown, "Unknown"
	}
}

// ExtractSeverity maps severity strings to OCSF severity codes
func ExtractSeverity(payload map[string]interface{}) (int, string) {
	sev := strings.ToLower(ExtractString(payload, "severity", "level", "priority"))
	switch {
	case strings.Contains(sev, "critical") || strings.Contains(sev, "fatal"):
		return ocsf.SeverityCritical, "Critical"
	case strings.Contains(sev, "high") || strings.Contains(sev, "error"):
		return ocsf.SeverityHigh, "High"
	case strings.Contains(sev, "medium") || strings.Contains(sev, "warn"):
		return ocsf.SeverityMedium, "Medium"
	case strings.Contains(sev, "low") || strings.Contains(sev, "info"):
		return ocsf.SeverityLow, "Low"
	default:
		return ocsf.SeverityUnknown, "Unknown"
	}
}
`)

	filename := filepath.Join(*outputDir, "helpers.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}
