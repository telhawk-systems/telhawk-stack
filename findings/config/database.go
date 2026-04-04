package config

// DatabaseConfig holds named database connections with a default.
type DatabaseConfig struct {
	Default     string
	Connections map[string]ConnectionConfig
}

func database() DatabaseConfig {
	return DatabaseConfig{
		Default: Env("DB_CONNECTION", "pgsql"),
		Connections: map[string]ConnectionConfig{
			"pgsql": {
				Driver:   "pgsql",
				Host:     Env("DB_HOST", "127.0.0.1"),
				Port:     Env("DB_PORT", "5432"),
				Name:     Env("DB_DATABASE", "telhawk_findings"),
				User:     Env("DB_USERNAME", "postgres"),
				Password: Env("DB_PASSWORD", ""),
			},
			"sqlite": {
				Driver: "sqlite",
				Name:   Env("DB_DATABASE", "database.sqlite"),
			},
		},
	}
}
