package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config contains runtime configuration for the search service.
type Config struct {
	Server      ServerConfig     `yaml:"server" mapstructure:"server"`
	OpenSearch  OpenSearchConfig `yaml:"opensearch" mapstructure:"opensearch"`
	Alerting    AlertingConfig   `yaml:"alerting" mapstructure:"alerting"`
	Logging     LoggingConfig    `yaml:"logging" mapstructure:"logging"`
	NATS        NATSConfig       `yaml:"nats" mapstructure:"nats"`
	DatabaseURL string           `yaml:"database_url" mapstructure:"database_url"`
	AuthURL     string           `yaml:"auth_url" mapstructure:"auth_url"`
}

// NATSConfig captures NATS message broker connection settings.
type NATSConfig struct {
	URL           string `yaml:"url" mapstructure:"url"`
	Enabled       bool   `yaml:"enabled" mapstructure:"enabled"`
	MaxReconnects int    `yaml:"max_reconnects" mapstructure:"max_reconnects"`
	ReconnectWait int    `yaml:"reconnect_wait_seconds" mapstructure:"reconnect_wait_seconds"`
}

// ReconnectWaitDuration returns the reconnect wait as a time.Duration.
func (n NATSConfig) ReconnectWaitDuration() time.Duration {
	return time.Duration(n.ReconnectWait) * time.Second
}

// OpenSearchConfig captures OpenSearch connection settings.
type OpenSearchConfig struct {
	URL      string `yaml:"url" mapstructure:"url"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Insecure bool   `yaml:"insecure" mapstructure:"insecure"`
	Index    string `yaml:"index" mapstructure:"index"`
}

// ServerConfig captures HTTP server settings.
type ServerConfig struct {
	Port                int `yaml:"port" mapstructure:"port"`
	ReadTimeoutSeconds  int `yaml:"read_timeout_seconds" mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int `yaml:"write_timeout_seconds" mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds  int `yaml:"idle_timeout_seconds" mapstructure:"idle_timeout_seconds"`
}

// AlertingConfig captures alert scheduler and notification settings.
type AlertingConfig struct {
	Enabled              bool   `yaml:"enabled" mapstructure:"enabled"`
	CheckIntervalSeconds int    `yaml:"check_interval_seconds" mapstructure:"check_interval_seconds"`
	WebhookURL           string `yaml:"webhook_url" mapstructure:"webhook_url"`
	SlackWebhookURL      string `yaml:"slack_webhook_url" mapstructure:"slack_webhook_url"`
	NotificationTimeout  int    `yaml:"notification_timeout_seconds" mapstructure:"notification_timeout_seconds"`
}

// LoggingConfig captures logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`   // debug, info, warn, error
	Format string `yaml:"format" mapstructure:"format"` // json or text
}

// ReadTimeout returns the configured read timeout as a duration.
func (s ServerConfig) ReadTimeout() time.Duration {
	return time.Duration(s.ReadTimeoutSeconds) * time.Second
}

// WriteTimeout returns the configured write timeout as a duration.
func (s ServerConfig) WriteTimeout() time.Duration {
	return time.Duration(s.WriteTimeoutSeconds) * time.Second
}

// IdleTimeout returns the configured idle timeout as a duration.
func (s ServerConfig) IdleTimeout() time.Duration {
	return time.Duration(s.IdleTimeoutSeconds) * time.Second
}

// Load reads configuration from the provided path and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set all defaults
	v.SetDefault("server.port", 8082)
	v.SetDefault("server.read_timeout_seconds", 15)
	v.SetDefault("server.write_timeout_seconds", 15)
	v.SetDefault("server.idle_timeout_seconds", 60)

	v.SetDefault("opensearch.url", "https://localhost:9200")
	v.SetDefault("opensearch.username", "admin")
	v.SetDefault("opensearch.password", "admin")
	v.SetDefault("opensearch.insecure", true)
	v.SetDefault("opensearch.index", "ocsf-events")

	v.SetDefault("alerting.enabled", false)
	v.SetDefault("alerting.check_interval_seconds", 30)
	v.SetDefault("alerting.webhook_url", "")
	v.SetDefault("alerting.slack_webhook_url", "")
	v.SetDefault("alerting.notification_timeout_seconds", 10)

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	v.SetDefault("nats.url", "nats://nats:4222")
	v.SetDefault("nats.enabled", true)
	v.SetDefault("nats.max_reconnects", -1) // Infinite reconnects
	v.SetDefault("nats.reconnect_wait_seconds", 2)

	v.SetDefault("database_url", "")
	v.SetDefault("auth_url", "http://auth:8080")

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/telhawk/search")
	}

	// Environment variables override
	v.SetEnvPrefix("SEARCH")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// Config file not found; use defaults
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
