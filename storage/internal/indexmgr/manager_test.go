package indexmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/storage/internal/client"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/config"
)

func TestNewIndexManager(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, err := client.NewOpenSearchClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenSearch client: %v", err)
	}

	indexCfg := config.IndexManagementConfig{
		IndexPrefix:     "test-events",
		ShardCount:      1,
		ReplicaCount:    0,
		RefreshInterval: "5s",
		RetentionDays:   30,
		RolloverSizeGB:  50,
		RolloverAge:     24 * time.Hour,
	}

	mgr := NewIndexManager(osClient, indexCfg)

	if mgr == nil {
		t.Fatal("Expected non-nil index manager")
	}

	if mgr.client != osClient {
		t.Error("Expected client to be set correctly")
	}

	if mgr.config.IndexPrefix != "test-events" {
		t.Errorf("Expected index prefix test-events, got %s", mgr.config.IndexPrefix)
	}
}

func TestGetCurrentWriteIndex(t *testing.T) {
	now := time.Now()
	expected := fmt.Sprintf("test-events-%s-000001", now.Format("2006.01.02"))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix: "test-events",
	}

	mgr := NewIndexManager(osClient, indexCfg)
	indexName := mgr.GetCurrentWriteIndex()

	if indexName != expected {
		t.Errorf("Expected index name %s, got %s", expected, indexName)
	}
}

func TestGetWriteAlias(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix: "test-events",
	}

	mgr := NewIndexManager(osClient, indexCfg)
	alias := mgr.GetWriteAlias()

	expected := "test-events-write"
	if alias != expected {
		t.Errorf("Expected write alias %s, got %s", expected, alias)
	}
}

func TestResolveIndexPattern(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix: "test-events",
	}

	mgr := NewIndexManager(osClient, indexCfg)

	tests := []struct {
		name     string
		classUID int
		expected string
	}{
		{"Authentication events", 3002, "test-events-*"},
		{"Process events", 1007, "test-events-*"},
		{"Network events", 4001, "test-events-*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := mgr.ResolveIndexPattern(tt.classUID)
			if pattern != tt.expected {
				t.Errorf("Expected pattern %s, got %s", tt.expected, pattern)
			}
		})
	}
}

