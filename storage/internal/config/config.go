package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server          ServerConfig          `yaml:"server"`
	OpenSearch      OpenSearchConfig      `yaml:"opensearch"`
	IndexManagement IndexManagementConfig `yaml:"index_management"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type OpenSearchConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
}

type IndexManagementConfig struct {
	IndexPrefix    string        `yaml:"index_prefix"`
	ShardCount     int           `yaml:"shard_count"`
	ReplicaCount   int           `yaml:"replica_count"`
	RefreshInterval string       `yaml:"refresh_interval"`
	RetentionDays  int           `yaml:"retention_days"`
	RolloverSizeGB int           `yaml:"rollover_size_gb"`
	RolloverAge    time.Duration `yaml:"rollover_age"`
}

func Load(configPath string) (*Config, error) {
	cfg := defaultConfig()

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	if url := os.Getenv("OPENSEARCH_URL"); url != "" {
		cfg.OpenSearch.URL = url
	}
	if username := os.Getenv("OPENSEARCH_USERNAME"); username != "" {
		cfg.OpenSearch.Username = username
	}
	if password := os.Getenv("OPENSEARCH_PASSWORD"); password != "" {
		cfg.OpenSearch.Password = password
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8083,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		OpenSearch: OpenSearchConfig{
			URL:      "http://opensearch:9200",
			Username: "admin",
			Password: "admin",
			Insecure: true,
		},
		IndexManagement: IndexManagementConfig{
			IndexPrefix:     "telhawk-events",
			ShardCount:      1,
			ReplicaCount:    0,
			RefreshInterval: "5s",
			RetentionDays:   30,
			RolloverSizeGB:  50,
			RolloverAge:     24 * time.Hour,
		},
	}
}
