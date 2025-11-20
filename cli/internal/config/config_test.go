package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	assert.Equal(t, "default", cfg.CurrentProfile)
	assert.NotNil(t, cfg.Profiles)
	assert.Empty(t, cfg.Profiles)
	assert.NotNil(t, cfg.Defaults)
	assert.Equal(t, "http://localhost:8080", cfg.Defaults.AuthURL)
	assert.Equal(t, "http://localhost:8088", cfg.Defaults.IngestURL)
	assert.Equal(t, "http://localhost:8082", cfg.Defaults.QueryURL)
	assert.Equal(t, "http://localhost:8084", cfg.Defaults.RulesURL)
	assert.Equal(t, "http://localhost:8085", cfg.Defaults.AlertingURL)
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Load with non-existent path (should use defaults)
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)

	assert.Equal(t, "default", cfg.CurrentProfile)
	assert.NotNil(t, cfg.Defaults)
	assert.Equal(t, "http://localhost:8080", cfg.Defaults.AuthURL)
}

func TestLoad_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a test config file
	configContent := `current_profile: production
profiles:
  production:
    auth_url: https://auth.telhawk.example.com
    access_token: test-token-123
    refresh_token: refresh-token-456
defaults:
  auth_url: http://localhost:8080
  ingest_url: http://localhost:8088
`
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "production", cfg.CurrentProfile)
	assert.Contains(t, cfg.Profiles, "production")
	assert.Equal(t, "https://auth.telhawk.example.com", cfg.Profiles["production"].AuthURL)
	assert.Equal(t, "test-token-123", cfg.Profiles["production"].AccessToken)
	assert.Equal(t, "refresh-token-456", cfg.Profiles["production"].RefreshToken)
}

func TestLoad_WithEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	t.Setenv("THAWK_AUTH_URL", "http://env-auth-url:9000")
	t.Setenv("THAWK_INGEST_URL", "http://env-ingest-url:9001")
	t.Setenv("THAWK_QUERY_URL", "http://env-query-url:9002")

	cfg, err := Load("")
	require.NoError(t, err)

	// Environment variables should override defaults
	assert.Equal(t, "http://env-auth-url:9000", cfg.Defaults.AuthURL)
	assert.Equal(t, "http://env-ingest-url:9001", cfg.Defaults.IngestURL)
	assert.Equal(t, "http://env-query-url:9002", cfg.Defaults.QueryURL)
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".thawk", "config.yaml")

	cfg := Default()
	cfg.path = configPath
	cfg.CurrentProfile = "test-profile"

	err := cfg.Save()
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configPath)

	// Verify directory permissions (should be 0700)
	dirInfo, err := os.Stat(filepath.Dir(configPath))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), dirInfo.Mode().Perm())

	// Verify file permissions (should be 0600)
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())

	// Load the saved config and verify contents
	loadedCfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "test-profile", loadedCfg.CurrentProfile)
}

func TestSaveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Default()
	cfg.path = configPath

	err := cfg.SaveProfile("staging", "https://staging-auth.example.com", "access-abc123", "refresh-xyz789")
	require.NoError(t, err)

	// Verify profile was saved
	assert.Contains(t, cfg.Profiles, "staging")
	assert.Equal(t, "https://staging-auth.example.com", cfg.Profiles["staging"].AuthURL)
	assert.Equal(t, "access-abc123", cfg.Profiles["staging"].AccessToken)
	assert.Equal(t, "refresh-xyz789", cfg.Profiles["staging"].RefreshToken)

	// Verify current profile was updated
	assert.Equal(t, "staging", cfg.CurrentProfile)

	// Verify it was persisted to disk
	loadedCfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Contains(t, loadedCfg.Profiles, "staging")
	assert.Equal(t, "staging", loadedCfg.CurrentProfile)
}

func TestSaveProfile_MultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Default()
	cfg.path = configPath

	// Save first profile
	err := cfg.SaveProfile("dev", "http://dev-auth:8080", "dev-token", "dev-refresh")
	require.NoError(t, err)

	// Save second profile
	err = cfg.SaveProfile("prod", "https://prod-auth.example.com", "prod-token", "prod-refresh")
	require.NoError(t, err)

	// Both profiles should exist
	assert.Contains(t, cfg.Profiles, "dev")
	assert.Contains(t, cfg.Profiles, "prod")

	// Current profile should be the last one saved
	assert.Equal(t, "prod", cfg.CurrentProfile)
}

func TestGetProfile(t *testing.T) {
	cfg := Default()
	cfg.Profiles["test"] = &Profile{
		AuthURL:      "https://test-auth.example.com",
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
	}
	cfg.CurrentProfile = "test"

	tests := []struct {
		name        string
		profileName string
		wantErr     bool
		wantAuthURL string
	}{
		{
			name:        "get existing profile by name",
			profileName: "test",
			wantErr:     false,
			wantAuthURL: "https://test-auth.example.com",
		},
		{
			name:        "get current profile with empty name",
			profileName: "",
			wantErr:     false,
			wantAuthURL: "https://test-auth.example.com",
		},
		{
			name:        "get non-existent profile",
			profileName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := cfg.GetProfile(tt.profileName)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, profile)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAuthURL, profile.AuthURL)
			}
		})
	}
}

func TestRemoveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := Default()
	cfg.path = configPath
	cfg.Profiles["dev"] = &Profile{AuthURL: "http://dev:8080"}
	cfg.Profiles["prod"] = &Profile{AuthURL: "http://prod:8080"}
	cfg.CurrentProfile = "dev"

	// Remove non-current profile
	err := cfg.RemoveProfile("prod")
	require.NoError(t, err)
	assert.NotContains(t, cfg.Profiles, "prod")
	assert.Equal(t, "dev", cfg.CurrentProfile) // Current profile unchanged

	// Remove current profile
	err = cfg.RemoveProfile("dev")
	require.NoError(t, err)
	assert.NotContains(t, cfg.Profiles, "dev")
	assert.Equal(t, "", cfg.CurrentProfile) // Current profile cleared

	// Try to remove non-existent profile
	err = cfg.RemoveProfile("nonexistent")
	assert.Error(t, err)
}

func TestGetAuthURL(t *testing.T) {
	cfg := Default()
	cfg.Profiles["custom"] = &Profile{
		AuthURL: "https://custom-auth.example.com",
	}

	tests := []struct {
		name    string
		profile string
		want    string
	}{
		{
			name:    "get from profile",
			profile: "custom",
			want:    "https://custom-auth.example.com",
		},
		{
			name:    "get from defaults when profile not found",
			profile: "nonexistent",
			want:    "http://localhost:8080",
		},
		{
			name:    "get from defaults when profile empty",
			profile: "",
			want:    "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetAuthURL(tt.profile)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetIngestURL(t *testing.T) {
	cfg := Default()
	cfg.Profiles["custom"] = &Profile{
		IngestURL: "https://custom-ingest.example.com",
	}

	tests := []struct {
		name    string
		profile string
		want    string
	}{
		{
			name:    "get from profile",
			profile: "custom",
			want:    "https://custom-ingest.example.com",
		},
		{
			name:    "get from defaults when profile not found",
			profile: "nonexistent",
			want:    "http://localhost:8088",
		},
		{
			name:    "get from defaults when profile empty",
			profile: "",
			want:    "http://localhost:8088",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetIngestURL(tt.profile)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetQueryURL(t *testing.T) {
	cfg := Default()
	cfg.Profiles["custom"] = &Profile{
		QueryURL: "https://custom-query.example.com",
	}

	assert.Equal(t, "https://custom-query.example.com", cfg.GetQueryURL("custom"))
	assert.Equal(t, "http://localhost:8082", cfg.GetQueryURL("nonexistent"))
}

func TestGetRulesURL(t *testing.T) {
	cfg := Default()
	cfg.Profiles["custom"] = &Profile{
		RulesURL: "https://custom-rules.example.com",
	}

	assert.Equal(t, "https://custom-rules.example.com", cfg.GetRulesURL("custom"))
	assert.Equal(t, "http://localhost:8084", cfg.GetRulesURL("nonexistent"))
}

func TestGetAlertingURL(t *testing.T) {
	cfg := Default()
	cfg.Profiles["custom"] = &Profile{
		AlertingURL: "https://custom-alerting.example.com",
	}

	assert.Equal(t, "https://custom-alerting.example.com", cfg.GetAlertingURL("custom"))
	assert.Equal(t, "http://localhost:8085", cfg.GetAlertingURL("nonexistent"))
}

func TestGetURLFallback(t *testing.T) {
	// Test profile with partial URL configuration
	cfg := Default()
	cfg.Profiles["partial"] = &Profile{
		AuthURL: "https://custom-auth.example.com",
		// Other URLs not set
	}

	assert.Equal(t, "https://custom-auth.example.com", cfg.GetAuthURL("partial"))
	assert.Equal(t, "http://localhost:8088", cfg.GetIngestURL("partial")) // Falls back to default
	assert.Equal(t, "http://localhost:8082", cfg.GetQueryURL("partial"))  // Falls back to default
}

func TestSaveProfile_InitializesProfilesMap(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		CurrentProfile: "default",
		Profiles:       nil, // nil map
		Defaults:       Default().Defaults,
		path:           configPath,
	}

	err := cfg.SaveProfile("new", "http://new-auth:8080", "token", "refresh")
	require.NoError(t, err)

	assert.NotNil(t, cfg.Profiles)
	assert.Contains(t, cfg.Profiles, "new")
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.yaml")

	cfg := Default()
	cfg.path = configPath

	err := cfg.Save()
	require.NoError(t, err)

	// Verify nested directory was created
	assert.DirExists(t, filepath.Dir(configPath))
	assert.FileExists(t, configPath)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write YAML that will parse but fail unmarshaling
	// (viper silently ignores read errors, but unmarshal errors are returned)
	invalidYAML := `current_profile:
  - this
  - is
  - an
  - array
  - but
  - should
  - be
  - string`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0600)
	require.NoError(t, err)

	_, err = Load(configPath)
	// Should return error for unmarshal failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal config")
}

func TestSave_WithoutPath(t *testing.T) {
	// Save with empty path should use home directory
	cfg := Default()
	cfg.path = "" // No path set
	cfg.CurrentProfile = "test"

	err := cfg.Save()
	require.NoError(t, err)

	// Verify it created the file in home directory
	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, ".thawk", "config.yaml")
	assert.Equal(t, expectedPath, cfg.path)

	// Clean up
	defer os.RemoveAll(filepath.Join(home, ".thawk"))

	// Verify file exists
	assert.FileExists(t, expectedPath)
}
