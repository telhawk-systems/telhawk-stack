package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	schemaDir = flag.String("schema", "../../ocsf-schema", "Path to OCSF schema directory")
	outputDir = flag.String("output", "../../ingest/internal/normalizer/generated", "Output directory for generated code")
	verbose   = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load pattern configuration (field mappings and sourcetype patterns)
	if err := loadPatternConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Generate shared helpers file first
	if err := generateHelpersFile(*outputDir); err != nil {
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
		totalGenerated, err = generateFromSchema(schemaEventsDir, *outputDir)
		if err != nil {
			return fmt.Errorf("generate from schema: %w", err)
		}
	} else {
		// Schema not available - scan generated event classes
		if *verbose {
			fmt.Println("Schema not found, scanning generated event classes...")
		}
		generatedEventsDir := filepath.Join("../../core/pkg/ocsf/events")
		totalGenerated, err = generateFromEventClasses(generatedEventsDir, *outputDir)
		if err != nil {
			return fmt.Errorf("generate from event classes: %w", err)
		}
	}

	fmt.Printf("✅ Generated %d normalizers + %d validators + helpers + registry in %s/\n", totalGenerated, totalGenerated, *outputDir)
	return nil
}
