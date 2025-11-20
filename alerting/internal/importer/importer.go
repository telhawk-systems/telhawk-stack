package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// RuleFile represents a detection rule loaded from JSON
type RuleFile struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Model       map[string]interface{} `json:"model"`
	View        map[string]interface{} `json:"view"`
	Controller  map[string]interface{} `json:"controller"`
}

// CreateRuleRequest matches the rules service API
type CreateRuleRequest struct {
	ID         string                 `json:"id,omitempty"`
	Model      map[string]interface{} `json:"model"`
	View       map[string]interface{} `json:"view"`
	Controller map[string]interface{} `json:"controller"`
}

// Importer handles importing detection rules from JSON files
type Importer struct {
	rulesDir   string
	rulesURL   string
	httpClient *http.Client
}

// NewImporter creates a new rule importer
func NewImporter(rulesDir, rulesURL string) *Importer {
	return &Importer{
		rulesDir: rulesDir,
		rulesURL: rulesURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Import loads all rules from the rules directory and imports them
func (imp *Importer) Import(ctx context.Context) error {
	log.Printf("Starting rule import from %s", imp.rulesDir)

	// Check if directory exists
	if _, err := os.Stat(imp.rulesDir); os.IsNotExist(err) {
		log.Printf("Rules directory does not exist: %s", imp.rulesDir)
		return nil // Not an error, just no rules to import
	}

	// Read all JSON files
	files, err := filepath.Glob(filepath.Join(imp.rulesDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list rule files: %w", err)
	}

	if len(files) == 0 {
		log.Printf("No rule files found in %s", imp.rulesDir)
		return nil
	}

	log.Printf("Found %d rule file(s) to import", len(files))

	successCount := 0
	errorCount := 0

	for _, file := range files {
		if err := imp.importRuleFile(ctx, file); err != nil {
			log.Printf("ERROR: Failed to import %s: %v", filepath.Base(file), err)
			errorCount++
		} else {
			successCount++
		}
	}

	log.Printf("Rule import complete: %d succeeded, %d failed", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("%d rule(s) failed to import", errorCount)
	}

	return nil
}

// importRuleFile imports a single rule file
func (imp *Importer) importRuleFile(ctx context.Context, filePath string) error {
	fileName := filepath.Base(filePath)
	log.Printf("Importing rule from %s", fileName)

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse rule
	var rule RuleFile
	if err := json.Unmarshal(data, &rule); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if rule.Name == "" {
		return fmt.Errorf("rule missing required field: name")
	}
	if rule.Model == nil {
		return fmt.Errorf("rule missing required field: model")
	}
	if rule.View == nil {
		return fmt.Errorf("rule missing required field: view")
	}

	// Generate deterministic UUID based on rule name
	ruleID := generateDeterministicUUID(rule.Name)

	// Validate that the .id file matches the generated ID (CRITICAL for consistency across environments)
	if err := validateRuleID(filePath, rule.Name, ruleID); err != nil {
		return fmt.Errorf("CRITICAL: Rule ID validation failed for '%s': %w", rule.Name, err)
	}

	// Check if rule already exists
	exists, currentVersion, err := imp.checkRuleExists(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("failed to check if rule exists: %w", err)
	}

	// Calculate content hash to detect changes
	contentHash := calculateContentHash(rule)

	if exists {
		// Check if content has changed
		if currentVersion == contentHash {
			log.Printf("  Rule '%s' already exists with same content, skipping", rule.Name)
			return nil
		}
		// Builtin rules exist but content changed - skip update to avoid conflicts
		// TODO: Implement proper versioning for builtin rule updates
		log.Printf("  WARN: Rule '%s' exists with different content, but updates are skipped (builtin protection)", rule.Name)
		return nil
	}

	log.Printf("  Creating new rule '%s'", rule.Name)
	return imp.createRule(ctx, ruleID, rule, contentHash)
}

// checkRuleExists checks if a rule already exists and returns its content hash
func (imp *Importer) checkRuleExists(ctx context.Context, ruleID string) (bool, string, error) {
	url := fmt.Sprintf("%s/%s", imp.rulesURL, ruleID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return false, "", err
	}

	resp, err := imp.httpClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, "", nil
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, "", fmt.Errorf("unexpected status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return false, "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get content hash from metadata
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, "", err
	}

	// Extract content hash from controller metadata
	if controller, ok := response["controller"].(map[string]interface{}); ok {
		if metadata, ok := controller["metadata"].(map[string]interface{}); ok {
			if hash, ok := metadata["content_hash"].(string); ok {
				return true, hash, nil
			}
		}
	}

	return true, "", nil
}

// createRule creates a new rule with deterministic ID
func (imp *Importer) createRule(ctx context.Context, ruleID string, rule RuleFile, contentHash string) error {
	// Add metadata to controller
	if rule.Controller == nil {
		rule.Controller = make(map[string]interface{})
	}

	metadata := map[string]interface{}{
		"source":       "builtin",
		"content_hash": contentHash,
		"imported_at":  time.Now().UTC().Format(time.RFC3339),
	}
	rule.Controller["metadata"] = metadata

	// Create request with deterministic ID
	reqBody := CreateRuleRequest{
		ID:         ruleID,
		Model:      rule.Model,
		View:       rule.View,
		Controller: rule.Controller,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", imp.rulesURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := imp.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("create failed with status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("create failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("  Successfully created rule '%s' with ID %s", rule.Name, ruleID)
	return nil
}

// updateRule updates an existing rule
func (imp *Importer) updateRule(ctx context.Context, ruleID string, rule RuleFile, contentHash string) error {
	// Add metadata to controller
	if rule.Controller == nil {
		rule.Controller = make(map[string]interface{})
	}

	metadata := map[string]interface{}{
		"source":       "builtin",
		"content_hash": contentHash,
		"updated_at":   time.Now().UTC().Format(time.RFC3339),
	}
	rule.Controller["metadata"] = metadata

	// Create request body
	body, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	url := fmt.Sprintf("%s/%s", imp.rulesURL, ruleID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := imp.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("update failed with status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("  Successfully updated rule '%s'", rule.Name)
	return nil
}

// generateDeterministicUUID generates a UUID v5 based on rule name
// This ensures the same rule name always gets the same ID
func generateDeterministicUUID(ruleName string) string {
	// Use a namespace UUID for TelHawk system rules
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
	return uuid.NewSHA1(namespace, []byte("telhawk:builtin:"+ruleName)).String()
}

// validateRuleID validates that the .id file exists and contains the expected deterministic UUID
// This ensures consistency across all environments and prevents ID drift
func validateRuleID(jsonFilePath, ruleName, expectedID string) error {
	// Construct the .id file path (e.g., /path/to/rule.json -> /path/to/rule.json.id)
	idFilePath := jsonFilePath + ".id"

	// Read the .id file
	idData, err := os.ReadFile(idFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(".id file not found at %s - all rules MUST have committed .id files for deterministic UUIDs across environments", idFilePath)
		}
		return fmt.Errorf("failed to read .id file: %w", err)
	}

	// Parse the ID from the file (trim whitespace/newlines)
	fileID := string(bytes.TrimSpace(idData))

	// Validate UUID format
	if _, err := uuid.Parse(fileID); err != nil {
		return fmt.Errorf(".id file contains invalid UUID '%s': %w", fileID, err)
	}

	// CRITICAL: Ensure the .id file matches the deterministic UUID
	if fileID != expectedID {
		return fmt.Errorf(
			"ID MISMATCH: .id file contains '%s' but rule name '%s' generates '%s'. "+
				"This indicates the .id file is out of sync. "+
				"Run tools/generate_rule_ids.go to regenerate .id files, then commit to git.",
			fileID, ruleName, expectedID,
		)
	}

	log.Printf("  Validated rule ID: %s matches %s", ruleName, fileID)
	return nil
}

// calculateContentHash calculates a hash of the rule content
// Used to detect if a rule has changed
func calculateContentHash(rule RuleFile) string {
	// Serialize rule to JSON (deterministic order)
	data, err := json.Marshal(map[string]interface{}{
		"model":      rule.Model,
		"view":       rule.View,
		"controller": rule.Controller,
	})
	if err != nil {
		// This should never happen for a simple map, but handle it
		log.Printf("Warning: failed to marshal rule for hashing: %v", err)
		return ""
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes for shorter hash
}