func TestFormatDurationForOpenSearch(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"1 day", 24 * time.Hour, "1d"},
		{"7 days", 7 * 24 * time.Hour, "7d"},
		{"30 days", 30 * 24 * time.Hour, "30d"},
		{"12 hours", 12 * time.Hour, "12h"},
		{"36 hours", 36 * time.Hour, "36h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationForOpenSearch(tt.duration)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInitialize(t *testing.T) {
	// Track which endpoints were called
	templateCreated := false
	policyCreated := false
	indexCreated := false
	aliasUpdated := false

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/":
			// Info endpoint
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))

		case strings.HasPrefix(r.URL.Path, "/_index_template/"):
			// Index template
			templateCreated = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged": true}`))

		case strings.HasPrefix(r.URL.Path, "/_plugins/_ism/policies/"):
			// ISM policy
			if r.Method == "GET" {
				// Policy doesn't exist
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "not found"}`))
			} else {
				// Create policy
				policyCreated = true
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"_id": "test-events-policy", "_version": 1}`))
			}

		case strings.HasPrefix(r.URL.Path, "/test-events-"):
			// Index operations
			if r.Method == "HEAD" {
				// Index doesn't exist
				w.WriteHeader(http.StatusNotFound)
			} else if r.Method == "PUT" {
				// Create index
				indexCreated = true
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"acknowledged": true, "index": "test-events-2025.01.15-000001"}`))
			}

		case r.URL.Path == "/_aliases":
			// Update alias
			aliasUpdated = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged": true}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		}
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, err := client.NewOpenSearchClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenSearch client: %v", err)
	}

	indexCfg := config.IndexManagementConfig{
		IndexPrefix:     "test-events",
		ShardCount:      1,
		ReplicaCount:    0,
		RefreshInterval: "5s",
		RetentionDays:   30,
		RolloverSizeGB:  50,
		RolloverAge:     24 * time.Hour,
	}

	mgr := NewIndexManager(osClient, indexCfg)

	ctx := context.Background()
	err = mgr.Initialize(ctx)
	if err != nil {
		t.Fatalf("Expected successful initialization, got error: %v", err)
	}

	if !templateCreated {
		t.Error("Expected index template to be created")
	}
	if !policyCreated {
		t.Error("Expected ISM policy to be created")
	}
	if !indexCreated {
		t.Error("Expected initial index to be created")
	}
	if !aliasUpdated {
		t.Error("Expected write alias to be updated")
	}
}

func TestGetOCSFMappings(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix: "test-events",
	}

	mgr := NewIndexManager(osClient, indexCfg)
	mappings := mgr.getOCSFMappings()

	// Verify dynamic settings
	if mappings["dynamic"] != true {
		t.Error("Expected dynamic mapping to be enabled")
	}

	// Verify dynamic_templates exists
	if _, ok := mappings["dynamic_templates"]; !ok {
		t.Error("Expected dynamic_templates to be present")
	}

	// Verify properties exists
	properties, ok := mappings["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Test critical OCSF fields are present
	requiredFields := []string{
		"time",
		"@timestamp",
		"metadata",
		"class_uid",
		"class_name",
		"category_uid",
		"category_name",
		"activity_id",
		"type_uid",
		"severity",
		"severity_id",
		"status",
		"message",
		"user",
		"actor",
		"process",
		"file",
		"src_endpoint",
		"dst_endpoint",
	}

	for _, field := range requiredFields {
		if _, ok := properties[field]; !ok {
			t.Errorf("Expected field %s to be present in mappings", field)
		}
	}

	// Verify time field is date type
	timeField, ok := properties["time"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected time field to be a map")
	}
	if timeField["type"] != "date" {
		t.Errorf("Expected time field type to be date, got %v", timeField["type"])
	}

	// Verify class_uid is integer type
	classUIDField, ok := properties["class_uid"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected class_uid field to be a map")
	}
	if classUIDField["type"] != "integer" {
		t.Errorf("Expected class_uid field type to be integer, got %v", classUIDField["type"])
	}
}

func TestCreateIndexTemplate(t *testing.T) {
	var receivedTemplate map[string]interface{}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))

		case strings.HasPrefix(r.URL.Path, "/_index_template/"):
			// Read and parse the template
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedTemplate)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"acknowledged": true}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix:     "test-events",
		ShardCount:      2,
		ReplicaCount:    1,
		RefreshInterval: "10s",
		RetentionDays:   60,
		RolloverSizeGB:  100,
		RolloverAge:     48 * time.Hour,
	}

	mgr := NewIndexManager(osClient, indexCfg)

	ctx := context.Background()
	err := mgr.createIndexTemplate(ctx)
	if err != nil {
		t.Fatalf("Expected successful template creation, got error: %v", err)
	}

	// Verify template structure
	if receivedTemplate == nil {
		t.Fatal("Expected template to be received")
	}

	// Verify index patterns
	indexPatterns, ok := receivedTemplate["index_patterns"].([]interface{})
	if !ok || len(indexPatterns) == 0 {
		t.Error("Expected index_patterns to be set")
	}

	// Verify template settings
	template, ok := receivedTemplate["template"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected template section to be present")
	}

	settings, ok := template["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected settings to be present")
	}

	// JSON unmarshaling converts numbers to float64
	if shards, ok := settings["number_of_shards"].(float64); !ok || int(shards) != 2 {
		t.Errorf("Expected 2 shards, got %v (type: %T)", settings["number_of_shards"], settings["number_of_shards"])
	}

	if replicas, ok := settings["number_of_replicas"].(float64); !ok || int(replicas) != 1 {
		t.Errorf("Expected 1 replica, got %v (type: %T)", settings["number_of_replicas"], settings["number_of_replicas"])
	}

	if settings["refresh_interval"] != "10s" {
		t.Errorf("Expected refresh_interval 10s, got %v", settings["refresh_interval"])
	}
}

func TestCreateISMPolicy(t *testing.T) {
	var receivedPolicy map[string]interface{}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "test-node", "version": {"number": "2.0.0"}}`))

		case strings.HasPrefix(r.URL.Path, "/_plugins/_ism/policies/"):
			if r.Method == "GET" {
				// Policy doesn't exist
				w.WriteHeader(http.StatusNotFound)
			} else {
				// Create/update policy
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &receivedPolicy)

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"_id": "test-events-policy", "_version": 1}`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	cfg := config.OpenSearchConfig{
		URL:      mockServer.URL,
		Username: "admin",
		Password: "admin",
		Insecure: true,
	}

	osClient, _ := client.NewOpenSearchClient(cfg)

	indexCfg := config.IndexManagementConfig{
		IndexPrefix:    "test-events",
		RetentionDays:  60,
		RolloverSizeGB: 100,
		RolloverAge:    48 * time.Hour,
	}

	mgr := NewIndexManager(osClient, indexCfg)

	ctx := context.Background()
	err := mgr.createISMPolicy(ctx)
	if err != nil {
		t.Fatalf("Expected successful policy creation, got error: %v", err)
	}

	// Verify policy structure
	if receivedPolicy == nil {
		t.Fatal("Expected policy to be received")
	}

	policy, ok := receivedPolicy["policy"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected policy section to be present")
	}

	if policy["description"] == nil {
		t.Error("Expected policy description to be set")
	}

	if policy["default_state"] != "hot" {
		t.Errorf("Expected default_state to be hot, got %v", policy["default_state"])
	}

	// Verify states
	states, ok := policy["states"].([]interface{})
	if !ok || len(states) != 2 {
		t.Error("Expected 2 states (hot and delete)")
	}
}
