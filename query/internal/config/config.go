package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config contains runtime configuration for the query service.
type Config struct {
	Server ServerConfig `json:"server"`
}

// ServerConfig captures HTTP server settings.
type ServerConfig struct {
	Port                int `json:"port"`
	ReadTimeoutSeconds  int `json:"read_timeout_seconds"`
	WriteTimeoutSeconds int `json:"write_timeout_seconds"`
	IdleTimeoutSeconds  int `json:"idle_timeout_seconds"`
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
	if err := json.Unmarshal(data, cfg); err != nil {
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
}
