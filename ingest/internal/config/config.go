package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Core       CoreConfig       `mapstructure:"core"`
	OpenSearch OpenSearchConfig `mapstructure:"opensearch"`
	Ingestion  IngestionConfig  `mapstructure:"ingestion"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type AuthConfig struct {
	URL                     string        `mapstructure:"url"`
	TokenValidationCacheTTL time.Duration `mapstructure:"token_validation_cache_ttl"`
}

type CoreConfig struct {
	URL string `mapstructure:"url"`
}

type OpenSearchConfig struct {
	URL               string        `mapstructure:"url"`
	Username          string        `mapstructure:"username"`
	Password          string        `mapstructure:"password"`
	TLSSkipVerify     bool          `mapstructure:"tls_skip_verify"`
	IndexPrefix       string        `mapstructure:"index_prefix"`
	BulkBatchSize     int           `mapstructure:"bulk_batch_size"`
	BulkFlushInterval time.Duration `mapstructure:"bulk_flush_interval"`
}

type IngestionConfig struct {
	MaxEventSize      int           `mapstructure:"max_event_size"`
	RateLimitEnabled  bool          `mapstructure:"rate_limit_enabled"`
	RateLimitRequests int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow   time.Duration `mapstructure:"rate_limit_window"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8088)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("auth.url", "http://localhost:8080")
	v.SetDefault("auth.token_validation_cache_ttl", "5m")
	v.SetDefault("core.url", "http://localhost:8090")
	v.SetDefault("opensearch.url", "https://localhost:9200")
	v.SetDefault("opensearch.username", "admin")
	v.SetDefault("opensearch.tls_skip_verify", true)
	v.SetDefault("opensearch.index_prefix", "telhawk")
	v.SetDefault("opensearch.bulk_batch_size", 1000)
	v.SetDefault("opensearch.bulk_flush_interval", "5s")
	v.SetDefault("ingestion.max_event_size", 1048576)
	v.SetDefault("ingestion.rate_limit_enabled", true)
	v.SetDefault("ingestion.rate_limit_requests", 10000)
	v.SetDefault("ingestion.rate_limit_window", "1m")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/telhawk/ingest")
	}

	// Environment variables override
	v.SetEnvPrefix("INGEST")
	v.AutomaticEnv()

	// Read config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found; use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
