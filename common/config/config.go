// Package config provides centralized configuration management for all TelHawk services.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	globalConfig *Config
	once         sync.Once
)

// Config is the master configuration struct containing all service configs and shared infrastructure.
type Config struct {
	// Service-specific configurations
	Authenticate AuthenticateConfig `mapstructure:"authenticate"`
	Ingest       IngestConfig       `mapstructure:"ingest"`
	Search       SearchConfig       `mapstructure:"search"`
	Respond      RespondConfig      `mapstructure:"respond"`
	Web          WebConfig          `mapstructure:"web"`

	// Shared infrastructure configurations
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	OpenSearch OpenSearchConfig `mapstructure:"opensearch"`
	NATS       NATSConfig       `mapstructure:"nats"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// AuthenticateConfig holds authenticate service configuration
type AuthenticateConfig struct {
	Server   ServerConfig    `mapstructure:"server"`
	Auth     AuthConfig      `mapstructure:"auth"`
	Ingest   IngestFwdConfig `mapstructure:"ingest"`
	Database DatabaseConfig  `mapstructure:"database"`
}

// AuthConfig holds JWT and token configuration
type AuthConfig struct {
	JWTSecret        string        `mapstructure:"jwt_secret"`
	JWTRefreshSecret string        `mapstructure:"jwt_refresh_secret"`
	AuditSecret      string        `mapstructure:"audit_secret"`
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl"`
}

// IngestFwdConfig holds ingest forwarding configuration
type IngestFwdConfig struct {
	URL      string `mapstructure:"url"`
	HECToken string `mapstructure:"hec_token"`
	Enabled  bool   `mapstructure:"enabled"`
}

// IngestConfig holds ingest service configuration
type IngestConfig struct {
	Server       ServerConfig          `mapstructure:"server"`
	Authenticate AuthenticateURLConfig `mapstructure:"authenticate"`
	Ingestion    IngestionConfig       `mapstructure:"ingestion"`
	Ack          AckConfig             `mapstructure:"ack"`
	DLQ          DLQConfig             `mapstructure:"dlq"`
}

// AuthenticateURLConfig holds authenticate service URL and caching config
type AuthenticateURLConfig struct {
	URL                     string        `mapstructure:"url"`
	TokenValidationCacheTTL time.Duration `mapstructure:"token_validation_cache_ttl"`
}

// IngestionConfig holds ingestion pipeline configuration
type IngestionConfig struct {
	MaxEventSize      int           `mapstructure:"max_event_size"`
	RateLimitEnabled  bool          `mapstructure:"rate_limit_enabled"`
	RateLimitRequests int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow   time.Duration `mapstructure:"rate_limit_window"`
}

// AckConfig holds acknowledgment configuration
type AckConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	TTL     time.Duration `mapstructure:"ttl"`
}

// DLQConfig holds dead letter queue configuration
type DLQConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Backend  string `mapstructure:"backend"`   // "jetstream" (default) or "file"
	BasePath string `mapstructure:"base_path"` // Only used for file backend
	NatsURL  string `mapstructure:"nats_url"`  // Only used for jetstream backend
}

// SearchConfig holds search service configuration
type SearchConfig struct {
	Server      ServerConfig   `mapstructure:"server"`
	Alerting    AlertingConfig `mapstructure:"alerting"`
	DatabaseURL string         `mapstructure:"database_url"`
	AuthURL     string         `mapstructure:"auth_url"`
}

// AlertingConfig holds alert scheduler and notification settings
type AlertingConfig struct {
	Enabled              bool   `mapstructure:"enabled"`
	CheckIntervalSeconds int    `mapstructure:"check_interval_seconds"`
	WebhookURL           string `mapstructure:"webhook_url"`
	SlackWebhookURL      string `mapstructure:"slack_webhook_url"`
	NotificationTimeout  int    `mapstructure:"notification_timeout_seconds"`
}

