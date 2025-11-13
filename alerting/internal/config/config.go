package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the alerting service
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Rules    RulesConfig    `mapstructure:"rules"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
}

// PostgresConfig holds PostgreSQL connection settings
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig holds Redis configuration for state management
type RedisConfig struct {
	URL        string `mapstructure:"url"`
	Enabled    bool   `mapstructure:"enabled"`
	MaxRetries int    `mapstructure:"max_retries"`
	PoolSize   int    `mapstructure:"pool_size"`
}

// StorageConfig holds OpenSearch configuration
type StorageConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Insecure bool   `mapstructure:"insecure"`
}

// RulesConfig holds Rules service configuration
type RulesConfig struct {
	URL string `mapstructure:"url"`
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8085)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")

	v.SetDefault("database.postgres.host", "localhost")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.user", "telhawk")
	v.SetDefault("database.postgres.password", "")
	v.SetDefault("database.postgres.database", "telhawk_alerting")
	v.SetDefault("database.postgres.sslmode", "disable")

	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.enabled", true)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)

	v.SetDefault("storage.url", "https://localhost:9200")
	v.SetDefault("storage.username", "admin")
	v.SetDefault("storage.password", "")
	v.SetDefault("storage.insecure", true)

	v.SetDefault("rules.url", "http://localhost:8084")

	// Read from config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables override file config
	v.SetEnvPrefix("ALERTING")
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
