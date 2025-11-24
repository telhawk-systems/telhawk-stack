package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Authenticate AuthenticateConfig `mapstructure:"authenticate"`
	OpenSearch   OpenSearchConfig   `mapstructure:"opensearch"`
	Ingestion    IngestionConfig    `mapstructure:"ingestion"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Ack          AckConfig          `mapstructure:"ack"`
	DLQ          DLQConfig          `mapstructure:"dlq"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type AuthenticateConfig struct {
	URL                     string        `mapstructure:"url"`
	TokenValidationCacheTTL time.Duration `mapstructure:"token_validation_cache_ttl"`
}

type OpenSearchConfig struct {
	URL             string        `mapstructure:"url"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	TLSSkipVerify   bool          `mapstructure:"tls_skip_verify"`
	IndexPrefix     string        `mapstructure:"index_prefix"`
	ShardCount      int           `mapstructure:"shard_count"`
	ReplicaCount    int           `mapstructure:"replica_count"`
	RefreshInterval string        `mapstructure:"refresh_interval"`
	RetentionDays   int           `mapstructure:"retention_days"`
	RolloverSizeGB  int           `mapstructure:"rollover_size_gb"`
	RolloverAge     time.Duration `mapstructure:"rollover_age"`
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

type RedisConfig struct {
	URL     string `mapstructure:"url"`
	Enabled bool   `mapstructure:"enabled"`
}

type AckConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	TTL     time.Duration `mapstructure:"ttl"`
}

type DLQConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Backend  string `mapstructure:"backend"`   // "jetstream" (default) or "file"
	BasePath string `mapstructure:"base_path"` // Only used for file backend
	NatsURL  string `mapstructure:"nats_url"`  // Only used for jetstream backend
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8088)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "120s")
	v.SetDefault("authenticate.url", "http://localhost:8080")
	v.SetDefault("authenticate.token_validation_cache_ttl", "5m")
	v.SetDefault("opensearch.url", "https://localhost:9200")
	v.SetDefault("opensearch.username", "admin")
	v.SetDefault("opensearch.tls_skip_verify", true)
	v.SetDefault("opensearch.index_prefix", "telhawk-events")
	v.SetDefault("opensearch.shard_count", 1)
	v.SetDefault("opensearch.replica_count", 0)
	v.SetDefault("opensearch.refresh_interval", "5s")
	v.SetDefault("opensearch.retention_days", 30)
	v.SetDefault("opensearch.rollover_size_gb", 50)
	v.SetDefault("opensearch.rollover_age", "24h")
	v.SetDefault("ingestion.max_event_size", 1048576)
	v.SetDefault("ingestion.rate_limit_enabled", true)
	v.SetDefault("ingestion.rate_limit_requests", 10000)
	v.SetDefault("ingestion.rate_limit_window", "1m")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.enabled", false)
	v.SetDefault("ack.enabled", true)
	v.SetDefault("ack.ttl", "10m")
	v.SetDefault("dlq.enabled", true)
	v.SetDefault("dlq.backend", "jetstream")
	v.SetDefault("dlq.base_path", "/var/lib/telhawk/dlq")
	v.SetDefault("dlq.nats_url", "nats://localhost:4222")

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
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// If a specific config path was provided and not found, return error
			if configPath != "" {
				return nil, fmt.Errorf("failed to read config: %w", err)
			}
			// Otherwise, config file not found; use defaults (this is OK)
		} else {
			// Some other error occurred
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
