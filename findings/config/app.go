package config

// AppConfig holds application-level settings.
type AppConfig struct {
	Name  string
	Env   string
	Debug bool
	Port  string
	URL   string
}

func app() AppConfig {
	return AppConfig{
		Name:  Env("APP_NAME", "findings"),
		Env:   Env("APP_ENV", "local"),
		Debug: Env("APP_DEBUG", "true") == "true",
		Port:  Env("APP_PORT", "8080"),
		URL:   Env("APP_URL", "http://localhost:8080"),
	}
}