// RespondConfig holds respond service configuration
type RespondConfig struct {
	Server     ServerConfig     `mapstructure:"server"`
	Auth       AuthURLConfig    `mapstructure:"auth"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Validation ValidationConfig `mapstructure:"validation"`
	Database   DatabaseConfig   `mapstructure:"database"`
}

// AuthURLConfig holds authentication service URL
type AuthURLConfig struct {
	URL string `mapstructure:"url"`
}

// StorageConfig holds OpenSearch configuration for respond service
type StorageConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Insecure bool   `mapstructure:"insecure"`
}

// ValidationConfig holds rule validation settings
type ValidationConfig struct {
	MaxTimeWindow       string   `mapstructure:"max_time_window"`
	MaxThreshold        int      `mapstructure:"max_threshold"`
	AllowedAggregations []string `mapstructure:"allowed_aggregations"`
}

// WebConfig holds web service configuration
type WebConfig struct {
	Server    ServerConfig `mapstructure:"server"`
	StaticDir string       `mapstructure:"static_dir"`
	DevMode   bool         `mapstructure:"dev_mode"`
	Services  struct {
		AuthenticateURL string `mapstructure:"authenticate_url"`
		SearchURL       string `mapstructure:"search_url"`
		RespondURL      string `mapstructure:"respond_url"`
		IngestURL       string `mapstructure:"ingest_url"`
	} `mapstructure:"services"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port                int           `mapstructure:"port"`
	ReadTimeout         time.Duration `mapstructure:"read_timeout"`
	WriteTimeout        time.Duration `mapstructure:"write_timeout"`
	IdleTimeout         time.Duration `mapstructure:"idle_timeout"`
	ReadTimeoutSeconds  int           `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int           `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds  int           `mapstructure:"idle_timeout_seconds"`
}

// ReadTimeoutDuration returns the read timeout as a duration (handles both formats)
func (s ServerConfig) ReadTimeoutDuration() time.Duration {
	if s.ReadTimeout != 0 {
		return s.ReadTimeout
	}
	return time.Duration(s.ReadTimeoutSeconds) * time.Second
}

// WriteTimeoutDuration returns the write timeout as a duration (handles both formats)
func (s ServerConfig) WriteTimeoutDuration() time.Duration {
	if s.WriteTimeout != 0 {
		return s.WriteTimeout
	}
	return time.Duration(s.WriteTimeoutSeconds) * time.Second
}

// IdleTimeoutDuration returns the idle timeout as a duration (handles both formats)
func (s ServerConfig) IdleTimeoutDuration() time.Duration {
	if s.IdleTimeout != 0 {
		return s.IdleTimeout
	}
	return time.Duration(s.IdleTimeoutSeconds) * time.Second
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string         `mapstructure:"type"`
	Postgres PostgresConfig `mapstructure:"postgres"`
}

// PostgresConfig holds PostgreSQL connection settings
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"sslmode"`
}

// OpenSearchConfig holds OpenSearch connection settings
type OpenSearchConfig struct {
	URL             string        `mapstructure:"url"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	TLSSkipVerify   bool          `mapstructure:"tls_skip_verify"`
	Insecure        bool          `mapstructure:"insecure"`
	IndexPrefix     string        `mapstructure:"index_prefix"`
	Index           string        `mapstructure:"index"`
	ShardCount      int           `mapstructure:"shard_count"`
	ReplicaCount    int           `mapstructure:"replica_count"`
	RefreshInterval string        `mapstructure:"refresh_interval"`
	RetentionDays   int           `mapstructure:"retention_days"`
	RolloverSizeGB  int           `mapstructure:"rollover_size_gb"`
	RolloverAge     time.Duration `mapstructure:"rollover_age"`
}

// NATSConfig holds NATS message broker configuration
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	Enabled       bool          `mapstructure:"enabled"`
	MaxReconnects int           `mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
}

// ReconnectWaitDuration returns the reconnect wait as a time.Duration
func (n NATSConfig) ReconnectWaitDuration() time.Duration {
	return n.ReconnectWait
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL        string `mapstructure:"url"`
	Enabled    bool   `mapstructure:"enabled"`
	MaxRetries int    `mapstructure:"max_retries"`
	PoolSize   int    `mapstructure:"pool_size"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CLI-specific configuration structures

// CLIConfig holds CLI tool configuration (profiles, tokens, etc.)
type CLIConfig struct {
	CurrentProfile string                 `yaml:"current_profile" mapstructure:"current_profile"`
	Profiles       map[string]*CLIProfile `yaml:"profiles" mapstructure:"profiles"`
	Defaults       *CLIDefaults           `yaml:"defaults" mapstructure:"defaults"`
	path           string
}

// CLIProfile holds authentication and endpoint configuration for a CLI profile
type CLIProfile struct {
	AuthURL      string `yaml:"auth_url" mapstructure:"auth_url"`
	IngestURL    string `yaml:"ingest_url" mapstructure:"ingest_url"`
	QueryURL     string `yaml:"query_url" mapstructure:"query_url"`
	RulesURL     string `yaml:"rules_url" mapstructure:"rules_url"`
	AlertingURL  string `yaml:"alerting_url" mapstructure:"alerting_url"`
	AccessToken  string `yaml:"access_token" mapstructure:"access_token"`
	RefreshToken string `yaml:"refresh_token" mapstructure:"refresh_token"`
	HECToken     string `yaml:"hec_token" mapstructure:"hec_token"` // Optional default HEC token for ingestion
}

