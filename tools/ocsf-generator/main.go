package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	schemaDir = flag.String("schema", "../../ocsf-schema", "Path to OCSF schema directory")
	outputDir = flag.String("output", "../../core/pkg/ocsf", "Output directory for generated code")
	verbose   = flag.Bool("v", false, "Verbose output")

	// Global dictionary loaded once and used throughout generation
	globalDictionary Dictionary
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load categories
	categories, err := loadCategories(filepath.Join(*schemaDir, "categories.json"))
	if err != nil {
		return fmt.Errorf("load categories: %w", err)
	}

	// Load dictionary
	dictionary, err := loadDictionary(filepath.Join(*schemaDir, "dictionary.json"))
	if err != nil {
		return fmt.Errorf("load dictionary: %w", err)
	}

	// Store globally for type inference
	globalDictionary = dictionary

	// Create output directories
	eventsDir := filepath.Join(*outputDir, "events")
	objectsDir := filepath.Join(*outputDir, "objects")
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return fmt.Errorf("create events dir: %w", err)
	}
	if err := os.MkdirAll(objectsDir, 0755); err != nil {
		return fmt.Errorf("create objects dir: %w", err)
	}

	// Track generated event class names to avoid duplicate object types
	generatedEventTypes := make(map[string]bool)

	// Collect all object types needed
	objectTypes := make(map[string]bool)

	// Process each category
	schemaEventsDir := filepath.Join(*schemaDir, "events")
	categoryDirs, err := os.ReadDir(schemaEventsDir)
	if err != nil {
		return fmt.Errorf("read events dir: %w", err)
	}

	totalClasses := 0
	for _, categoryDir := range categoryDirs {
		if !categoryDir.IsDir() {
			continue
		}

		categoryName := categoryDir.Name()
		categoryPath := filepath.Join(schemaEventsDir, categoryName)

		// Create category subdirectory
		categoryOutputDir := filepath.Join(eventsDir, categoryName)
		if err := os.MkdirAll(categoryOutputDir, 0755); err != nil {
			return fmt.Errorf("create category dir %s: %w", categoryName, err)
		}

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

			// Skip category base files (e.g., iam.json)
			if eventFile.Name() == categoryName+".json" {
				continue
			}

			eventPath := filepath.Join(categoryPath, eventFile.Name())
			class, err := loadEventClass(eventPath)
			if err != nil {
				if *verbose {
					fmt.Printf("Warning: cannot load %s: %v\n", eventPath, err)
				}
				continue
			}

			// Set category name from directory
			class.Category = categoryName

			// Compute full class UID (category_uid * 1000 + class_uid)
			categoryUID := getCategoryUID(categoryName, categories)
			class.UID = categoryUID*1000 + class.UID

			// Track this as a generated event type
			generatedEventTypes[toGoStructName(class.Name)] = true

			// Collect object types from this class
			collectObjectTypes(class, dictionary, objectTypes)

			// Generate class file in category subdirectory
			if err := generateEventClassFile(class, categoryUID, categoryOutputDir, categoryName); err != nil {
				return fmt.Errorf("generate class %s: %w", class.Name, err)
			}

			totalClasses++
			if *verbose {
				fmt.Printf("Generated: %s/%s (UID %d)\n", categoryName, class.Name, class.UID)
			}
		}
	}

	// Generate placeholder object types (excluding event types)
	for eventType := range generatedEventTypes {
		delete(objectTypes, eventType)
	}

	// Generate actual object types from schema
	if err := generateObjects(objectsDir); err != nil {
		return fmt.Errorf("generate objects: %w", err)
	}

	// Generate mappings file for category, class, and activity names
	if err := generateMappings(categories, *outputDir); err != nil {
		return fmt.Errorf("generate mappings: %w", err)
	}

	fmt.Printf("✅ Generated %d OCSF event classes in %s/events/\n", totalClasses, *outputDir)
	fmt.Printf("✅ Generated OCSF object types in %s/objects/\n", *outputDir)
	fmt.Printf("✅ Generated OCSF mappings in %s/mappings.go\n", *outputDir)

	// Validate no generic JSON types were generated
	if err := validateNoGenericTypes(*outputDir); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	fmt.Printf("✅ Type safety validation passed\n")

	return nil
}
