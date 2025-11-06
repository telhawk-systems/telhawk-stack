package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config contains runtime configuration for the query service.
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	OpenSearch  OpenSearchConfig  `yaml:"opensearch"`
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
	if v := os.Getenv("QUERY_SERVER_PORT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = parsed
		}
	}
	if v := os.Getenv("QUERY_SERVER_READ_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.ReadTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("QUERY_SERVER_WRITE_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.WriteTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("QUERY_SERVER_IDLE_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.IdleTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("QUERY_OPENSEARCH_URL"); v != "" {
		cfg.OpenSearch.URL = v
	}
	if v := os.Getenv("QUERY_OPENSEARCH_USERNAME"); v != "" {
		cfg.OpenSearch.Username = v
	}
	if v := os.Getenv("QUERY_OPENSEARCH_PASSWORD"); v != "" {
		cfg.OpenSearch.Password = v
	}
	if v := os.Getenv("QUERY_OPENSEARCH_INSECURE"); v != "" {
		cfg.OpenSearch.Insecure = v == "true"
	}
	if v := os.Getenv("QUERY_OPENSEARCH_INDEX"); v != "" {
		cfg.OpenSearch.Index = v
	}
}
