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
	Storage  StorageConfig  `json:"storage"`
	DLQ      DLQConfig      `json:"dlq"`
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

// StorageConfig holds storage service connection settings.
type StorageConfig struct {
	URL string `json:"url"`
}

// DLQConfig controls dead-letter queue settings.
type DLQConfig struct {
	Enabled  bool   `json:"enabled"`
	BasePath string `json:"base_path"`
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
		Storage: StorageConfig{
			URL: "http://storage:8083",
		},
		DLQ: DLQConfig{
			Enabled:  true,
			BasePath: "/var/lib/telhawk/dlq",
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
	if v := os.Getenv("CORE_STORAGE_URL"); v != "" {
		cfg.Storage.URL = v
	}
	if v := os.Getenv("CORE_DLQ_ENABLED"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.DLQ.Enabled = parsed
		}
	}
	if v := os.Getenv("CORE_DLQ_BASE_PATH"); v != "" {
		cfg.DLQ.BasePath = v
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
