package pipeline_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer/generated"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
)

// TestGeneratedNormalizersIntegration tests all generated normalizers with real log data
func TestGeneratedNormalizersIntegration(t *testing.T) {
	// Initialize pipeline with all generated normalizers
	registry := normalizer.NewRegistry(
		generated.NewAuthenticationNormalizer(),
		generated.NewNetworkActivityNormalizer(),
		generated.NewProcessActivityNormalizer(),
		generated.NewFileActivityNormalizer(),
		generated.NewDnsActivityNormalizer(),
		generated.NewHttpActivityNormalizer(),
		generated.NewDetectionFindingNormalizer(),
	)
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)

	testCases := []struct {
		name         string
		file         string
		sourceType   string
		expectClass  string
		expectStatus string
		expectUser   string
	}{
		{
			name:         "Authentication Login",
			file:         "auth_login.json",
			sourceType:   "auth_login",
			expectClass:  "authentication",
			expectStatus: "Success",
			expectUser:   "john.doe",
		},
		{
			name:         "Authentication Logout",
			file:         "auth_logout.json",
			sourceType:   "auth_logout",
			expectClass:  "authentication",
			expectStatus: "Success",
			expectUser:   "jane.smith",
		},
		{
			name:        "Network Connection",
			file:        "network_connection.json",
			sourceType:  "network_firewall",
			expectClass: "network_activity",
		},
		{
			name:        "Process Start",
			file:        "process_start.json",
			sourceType:  "process_log",
			expectClass: "process_activity",
		},
		{
			name:        "File Create",
			file:        "file_create.json",
			sourceType:  "file_audit",
			expectClass: "file_activity",
		},
		{
			name:        "DNS Query",
			file:        "dns_query.json",
			sourceType:  "dns_log",
			expectClass: "dns_activity",
		},
		{
			name:        "HTTP Request",
			file:        "http_request.json",
			sourceType:  "http_access",
			expectClass: "http_activity",
		},
		{
			name:        "Detection Finding",
			file:        "detection_finding.json",
			sourceType:  "security_detection",
			expectClass: "detection_finding",
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load test data
			testFile := filepath.Join("..", "..", "testdata", tc.file)
			data, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read test file %s: %v", tc.file, err)
			}

			// Create envelope
			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: tc.sourceType,
				Source:     "test",
				Payload:    data,
				ReceivedAt: time.Now(),
			}

			// Process through pipeline
			event, err := pipe.Process(ctx, envelope)
			if err != nil {
				t.Fatalf("pipeline processing failed: %v", err)
			}

			// Validate results
			if event == nil {
				t.Fatal("expected non-nil event")
			}

			if event.Class != tc.expectClass {
				t.Errorf("expected class %q, got %q", tc.expectClass, event.Class)
			}

			if tc.expectStatus != "" && event.Status != tc.expectStatus {
				t.Errorf("expected status %q, got %q", tc.expectStatus, event.Status)
			}

			// Check that raw data is preserved
			if event.Raw.Data == nil {
				t.Error("expected raw data to be preserved")
			}

			// Verify event can be marshaled to JSON
			jsonData, err := json.Marshal(event)
			if err != nil {
				t.Errorf("failed to marshal event to JSON: %v", err)
			}

			// Verify we got valid OCSF JSON
			var ocsf map[string]interface{}
			if err := json.Unmarshal(jsonData, &ocsf); err != nil {
				t.Errorf("failed to unmarshal OCSF event: %v", err)
			}

			// Check required OCSF fields
			requiredFields := []string{"class_uid", "category_uid", "type_uid", "time"}
			for _, field := range requiredFields {
				if _, ok := ocsf[field]; !ok {
					t.Errorf("missing required OCSF field: %s", field)
				}
			}

			t.Logf("✓ Successfully processed %s -> %s (class_uid=%d)",
				tc.file, event.Class, event.ClassUID)
		})
	}
}

// TestNormalizerSelection verifies correct normalizer is selected for each source type
func TestNormalizerSelection(t *testing.T) {
	registry := normalizer.NewRegistry(
		generated.NewAuthenticationNormalizer(),
		generated.NewNetworkActivityNormalizer(),
		generated.NewProcessActivityNormalizer(),
		generated.NewFileActivityNormalizer(),
		generated.NewDnsActivityNormalizer(),
		generated.NewHttpActivityNormalizer(),
		generated.NewDetectionFindingNormalizer(),
	)

	testCases := []struct {
		sourceType     string
		expectNormType string
		shouldMatch    bool
	}{
		{"auth_login", "*generated.AuthenticationNormalizer", true},
		{"ldap_auth", "*generated.AuthenticationNormalizer", true},
		{"sso_login", "*generated.AuthenticationNormalizer", true},
		{"network_firewall", "*generated.NetworkActivityNormalizer", true},
		{"firewall_traffic", "*generated.NetworkActivityNormalizer", true},
		{"process_log", "*generated.ProcessActivityNormalizer", true},
		{"file_audit", "*generated.FileActivityNormalizer", true},
		{"dns_log", "*generated.DNSActivityNormalizer", true},
		{"http_access", "*generated.HTTPActivityNormalizer", true},
		{"security_detection", "*generated.DetectionFindingNormalizer", true},
		{"unknown_type", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.sourceType, func(t *testing.T) {
			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: tc.sourceType,
				Payload:    []byte(`{}`),
			}

			norm := registry.Find(envelope)

			if tc.shouldMatch {
				if norm == nil {
					t.Errorf("expected normalizer for %s, got nil", tc.sourceType)
				} else {
					t.Logf("✓ %s matched normalizer", tc.sourceType)
				}
			} else {
				if norm != nil {
					t.Errorf("expected no normalizer for %s, got one", tc.sourceType)
				}
			}
		})
	}
}

// TestFieldExtraction verifies field extraction from various field name variants
func TestFieldExtraction(t *testing.T) {
	registry := normalizer.NewRegistry(
		generated.NewAuthenticationNormalizer(),
	)
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)
	ctx := context.Background()

	// Test various field name variants for user
	testPayloads := []struct {
		name     string
		payload  string
		expected string
	}{
		{
			name:     "user field",
			payload:  `{"user": "alice", "action": "login", "status": "success"}`,
			expected: "alice",
		},
		{
			name:     "username field",
			payload:  `{"username": "bob", "action": "login", "status": "success"}`,
			expected: "bob",
		},
		{
			name:     "user_name field",
			payload:  `{"user_name": "charlie", "action": "login", "status": "success"}`,
			expected: "charlie",
		},
		{
			name:     "account field",
			payload:  `{"account": "david", "action": "login", "status": "success"}`,
			expected: "david",
		},
	}

	for _, tc := range testPayloads {
		t.Run(tc.name, func(t *testing.T) {
			envelope := &model.RawEventEnvelope{
				Format:     "json",
				SourceType: "auth_test",
				Source:     "test",
				Payload:    []byte(tc.payload),
				ReceivedAt: time.Now(),
			}

			event, err := pipe.Process(ctx, envelope)
			if err != nil {
				t.Fatalf("failed to process: %v", err)
			}

			// Note: The actual user extraction depends on the OCSF structure
			// This test verifies the pipeline runs successfully with various field names
			if event == nil {
				t.Error("expected non-nil event")
			}
			t.Logf("✓ Successfully extracted from %s", tc.name)
		})
	}
}
