package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CurrentProfile string              `yaml:"current_profile"`
	Profiles       map[string]*Profile `yaml:"profiles"`
	path           string
}

type Profile struct {
	AuthURL      string `yaml:"auth_url"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
}

func Default() *Config {
	return &Config{
		CurrentProfile: "default",
		Profiles:       make(map[string]*Profile),
	}
}

func Load(cfgFile string) (*Config, error) {
	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfgFile = filepath.Join(home, ".thawk", "config.yaml")
	}

	cfg := Default()
	cfg.path = cfgFile

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if c.path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(home, ".thawk", "config.yaml")
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}

func (c *Config) SaveProfile(name, authURL, accessToken, refreshToken string) error {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}

	c.Profiles[name] = &Profile{
		AuthURL:      authURL,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	c.CurrentProfile = name
	return c.Save()
}

func (c *Config) GetProfile(name string) (*Profile, error) {
	if name == "" {
		name = c.CurrentProfile
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	return profile, nil
}

func (c *Config) RemoveProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}

	delete(c.Profiles, name)

	if c.CurrentProfile == name {
		c.CurrentProfile = ""
	}

	return c.Save()
}