// CLIDefaults holds default endpoint URLs for CLI operations
type CLIDefaults struct {
	AuthURL     string `yaml:"auth_url" mapstructure:"auth_url"`
	IngestURL   string `yaml:"ingest_url" mapstructure:"ingest_url"`
	QueryURL    string `yaml:"query_url" mapstructure:"query_url"`
	RulesURL    string `yaml:"rules_url" mapstructure:"rules_url"`
	AlertingURL string `yaml:"alerting_url" mapstructure:"alerting_url"`
}

// MustLoad loads the configuration and panics on error.
// This initializes the global singleton.
func MustLoad(serviceName string) {
	once.Do(func() {
		cfg, err := Load(serviceName)
		if err != nil {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
		globalConfig = cfg
	})
}

// GetConfig returns the global configuration singleton.
// Panics if MustLoad has not been called first.
func GetConfig() *Config {
	if globalConfig == nil {
		panic("config not initialized - call MustLoad first")
	}
	return globalConfig
}

// Load reads configuration from $TELHAWK_CONFIG_DIR/config.yaml and environment variables.
// The serviceName parameter is currently passed but not used (all services load one config.yaml).
func Load(serviceName string) (*Config, error) {
	v := viper.New()

	// Set all defaults
	setDefaults(v)

	// Determine config directory
	configDir := os.Getenv("TELHAWK_CONFIG_DIR")
	if configDir == "" {
		configDir = "/etc/telhawk"
	}

	// Read config file
	configPath := fmt.Sprintf("%s/config.yaml", configDir)
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Environment variables override with NO prefix (empty string)
	v.SetEnvPrefix("")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file - don't fail if file doesn't exist
	if err := v.ReadInConfig(); err != nil {
		// Only return error if config file was explicitly expected but had read errors
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Some other error occurred
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found - continue with defaults and env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets all default configuration values
func setDefaults(v *viper.Viper) {
	// Authenticate service defaults
	v.SetDefault("authenticate.auth.jwt_secret", "change-this-in-production")
	v.SetDefault("authenticate.auth.jwt_refresh_secret", "change-this-in-production")
	v.SetDefault("authenticate.auth.audit_secret", "change-this-in-production")
	v.SetDefault("authenticate.auth.access_token_ttl", "15m")
	v.SetDefault("authenticate.auth.refresh_token_ttl", "168h")
	v.SetDefault("authenticate.ingest.enabled", false)
	v.SetDefault("authenticate.ingest.url", "http://ingest:8088")
	v.SetDefault("authenticate.database.type", "postgres")
	v.SetDefault("authenticate.database.postgres.user", "telhawk_auth_user")
	v.SetDefault("authenticate.database.postgres.password", "")
	v.SetDefault("authenticate.database.postgres.database", "telhawk_auth")

	// Ingest service defaults
	v.SetDefault("ingest.authenticate.url", "http://authenticate:8080")
	v.SetDefault("ingest.authenticate.token_validation_cache_ttl", "5m")
	v.SetDefault("ingest.ingestion.max_event_size", 1048576)
	v.SetDefault("ingest.ingestion.rate_limit_enabled", true)
	v.SetDefault("ingest.ingestion.rate_limit_requests", 10000)
	v.SetDefault("ingest.ingestion.rate_limit_window", "1m")
	v.SetDefault("ingest.ack.enabled", true)
	v.SetDefault("ingest.ack.ttl", "10m")
	v.SetDefault("ingest.dlq.enabled", true)
	v.SetDefault("ingest.dlq.backend", "jetstream")
	v.SetDefault("ingest.dlq.base_path", "/var/lib/telhawk/dlq")
	v.SetDefault("ingest.dlq.nats_url", "nats://nats:4222")

	// Search service defaults
	v.SetDefault("search.alerting.enabled", false)
	v.SetDefault("search.alerting.check_interval_seconds", 30)
	v.SetDefault("search.alerting.webhook_url", "")
	v.SetDefault("search.alerting.slack_webhook_url", "")
	v.SetDefault("search.alerting.notification_timeout_seconds", 10)
	v.SetDefault("search.database_url", "")
	v.SetDefault("search.auth_url", "http://authenticate:8080")

	// Respond service defaults
	v.SetDefault("respond.auth.url", "http://authenticate:8080")
	v.SetDefault("respond.storage.url", "https://localhost:9200")
	v.SetDefault("respond.storage.username", "admin")
	v.SetDefault("respond.storage.password", "")
	v.SetDefault("respond.storage.insecure", true)
	v.SetDefault("respond.validation.max_time_window", "24h")
	v.SetDefault("respond.validation.max_threshold", 100000)
	v.SetDefault("respond.validation.allowed_aggregations", []string{"count", "sum", "avg", "max", "min"})
	v.SetDefault("respond.database.type", "postgres")
	v.SetDefault("respond.database.postgres.user", "telhawk_respond_user")
	v.SetDefault("respond.database.postgres.password", "")
	v.SetDefault("respond.database.postgres.database", "telhawk_respond")

	// Server defaults (port varies by service, so no default here)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")

	// Database defaults
	v.SetDefault("database.type", "postgres")
	v.SetDefault("database.postgres.host", "localhost")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.database", "telhawk")
	v.SetDefault("database.postgres.user", "telhawk")
	v.SetDefault("database.postgres.password", "")
	v.SetDefault("database.postgres.sslmode", "disable")

	// OpenSearch defaults
	v.SetDefault("opensearch.url", "https://localhost:9200")
	v.SetDefault("opensearch.username", "admin")
	v.SetDefault("opensearch.password", "admin")
	v.SetDefault("opensearch.tls_skip_verify", true)
	v.SetDefault("opensearch.insecure", true)
	v.SetDefault("opensearch.index_prefix", "telhawk-events")
	v.SetDefault("opensearch.index", "telhawk-events")
	v.SetDefault("opensearch.shard_count", 1)
	v.SetDefault("opensearch.replica_count", 0)
	v.SetDefault("opensearch.refresh_interval", "5s")
	v.SetDefault("opensearch.retention_days", 30)
	v.SetDefault("opensearch.rollover_size_gb", 50)
	v.SetDefault("opensearch.rollover_age", "24h")

	// NATS defaults
	v.SetDefault("nats.url", "nats://nats:4222")
	v.SetDefault("nats.enabled", true)
	v.SetDefault("nats.max_reconnects", -1)
	v.SetDefault("nats.reconnect_wait", "2s")

	// Redis defaults
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

// CLI-specific helper methods

// DefaultCLI returns a CLIConfig with default values
func DefaultCLI() *CLIConfig {
	return &CLIConfig{
		CurrentProfile: "default",
		Profiles:       make(map[string]*CLIProfile),
		Defaults: &CLIDefaults{
			// All CLI operations go through the web backend
			AuthURL:     "http://localhost:3000",
			IngestURL:   "http://localhost:8088", // Ingest is exposed directly for HEC
			QueryURL:    "http://localhost:3000",
			RulesURL:    "http://localhost:3000",
			AlertingURL: "http://localhost:3000",
		},
	}
}

// Save writes the CLI config to disk
func (c *CLIConfig) Save() error {
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

// SaveProfile saves authentication tokens to a profile
func (c *CLIConfig) SaveProfile(name, authURL, accessToken, refreshToken string) error {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*CLIProfile)
	}

	// Preserve existing HEC token if profile already exists
	existingHECToken := ""
	if existing, ok := c.Profiles[name]; ok {
		existingHECToken = existing.HECToken
	}

	c.Profiles[name] = &CLIProfile{
		AuthURL:      authURL,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		HECToken:     existingHECToken,
	}

	c.CurrentProfile = name
	return c.Save()
}

// SaveHECToken saves a default HEC token to the specified profile
func (c *CLIConfig) SaveHECToken(name, hecToken string) error {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*CLIProfile)
	}

	// Get or create profile
	profile, ok := c.Profiles[name]
	if !ok {
		profile = &CLIProfile{}
		c.Profiles[name] = profile
	}

	profile.HECToken = hecToken
	return c.Save()
}

// GetProfile retrieves a profile by name (or current profile if name is empty)
func (c *CLIConfig) GetProfile(name string) (*CLIProfile, error) {
	if name == "" {
		name = c.CurrentProfile
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	return profile, nil
}

// RemoveProfile removes a profile from the configuration
func (c *CLIConfig) RemoveProfile(name string) error {
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
func (c *CLIConfig) GetAuthURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.AuthURL != "" {
			return p.AuthURL
		}
	}
	return c.Defaults.AuthURL
}

// GetIngestURL returns the ingest URL from profile or defaults
func (c *CLIConfig) GetIngestURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.IngestURL != "" {
			return p.IngestURL
		}
	}
	return c.Defaults.IngestURL
}

// GetQueryURL returns the query URL from profile or defaults
func (c *CLIConfig) GetQueryURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.QueryURL != "" {
			return p.QueryURL
		}
	}
	return c.Defaults.QueryURL
}

// GetRulesURL returns the rules URL from profile or defaults
func (c *CLIConfig) GetRulesURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.RulesURL != "" {
			return p.RulesURL
		}
	}
	return c.Defaults.RulesURL
}

// GetAlertingURL returns the alerting URL from profile or defaults
func (c *CLIConfig) GetAlertingURL(profile string) string {
	if profile != "" {
		if p, err := c.GetProfile(profile); err == nil && p.AlertingURL != "" {
			return p.AlertingURL
		}
	}
	return c.Defaults.AlertingURL
}
