package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config contains runtime configuration for the search service.
type Config struct {
	Server      ServerConfig     `yaml:"server"`
	OpenSearch  OpenSearchConfig `yaml:"opensearch"`
	Alerting    AlertingConfig   `yaml:"alerting"`
	Logging     LoggingConfig    `yaml:"logging"`
	NATS        NATSConfig       `yaml:"nats"`
	DatabaseURL string           `yaml:"database_url"`
	AuthURL     string           `yaml:"auth_url"`
}

// NATSConfig captures NATS message broker connection settings.
type NATSConfig struct {
	URL           string `yaml:"url"`
	Enabled       bool   `yaml:"enabled"`
	MaxReconnects int    `yaml:"max_reconnects"`
	ReconnectWait int    `yaml:"reconnect_wait_seconds"`
}

// ReconnectWaitDuration returns the reconnect wait as a time.Duration.
func (n NATSConfig) ReconnectWaitDuration() time.Duration {
	return time.Duration(n.ReconnectWait) * time.Second
}

// OpenSearchConfig captures OpenSearch connection settings.
type OpenSearchConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
	Index    string `yaml:"index"`
}

// ServerConfig captures HTTP server settings.
type ServerConfig struct {
	Port                int `yaml:"port"`
	ReadTimeoutSeconds  int `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int `yaml:"write_timeout_seconds"`
	IdleTimeoutSeconds  int `yaml:"idle_timeout_seconds"`
}

// AlertingConfig captures alert scheduler and notification settings.
type AlertingConfig struct {
	Enabled              bool   `yaml:"enabled"`
	CheckIntervalSeconds int    `yaml:"check_interval_seconds"`
	WebhookURL           string `yaml:"webhook_url"`
	SlackWebhookURL      string `yaml:"slack_webhook_url"`
	NotificationTimeout  int    `yaml:"notification_timeout_seconds"`
}

// LoggingConfig captures logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json or text
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

// Default returns Config with sane defaults.
func Default() Config {
	return Config{
		Server: ServerConfig{
			Port:                8082,
			ReadTimeoutSeconds:  15,
			WriteTimeoutSeconds: 15,
			IdleTimeoutSeconds:  60,
		},
		OpenSearch: OpenSearchConfig{
			URL:      "https://localhost:9200",
			Username: "admin",
			Password: "admin",
			Insecure: true,
			Index:    "ocsf-events",
		},
		Alerting: AlertingConfig{
			Enabled:              false,
			CheckIntervalSeconds: 30,
			NotificationTimeout:  10,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		NATS: NATSConfig{
			URL:           "nats://nats:4222",
			Enabled:       true,
			MaxReconnects: -1, // Infinite reconnects
			ReconnectWait: 2,  // 2 seconds
		},
		DatabaseURL: "",
		AuthURL:     "http://auth:8080",
	}
}

// Load reads configuration from the provided path and environment variables.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		if err := hydrateFromFile(path, &cfg); err != nil {
			return nil, err
		}
	}
	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func hydrateFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SEARCH_SERVER_PORT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = parsed
		}
	}
	if v := os.Getenv("SEARCH_SERVER_READ_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.ReadTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("SEARCH_SERVER_WRITE_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.WriteTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("SEARCH_SERVER_IDLE_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.IdleTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("SEARCH_OPENSEARCH_URL"); v != "" {
		cfg.OpenSearch.URL = v
	}
	if v := os.Getenv("SEARCH_OPENSEARCH_USERNAME"); v != "" {
		cfg.OpenSearch.Username = v
	}
	if v := os.Getenv("SEARCH_OPENSEARCH_PASSWORD"); v != "" {
		cfg.OpenSearch.Password = v
	}
	if v := os.Getenv("SEARCH_OPENSEARCH_INSECURE"); v != "" {
		cfg.OpenSearch.Insecure = v == "true"
	}
	if v := os.Getenv("SEARCH_OPENSEARCH_INDEX"); v != "" {
		cfg.OpenSearch.Index = v
	}
	if v := os.Getenv("SEARCH_ALERTING_ENABLED"); v != "" {
		cfg.Alerting.Enabled = v == "true"
	}
	if v := os.Getenv("SEARCH_ALERTING_CHECK_INTERVAL_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Alerting.CheckIntervalSeconds = parsed
		}
	}
	if v := os.Getenv("SEARCH_ALERTING_WEBHOOK_URL"); v != "" {
		cfg.Alerting.WebhookURL = v
	}
	if v := os.Getenv("SEARCH_ALERTING_SLACK_WEBHOOK_URL"); v != "" {
		cfg.Alerting.SlackWebhookURL = v
	}
	if v := os.Getenv("SEARCH_ALERTING_NOTIFICATION_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Alerting.NotificationTimeout = parsed
		}
	}
	if v := os.Getenv("SEARCH_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("SEARCH_AUTH_URL"); v != "" {
		cfg.AuthURL = v
	}
	// NATS config overrides
	if v := os.Getenv("SEARCH_NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("SEARCH_NATS_ENABLED"); v != "" {
		cfg.NATS.Enabled = v == "true"
	}
	if v := os.Getenv("SEARCH_NATS_MAX_RECONNECTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.NATS.MaxReconnects = parsed
		}
	}
	if v := os.Getenv("SEARCH_NATS_RECONNECT_WAIT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.NATS.ReconnectWait = parsed
		}
	}
}
