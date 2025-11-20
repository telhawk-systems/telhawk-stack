package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// RuleFile represents the minimal structure needed to read rule names
type RuleFile struct {
	Name string `json:"name"`
}

// generateDeterministicUUID generates a UUID v5 based on rule name
// This MUST match the algorithm in alerting/internal/importer/importer.go
func generateDeterministicUUID(ruleName string) string {
	// Use a namespace UUID for TelHawk system rules
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
	return uuid.NewSHA1(namespace, []byte("telhawk:builtin:"+ruleName)).String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <rules-directory>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s alerting/rules\n", os.Args[0])
		os.Exit(1)
	}

	rulesDir := os.Args[1]

	// Find all .json files (but not .json.id files)
	pattern := filepath.Join(rulesDir, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to list rule files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No .json files found in %s\n", rulesDir)
		os.Exit(1)
	}

	createdCount := 0
	skippedCount := 0
	errorCount := 0

	for _, filePath := range files {
		// Skip .json.id files
		if strings.HasSuffix(filePath, ".json.id") {
			continue
		}

		// Read and parse the rule file
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to read %s: %v\n", filePath, err)
			errorCount++
			continue
		}

		var rule RuleFile
		if err := json.Unmarshal(data, &rule); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to parse %s: %v\n", filePath, err)
			errorCount++
			continue
		}

		if rule.Name == "" {
			fmt.Fprintf(os.Stderr, "ERROR: Rule in %s has no name field\n", filePath)
			errorCount++
			continue
		}

		// Generate deterministic UUID
		ruleID := generateDeterministicUUID(rule.Name)
		idFilePath := filePath + ".id"

		// Check if .id file already exists with correct content
		existingData, err := os.ReadFile(idFilePath)
		if err == nil {
			existingID := strings.TrimSpace(string(existingData))
			if existingID == ruleID {
				fmt.Printf("✓ %s (ID already correct)\n", filepath.Base(filePath))
				skippedCount++
				continue
			}
			fmt.Printf("⚠ %s (updating ID from %s to %s)\n", filepath.Base(filePath), existingID, ruleID)
		}

		// Write the .id file
		if err := os.WriteFile(idFilePath, []byte(ruleID+"\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to write %s: %v\n", idFilePath, err)
			errorCount++
			continue
		}

		fmt.Printf("✓ Created %s with ID: %s\n", filepath.Base(idFilePath), ruleID)
		createdCount++
	}

	fmt.Println()
	fmt.Printf("Summary: %d created/updated, %d already correct, %d errors\n", createdCount, skippedCount, errorCount)

	if errorCount > 0 {
		os.Exit(1)
	}
}
