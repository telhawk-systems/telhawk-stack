package attacks

import (
	"strconv"
	"time"
)

// HECEvent represents an event to be sent to HEC
type HECEvent struct {
	Time       float64                `json:"time"`
	Event      map[string]interface{} `json:"event"`
	SourceType string                 `json:"sourcetype"`
	Index      string                 `json:"index,omitempty"`
}

// Config holds configuration for attack pattern generation
type Config struct {
	// Time configuration
	Now        time.Time
	TimeSpread time.Duration

	// Attack-specific parameters (flexible key-value store)
	// Common parameters: target-user, target-host, source-ip, etc.
	Params map[string]interface{}
}

// Pattern represents a suspicious activity/attack pattern generator
type Pattern interface {
	// Name returns the MITRE ATT&CK technique ID (e.g., "T1110.001")
	Name() string

	// Description returns a human-readable description
	Description() string

	// Generate creates the attack events based on the configuration
	Generate(cfg *Config) ([]HECEvent, error)

	// DefaultParams returns default parameters for this attack pattern
	DefaultParams() map[string]interface{}
}

// Registry holds all registered attack patterns
var Registry = make(map[string]Pattern)

// Register adds an attack pattern to the registry
func Register(pattern Pattern) {
	Registry[pattern.Name()] = pattern
}

// Get retrieves an attack pattern by name
func Get(name string) (Pattern, bool) {
	p, ok := Registry[name]
	return p, ok
}

// List returns all registered attack pattern names
func List() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// GetParam is a helper to safely extract typed parameters from Config.Params
func GetParam[T any](cfg *Config, key string, defaultValue T) T {
	if cfg.Params == nil {
		return defaultValue
	}
	if val, ok := cfg.Params[key]; ok {
		if typedVal, ok := val.(T); ok {
			return typedVal
		}
	}
	return defaultValue
}

// GetIntParam extracts an integer parameter, parsing from string if necessary
func GetIntParam(cfg *Config, key string, defaultValue int) int {
	if cfg.Params == nil {
		return defaultValue
	}

	val, ok := cfg.Params[key]
	if !ok {
		return defaultValue
	}

	// Try direct int conversion
	if intVal, ok := val.(int); ok {
		return intVal
	}

	// Try parsing from string
	if strVal, ok := val.(string); ok {
		if parsed, err := strconv.Atoi(strVal); err == nil {
			return parsed
		}
	}

	return defaultValue
}

// GetStringParam extracts a string parameter
func GetStringParam(cfg *Config, key string, defaultValue string) string {
	if cfg.Params == nil {
		return defaultValue
	}

	val, ok := cfg.Params[key]
	if !ok {
		return defaultValue
	}

	if strVal, ok := val.(string); ok {
		return strVal
	}

	return defaultValue
}
