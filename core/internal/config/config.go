package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config captures runtime settings for the core normalization service.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Pipeline PipelineConfig `json:"pipeline"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port                int `json:"port"`
	ReadTimeoutSeconds  int `json:"read_timeout_seconds"`
	WriteTimeoutSeconds int `json:"write_timeout_seconds"`
	IdleTimeoutSeconds  int `json:"idle_timeout_seconds"`
}

// PipelineConfig controls validator/normalizer behaviour.
type PipelineConfig struct {
	MaxWorkers int `json:"max_workers"`
}

// Default returns Config populated with sane defaults.
func Default() Config {
	return Config{
		Server: ServerConfig{
			Port:                8090,
			ReadTimeoutSeconds:  15,
			WriteTimeoutSeconds: 15,
			IdleTimeoutSeconds:  60,
		},
		Pipeline: PipelineConfig{
			MaxWorkers: 4,
		},
	}
}

// Load reads configuration from disk and environment overrides.
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
	if v := os.Getenv("CORE_SERVER_PORT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = parsed
		}
	}
	if v := os.Getenv("CORE_SERVER_READ_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.ReadTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("CORE_SERVER_WRITE_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.WriteTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("CORE_SERVER_IDLE_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Server.IdleTimeoutSeconds = parsed
		}
	}
	if v := os.Getenv("CORE_PIPELINE_MAX_WORKERS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Pipeline.MaxWorkers = parsed
		}
	}
}

// ReadTimeout returns read timeout duration.
func (s ServerConfig) ReadTimeout() time.Duration {
	return time.Duration(s.ReadTimeoutSeconds) * time.Second
}

// WriteTimeout returns write timeout duration.
func (s ServerConfig) WriteTimeout() time.Duration {
	return time.Duration(s.WriteTimeoutSeconds) * time.Second
}

// IdleTimeout returns idle timeout duration.
func (s ServerConfig) IdleTimeout() time.Duration {
	return time.Duration(s.IdleTimeoutSeconds) * time.Second
}
