package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// LoadCLI loads configuration for CLI tools.
// Uses $HOME/.thawk as the default TELHAWK_CONFIG_DIR if not set.
func LoadCLI() (*CLIConfig, error) {
	v := viper.New()

	// Set CLI defaults
	v.SetDefault("current_profile", "default")
	v.SetDefault("defaults.auth_url", "http://localhost:3000")
	v.SetDefault("defaults.ingest_url", "http://localhost:8088")
	v.SetDefault("defaults.query_url", "http://localhost:3000")
	v.SetDefault("defaults.rules_url", "http://localhost:3000")
	v.SetDefault("defaults.alerting_url", "http://localhost:3000")

	// Determine config directory for CLI
	configDir := os.Getenv("TELHAWK_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine home directory: %w", err)
		}
		configDir = filepath.Join(home, ".thawk")
	}

	// Read config file
	configPath := filepath.Join(configDir, "config.yaml")
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Environment variables override with THAWK prefix
	v.SetEnvPrefix("THAWK")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind specific env vars (viper needs explicit bindings for nested keys)
	_ = v.BindEnv("defaults.auth_url", "THAWK_AUTH_URL")
	_ = v.BindEnv("defaults.ingest_url", "THAWK_INGEST_URL")
	_ = v.BindEnv("defaults.query_url", "THAWK_QUERY_URL")
	_ = v.BindEnv("defaults.rules_url", "THAWK_RULES_URL")
	_ = v.BindEnv("defaults.alerting_url", "THAWK_ALERTING_URL")

	// Also try alternate format (dots replaced with underscores)
	_ = v.BindEnv("defaults.auth_url", "THAWK_DEFAULTS_AUTH_URL")
	_ = v.BindEnv("defaults.ingest_url", "THAWK_DEFAULTS_INGEST_URL")
	_ = v.BindEnv("defaults.query_url", "THAWK_DEFAULTS_QUERY_URL")
	_ = v.BindEnv("defaults.rules_url", "THAWK_DEFAULTS_RULES_URL")
	_ = v.BindEnv("defaults.alerting_url", "THAWK_DEFAULTS_ALERTING_URL")

	// Read config file - don't fail if file doesn't exist
	_ = v.ReadInConfig() // Ignore errors - file may not exist yet

	cfg := DefaultCLI()
	cfg.path = configPath

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

// MustLoadCLI loads CLI configuration and panics on error.
func MustLoadCLI() *CLIConfig {
	cfg, err := LoadCLI()
	if err != nil {
		panic(fmt.Sprintf("failed to load CLI config: %v", err))
	}
	return cfg
}
