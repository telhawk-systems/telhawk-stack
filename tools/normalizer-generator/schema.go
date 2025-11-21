package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EventClass represents an OCSF event class definition
type EventClass struct {
	UID         int                        `json:"uid"`
	Caption     string                     `json:"caption"`
	Name        string                     `json:"name"`
	Category    string                     `json:"category"`
	Description string                     `json:"description"`
	Extends     string                     `json:"extends"`
	Attributes  map[string]json.RawMessage `json:"attributes"`
}

// Categories holds OCSF category definitions
type Categories struct {
	Attributes map[string]Category `json:"attributes"`
}

// Category represents an OCSF event category
type Category struct {
	UID         int    `json:"uid"`
	Caption     string `json:"caption"`
	Description string `json:"description"`
}

// Dictionary holds OCSF dictionary attributes
type Dictionary struct {
	Attributes map[string]DictionaryAttribute `json:"attributes"`
}

// DictionaryAttribute represents a dictionary field definition
type DictionaryAttribute struct {
	Caption     string `json:"caption"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Attribute represents a single OCSF attribute with its validation constraints
type Attribute struct {
	Requirement string                     `json:"requirement"` // required, recommended, optional
	Group       string                     `json:"group"`
	Type        string                     `json:"type"`
	Enum        map[string]json.RawMessage `json:"enum,omitempty"` // For enumerated values
}

// loadEventClass reads and parses an event class JSON file
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

// generateFromSchema scans OCSF schema directory and generates normalizers and validators
func generateFromSchema(schemaEventsDir string, outputDir string) (int, error) {
	totalGenerated := 0
	allValidationRules := make([]*ValidationRules, 0)
	allClasses := make([]*EventClass, 0)

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

			// Override category with the actual directory name (not the extends field)
			class.Category = categoryName

			// Track this class for the registry
			allClasses = append(allClasses, class)

			// Generate normalizer
			pattern := getPatternForClass(class)
			if err := generateNormalizer(class.Name, pattern, outputDir); err != nil {
				if *verbose {
					fmt.Printf("Warning: failed to generate normalizer for %s: %v\n", class.Name, err)
				}
				continue
			}

			// Extract validation rules and generate validator
			rules, err := extractValidationRules(class)
			if err != nil {
				if *verbose {
					fmt.Printf("Warning: failed to extract validation rules for %s: %v\n", class.Name, err)
				}
			} else {
				allValidationRules = append(allValidationRules, rules)
				if err := generateValidator(rules, outputDir); err != nil {
					if *verbose {
						fmt.Printf("Warning: failed to generate validator for %s: %v\n", class.Name, err)
					}
				} else if *verbose {
					fmt.Printf("Generated: %s_validator.go (required: %d, recommended: %d, enums: %d)\n",
						class.Name, len(rules.RequiredFields), len(rules.RecommendedFields), len(rules.EnumFields))
				}
			}

			totalGenerated++
			if *verbose {
				fmt.Printf("Generated: %s_normalizer.go (class_uid=%d, category=%s)\n", class.Name, class.UID, categoryName)
			}
		}
	}

	// Generate validators registry
	if err := generateValidatorsRegistry(allValidationRules, outputDir); err != nil {
		if *verbose {
			fmt.Printf("Warning: failed to generate validators registry: %v\n", err)
		}
	} else if *verbose {
		fmt.Printf("Generated: validators_registry.go (%d validators)\n", len(allValidationRules))
	}

	// Generate normalizers registry
	if err := generateNormalizersRegistry(allClasses, outputDir); err != nil {
		if *verbose {
			fmt.Printf("Warning: failed to generate normalizers registry: %v\n", err)
		}
	} else if *verbose {
		fmt.Printf("Generated: normalizers_registry.go (%d normalizers)\n", len(allClasses))
	}

	return totalGenerated, nil
}

// generateFromEventClasses scans generated event classes and creates normalizers
func generateFromEventClasses(generatedEventsDir string, outputDir string) (int, error) {
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
			if err := generateNormalizer(class.Name, pattern, outputDir); err != nil {
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
