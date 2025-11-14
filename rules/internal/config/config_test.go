package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Load config without providing a file path (empty string uses defaults)
	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify defaults
	assert.Equal(t, 8084, cfg.Server.Port)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)

	assert.Equal(t, "postgres", cfg.Database.Type)
	assert.Equal(t, "require", cfg.Database.Postgres.SSLMode)

	assert.Equal(t, "24h", cfg.Validation.MaxTimeWindow)
	assert.Equal(t, 100000, cfg.Validation.MaxThreshold)
	assert.Equal(t, []string{"count", "sum", "avg", "max", "min"}, cfg.Validation.AllowedAggregations)

	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_FromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 9090
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

database:
  type: postgres
  postgres:
    host: testhost
    port: 5433
    database: testdb
    user: testuser
    password: testpass
    sslmode: disable

validation:
  max_time_window: 48h
  max_threshold: 50000
  allowed_aggregations:
    - count
    - sum

logging:
  level: debug
  format: text
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config from file
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify values from file
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.Server.IdleTimeout)

	assert.Equal(t, "postgres", cfg.Database.Type)
	assert.Equal(t, "testhost", cfg.Database.Postgres.Host)
	assert.Equal(t, 5433, cfg.Database.Postgres.Port)
	assert.Equal(t, "testdb", cfg.Database.Postgres.Database)
	assert.Equal(t, "testuser", cfg.Database.Postgres.User)
	assert.Equal(t, "testpass", cfg.Database.Postgres.Password)
	assert.Equal(t, "disable", cfg.Database.Postgres.SSLMode)

	assert.Equal(t, "48h", cfg.Validation.MaxTimeWindow)
	assert.Equal(t, 50000, cfg.Validation.MaxThreshold)
	assert.Equal(t, []string{"count", "sum"}, cfg.Validation.AllowedAggregations)

	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("RULES_SERVER_PORT", "7777")
	os.Setenv("RULES_DATABASE_POSTGRES_HOST", "envhost")
	os.Setenv("RULES_DATABASE_POSTGRES_PORT", "5555")
	os.Setenv("RULES_LOGGING_LEVEL", "warn")
	defer func() {
		os.Unsetenv("RULES_SERVER_PORT")
		os.Unsetenv("RULES_DATABASE_POSTGRES_HOST")
		os.Unsetenv("RULES_DATABASE_POSTGRES_PORT")
		os.Unsetenv("RULES_LOGGING_LEVEL")
	}()

	// Create temporary config file with different values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8084

database:
  postgres:
    host: filehost
    port: 5432

logging:
  level: info
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config - environment variables should override file values
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify environment overrides took effect
	assert.Equal(t, 7777, cfg.Server.Port, "Environment variable should override file value")
	assert.Equal(t, "envhost", cfg.Database.Postgres.Host, "Environment variable should override file value")
	assert.Equal(t, 5555, cfg.Database.Postgres.Port, "Environment variable should override file value")
	assert.Equal(t, "warn", cfg.Logging.Level, "Environment variable should override file value")
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
server:
  port: not_a_number
  invalid yaml here [[[
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Load config - should return error
	cfg, err := Load(configPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_EmptyPath(t *testing.T) {
	// Load with empty path - should use defaults and search for config in standard locations
	// This will not find a config file in test environment, so it should use defaults
	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify defaults are used
	assert.Equal(t, 8084, cfg.Server.Port)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestLoad_PartialConfig(t *testing.T) {
	// Create config file with only some values set
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	partialConfig := `
server:
  port: 9999

logging:
  level: debug
`

	err := os.WriteFile(configPath, []byte(partialConfig), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify specified values
	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Logging.Level)

	// Verify defaults for unspecified values
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout, "Should use default")
	assert.Equal(t, "require", cfg.Database.Postgres.SSLMode, "Should use default")
	assert.Equal(t, 100000, cfg.Validation.MaxThreshold, "Should use default")
}
