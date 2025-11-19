package seeder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete seeder configuration
type Config struct {
	Version  string                  `mapstructure:"version" yaml:"version"`
	Defaults DefaultsConfig          `mapstructure:"defaults" yaml:"defaults"`
	Attacks  map[string]AttackConfig `mapstructure:"attacks" yaml:"attacks"`
	Rules    map[string]RuleConfig   `mapstructure:"rules" yaml:"rules"`
}

// DefaultsConfig holds default seeder settings
type DefaultsConfig struct {
	HECURL     string        `mapstructure:"hec_url" yaml:"hec_url"`
	Token      string        `mapstructure:"token" yaml:"token"`
	Count      int           `mapstructure:"count" yaml:"count"`
	TimeSpread time.Duration `mapstructure:"time_spread" yaml:"time_spread"`
	BatchSize  int           `mapstructure:"batch_size" yaml:"batch_size"`
	Interval   time.Duration `mapstructure:"interval" yaml:"interval"`
	EventTypes []string      `mapstructure:"event_types" yaml:"event_types"`
}

// AttackConfig defines an attack pattern configuration
type AttackConfig struct {
	Pattern string                 `mapstructure:"pattern" yaml:"pattern"`
	Enabled bool                   `mapstructure:"enabled" yaml:"enabled"`
	Params  map[string]interface{} `mapstructure:"params" yaml:"params"`
}

// RuleConfig defines a rule-based event generation configuration
type RuleConfig struct {
	RuleFile   string                 `mapstructure:"rule_file" yaml:"rule_file"`
	Enabled    bool                   `mapstructure:"enabled" yaml:"enabled"`
	Multiplier float64                `mapstructure:"multiplier" yaml:"multiplier"`
	Params     map[string]interface{} `mapstructure:"params" yaml:"params"`
}

// LoadConfig loads configuration with cascade: flags > ./seeder.yaml > ~/.thawk/seeder.yaml > defaults
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure Viper
	v.SetConfigName("seeder")
	v.SetConfigType("yaml")
	v.SetEnvPrefix("SEEDER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Config file search paths (in priority order)
	if configPath != "" {
		// Explicit config path from flag
		v.SetConfigFile(configPath)
	} else {
		// 1. Current directory
		v.AddConfigPath(".")

		// 2. User home directory (~/.thawk/)
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".thawk"))
		}
	}

	// Read config file (optional - don't fail if not found)
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			// Config file found but had error reading it
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found - that's OK, we have defaults
	}

	// Unmarshal into struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Version
	v.SetDefault("version", "1.0")

	// Defaults
	v.SetDefault("defaults.hec_url", "http://localhost:8088")
	v.SetDefault("defaults.count", 50000)
	v.SetDefault("defaults.time_spread", 90*24*time.Hour) // 90 days
	v.SetDefault("defaults.batch_size", 50)
	v.SetDefault("defaults.interval", 0)
	v.SetDefault("defaults.event_types", []string{
		"auth", "network", "process", "file", "dns", "http", "detection",
	})

	// No default token - must be provided
	v.SetDefault("defaults.token", "")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check time spread for event density
	if c.Defaults.TimeSpread > 0 {
		days := c.Defaults.TimeSpread / (24 * time.Hour)
		eventsPerDay := float64(c.Defaults.Count) / float64(days)
		if eventsPerDay < 1.0 {
			return fmt.Errorf("event density too low: %.2f events/day (need at least 1 event/day)", eventsPerDay)
		}
	}

	// Validate attack configurations
	for name, attack := range c.Attacks {
		if attack.Pattern == "" {
			return fmt.Errorf("attack %s: pattern is required", name)
		}
	}

	// Validate rule configurations
	for name, rule := range c.Rules {
		if rule.RuleFile == "" {
			return fmt.Errorf("rule %s: rule_file is required", name)
		}
		// Set default multiplier if not specified
		if rule.Multiplier == 0 {
			rule.Multiplier = 1.5
			c.Rules[name] = rule
		}
	}

	return nil
}

// GetEnabledAttacks returns only enabled attack configurations
func (c *Config) GetEnabledAttacks() map[string]AttackConfig {
	enabled := make(map[string]AttackConfig)
	for name, attack := range c.Attacks {
		if attack.Enabled {
			enabled[name] = attack
		}
	}
	return enabled
}

// GetAttack returns a specific attack configuration by name
func (c *Config) GetAttack(name string) (AttackConfig, bool) {
	attack, ok := c.Attacks[name]
	return attack, ok
}

// GetEnabledRules returns only enabled rule configurations
func (c *Config) GetEnabledRules() map[string]RuleConfig {
	enabled := make(map[string]RuleConfig)
	for name, rule := range c.Rules {
		if rule.Enabled {
			enabled[name] = rule
		}
	}
	return enabled
}

// GetRule returns a specific rule configuration by name
func (c *Config) GetRule(name string) (RuleConfig, bool) {
	rule, ok := c.Rules[name]
	return rule, ok
}
