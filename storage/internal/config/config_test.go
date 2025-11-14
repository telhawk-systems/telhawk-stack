package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	// Test server defaults
	if cfg.Server.Port != 8083 {
		t.Errorf("Expected default port 8083, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("Expected default read timeout 15s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 15*time.Second {
		t.Errorf("Expected default write timeout 15s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 60*time.Second {
		t.Errorf("Expected default idle timeout 60s, got %v", cfg.Server.IdleTimeout)
	}

	// Test OpenSearch defaults
	if cfg.OpenSearch.URL != "http://opensearch:9200" {
		t.Errorf("Expected default OpenSearch URL http://opensearch:9200, got %s", cfg.OpenSearch.URL)
	}
	if cfg.OpenSearch.Username != "admin" {
		t.Errorf("Expected default username admin, got %s", cfg.OpenSearch.Username)
	}
	if cfg.OpenSearch.Password != "admin" {
		t.Errorf("Expected default password admin, got %s", cfg.OpenSearch.Password)
	}
	if !cfg.OpenSearch.Insecure {
		t.Error("Expected default insecure to be true")
	}

	// Test index management defaults
	if cfg.IndexManagement.IndexPrefix != "telhawk-events" {
		t.Errorf("Expected default index prefix telhawk-events, got %s", cfg.IndexManagement.IndexPrefix)
	}
	if cfg.IndexManagement.ShardCount != 1 {
		t.Errorf("Expected default shard count 1, got %d", cfg.IndexManagement.ShardCount)
	}
	if cfg.IndexManagement.ReplicaCount != 0 {
		t.Errorf("Expected default replica count 0, got %d", cfg.IndexManagement.ReplicaCount)
	}
	if cfg.IndexManagement.RefreshInterval != "5s" {
		t.Errorf("Expected default refresh interval 5s, got %s", cfg.IndexManagement.RefreshInterval)
	}
	if cfg.IndexManagement.RetentionDays != 30 {
		t.Errorf("Expected default retention days 30, got %d", cfg.IndexManagement.RetentionDays)
	}
	if cfg.IndexManagement.RolloverSizeGB != 50 {
		t.Errorf("Expected default rollover size 50GB, got %d", cfg.IndexManagement.RolloverSizeGB)
	}
	if cfg.IndexManagement.RolloverAge != 24*time.Hour {
		t.Errorf("Expected default rollover age 24h, got %v", cfg.IndexManagement.RolloverAge)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: 9999
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

opensearch:
  url: "https://test-opensearch:9200"
  username: "testuser"
  password: "testpass"
  insecure: false

index_management:
  index_prefix: "test-events"
  shard_count: 3
  replica_count: 2
  refresh_interval: "10s"
  retention_days: 60
  rollover_size_gb: 100
  rollover_age: 48h
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify server config
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Expected read timeout 30s, got %v", cfg.Server.ReadTimeout)
	}

	// Verify OpenSearch config
	if cfg.OpenSearch.URL != "https://test-opensearch:9200" {
		t.Errorf("Expected URL https://test-opensearch:9200, got %s", cfg.OpenSearch.URL)
	}
	if cfg.OpenSearch.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", cfg.OpenSearch.Username)
	}
	if cfg.OpenSearch.Password != "testpass" {
		t.Errorf("Expected password testpass, got %s", cfg.OpenSearch.Password)
	}
	if cfg.OpenSearch.Insecure {
		t.Error("Expected insecure to be false")
	}

	// Verify index management config
	if cfg.IndexManagement.IndexPrefix != "test-events" {
		t.Errorf("Expected index prefix test-events, got %s", cfg.IndexManagement.IndexPrefix)
	}
	if cfg.IndexManagement.ShardCount != 3 {
		t.Errorf("Expected shard count 3, got %d", cfg.IndexManagement.ShardCount)
	}
	if cfg.IndexManagement.ReplicaCount != 2 {
		t.Errorf("Expected replica count 2, got %d", cfg.IndexManagement.ReplicaCount)
	}
	if cfg.IndexManagement.RetentionDays != 60 {
		t.Errorf("Expected retention days 60, got %d", cfg.IndexManagement.RetentionDays)
	}
	if cfg.IndexManagement.RolloverSizeGB != 100 {
		t.Errorf("Expected rollover size 100GB, got %d", cfg.IndexManagement.RolloverSizeGB)
	}
	if cfg.IndexManagement.RolloverAge != 48*time.Hour {
		t.Errorf("Expected rollover age 48h, got %v", cfg.IndexManagement.RolloverAge)
	}
}

func TestLoadConfigWithEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("OPENSEARCH_URL", "https://env-opensearch:9200")
	os.Setenv("OPENSEARCH_USERNAME", "envuser")
	os.Setenv("OPENSEARCH_PASSWORD", "envpass")
	defer func() {
		os.Unsetenv("OPENSEARCH_URL")
		os.Unsetenv("OPENSEARCH_USERNAME")
		os.Unsetenv("OPENSEARCH_PASSWORD")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override defaults
	if cfg.OpenSearch.URL != "https://env-opensearch:9200" {
		t.Errorf("Expected URL from env https://env-opensearch:9200, got %s", cfg.OpenSearch.URL)
	}
	if cfg.OpenSearch.Username != "envuser" {
		t.Errorf("Expected username from env envuser, got %s", cfg.OpenSearch.Username)
	}
	if cfg.OpenSearch.Password != "envpass" {
		t.Errorf("Expected password from env envpass, got %s", cfg.OpenSearch.Password)
	}
}

func TestLoadConfigWithEnvOverridesFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
opensearch:
  url: "https://file-opensearch:9200"
  username: "fileuser"
  password: "filepass"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables (should override file)
	os.Setenv("OPENSEARCH_URL", "https://env-opensearch:9200")
	os.Setenv("OPENSEARCH_USERNAME", "envuser")
	os.Setenv("OPENSEARCH_PASSWORD", "envpass")
	defer func() {
		os.Unsetenv("OPENSEARCH_URL")
		os.Unsetenv("OPENSEARCH_USERNAME")
		os.Unsetenv("OPENSEARCH_PASSWORD")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override file values
	if cfg.OpenSearch.URL != "https://env-opensearch:9200" {
		t.Errorf("Expected URL from env https://env-opensearch:9200, got %s", cfg.OpenSearch.URL)
	}
	if cfg.OpenSearch.Username != "envuser" {
		t.Errorf("Expected username from env envuser, got %s", cfg.OpenSearch.Username)
	}
	if cfg.OpenSearch.Password != "envpass" {
		t.Errorf("Expected password from env envpass, got %s", cfg.OpenSearch.Password)
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	invalidYAML := `
server:
  port: not_a_number
  invalid yaml here
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}
