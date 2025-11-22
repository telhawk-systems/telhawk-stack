// Package config provides configuration loading for the respond service.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the respond service
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Validation ValidationConfig `mapstructure:"validation"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	NATS       NATSConfig       `mapstructure:"nats"`
}

// NATSConfig holds NATS message broker configuration
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	Enabled       bool          `mapstructure:"enabled"`
	MaxReconnects int           `mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
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

// ValidationConfig holds rule validation settings
type ValidationConfig struct {
	MaxTimeWindow       string   `mapstructure:"max_time_window"`
	MaxThreshold        int      `mapstructure:"max_threshold"`
	AllowedAggregations []string `mapstructure:"allowed_aggregations"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8086)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")

	v.SetDefault("database.postgres.host", "localhost")
	v.SetDefault("database.postgres.port", 5432)
	v.SetDefault("database.postgres.user", "telhawk")
	v.SetDefault("database.postgres.password", "")
	v.SetDefault("database.postgres.database", "telhawk_respond")
	v.SetDefault("database.postgres.sslmode", "disable")

	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.enabled", true)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)

	v.SetDefault("storage.url", "https://localhost:9200")
	v.SetDefault("storage.username", "admin")
	v.SetDefault("storage.password", "")
	v.SetDefault("storage.insecure", true)

	v.SetDefault("validation.max_time_window", "24h")
	v.SetDefault("validation.max_threshold", 100000)
	v.SetDefault("validation.allowed_aggregations", []string{"count", "sum", "avg", "max", "min"})

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	v.SetDefault("nats.url", "nats://nats:4222")
	v.SetDefault("nats.enabled", true)
	v.SetDefault("nats.max_reconnects", -1)
	v.SetDefault("nats.reconnect_wait", "2s")

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/telhawk/respond")
	}

	// Environment variables override (RESPOND_SERVER_PORT, etc.)
	v.SetEnvPrefix("RESPOND")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config - ignore file not found for defaults
	if err := v.ReadInConfig(); err != nil {
		// Only fail if a specific config path was given
		if configPath != "" {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Otherwise use defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
