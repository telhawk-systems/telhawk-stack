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
		log.Printf("  Rule '%s' exists but content changed, updating", rule.Name)
		return imp.updateRule(ctx, ruleID, rule, contentHash)
	}

	log.Printf("  Creating new rule '%s'", rule.Name)
	return imp.createRule(ctx, rule, contentHash)
}

// checkRuleExists checks if a rule already exists and returns its content hash
func (imp *Importer) checkRuleExists(ctx context.Context, ruleID string) (bool, string, error) {
	url := fmt.Sprintf("%s/%s", imp.rulesURL, ruleID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		body, _ := io.ReadAll(resp.Body)
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

// createRule creates a new rule
func (imp *Importer) createRule(ctx context.Context, rule RuleFile, contentHash string) error {
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

	// Create request body
	body, err := json.Marshal(rule)
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("  Successfully created rule '%s'", rule.Name)
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
		body, _ := io.ReadAll(resp.Body)
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

// calculateContentHash calculates a hash of the rule content
// Used to detect if a rule has changed
func calculateContentHash(rule RuleFile) string {
	// Serialize rule to JSON (deterministic order)
	data, _ := json.Marshal(map[string]interface{}{
		"model":      rule.Model,
		"view":       rule.View,
		"controller": rule.Controller,
	})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes for shorter hash
}
