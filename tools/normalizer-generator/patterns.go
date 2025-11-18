package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SourceTypePatterns holds pattern configurations for matching events
type SourceTypePatterns struct {
	Patterns map[string]Pattern `json:"patterns"`
}

// Pattern defines how to match and classify events
type Pattern struct {
	ClassUID           int                 `json:"class_uid"`
	Category           string              `json:"category"`
	SourceTypePatterns []string            `json:"sourcetype_patterns"`
	ContentPatterns    []string            `json:"content_patterns"`
	Priority           int                 `json:"priority"`
	ActivityKeywords   map[string][]string `json:"activity_keywords,omitempty"`
}

// FieldMappings defines how to map raw event fields to OCSF fields
type FieldMappings struct {
	CommonFields map[string]FieldMapping `json:"common_fields"`
}

// FieldMapping describes a single field mapping
type FieldMapping struct {
	OCSFField string   `json:"ocsf_field"`
	Variants  []string `json:"variants"`
	Type      string   `json:"type"`
	Parser    string   `json:"parser,omitempty"`
}

var (
	fieldMappings      FieldMappings
	sourceTypePatterns SourceTypePatterns
)

// loadPatternConfig loads pattern configurations from JSON files
func loadPatternConfig() error {
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

// getPatternForClass returns the matching pattern for an event class
func getPatternForClass(class *EventClass) Pattern {
	// Check for explicit override in sourcetype_patterns.json
	if override, exists := sourceTypePatterns.Patterns[class.Name]; exists {
		return override
	}

	// Generate default pattern from class name
	patterns := generateDefaultPatterns(class.Name)

	return Pattern{
		ClassUID:           class.UID,
		Category:           class.Category, // Set by caller (directory name)
		SourceTypePatterns: patterns,
		Priority:           50, // Default priority
	}
}

// generateDefaultPatterns creates default sourcetype patterns from a class name
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
