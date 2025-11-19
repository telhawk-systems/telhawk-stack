package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CurrentProfile string              `yaml:"current_profile" mapstructure:"current_profile"`
	Profiles       map[string]*Profile `yaml:"profiles" mapstructure:"profiles"`
	Defaults       *Defaults           `yaml:"defaults" mapstructure:"defaults"`
	path           string
}

type Profile struct {
	AuthURL      string `yaml:"auth_url" mapstructure:"auth_url"`
	IngestURL    string `yaml:"ingest_url" mapstructure:"ingest_url"`
	QueryURL     string `yaml:"query_url" mapstructure:"query_url"`
	RulesURL     string `yaml:"rules_url" mapstructure:"rules_url"`
	AlertingURL  string `yaml:"alerting_url" mapstructure:"alerting_url"`
	AccessToken  string `yaml:"access_token" mapstructure:"access_token"`
	RefreshToken string `yaml:"refresh_token" mapstructure:"refresh_token"`
}

type Defaults struct {
	AuthURL     string `yaml:"auth_url" mapstructure:"auth_url"`
	IngestURL   string `yaml:"ingest_url" mapstructure:"ingest_url"`
	QueryURL    string `yaml:"query_url" mapstructure:"query_url"`
	RulesURL    string `yaml:"rules_url" mapstructure:"rules_url"`
	AlertingURL string `yaml:"alerting_url" mapstructure:"alerting_url"`
}

func Default() *Config {
	return &Config{
		CurrentProfile: "default",
		Profiles:       make(map[string]*Profile),
		Defaults: &Defaults{
			AuthURL:     "http://localhost:8080",
			IngestURL:   "http://localhost:8088",
			QueryURL:    "http://localhost:8082",
			RulesURL:    "http://localhost:8084",
			AlertingURL: "http://localhost:8085",
		},
	}
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("current_profile", "default")
	v.SetDefault("defaults.auth_url", "http://localhost:8080")
	v.SetDefault("defaults.ingest_url", "http://localhost:8088")
	v.SetDefault("defaults.query_url", "http://localhost:8082")
	v.SetDefault("defaults.rules_url", "http://localhost:8084")
	v.SetDefault("defaults.alerting_url", "http://localhost:8085")

	// Determine config file path
	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cfgFile = filepath.Join(home, ".thawk", "config.yaml")
		}
		// If we can't determine home dir, just skip config file
	}

	// Environment variable overrides
	v.SetEnvPrefix("THAWK")
	v.AutomaticEnv()

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

	cfg := Default()
	cfg.path = cfgFile

	// Read config file (optional - skip if path is empty or doesn't exist)
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		v.SetConfigType("yaml")

		// Try to read config file - ignore any errors
		_ = v.ReadInConfig()
	}

	// Unmarshal into config struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if c.path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(home, ".thawk", "config.yaml")
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}

func (c *Config) SaveProfile(name, authURL, accessToken, refreshToken string) error {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}

	c.Profiles[name] = &Profile{
		AuthURL:      authURL,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.CurrentProfile = name
	return c.Save()
}

func (c *Config) GetProfile(name string) (*Profile, error) {
	if name == "" {
		name = c.CurrentProfile
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	return profile, nil
}

func (c *Config) RemoveProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}

	delete(c.Profiles, name)

	if c.CurrentProfile == name {
		c.CurrentProfile = ""
	}

	return c.Save()
}

// GetAuthURL returns the auth URL from profile or defaults
func (c *Config) GetAuthURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.AuthURL != "" {
			return p.AuthURL
		}
	}
	return c.Defaults.AuthURL
}

// GetIngestURL returns the ingest URL from profile or defaults
func (c *Config) GetIngestURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.IngestURL != "" {
			return p.IngestURL
		}
	}
	return c.Defaults.IngestURL
}

// GetQueryURL returns the query URL from profile or defaults
func (c *Config) GetQueryURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.QueryURL != "" {
			return p.QueryURL
		}
	}
	return c.Defaults.QueryURL
}

// GetRulesURL returns the rules URL from profile or defaults
func (c *Config) GetRulesURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.RulesURL != "" {
			return p.RulesURL
		}
	}
	return c.Defaults.RulesURL
}

// GetAlertingURL returns the alerting URL from profile or defaults
func (c *Config) GetAlertingURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.AlertingURL != "" {
			return p.AlertingURL
		}
	}
	return c.Defaults.AlertingURL
}
