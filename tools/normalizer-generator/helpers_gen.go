package main

import (
	"os"
	"path/filepath"
	"strings"
)

// generateHelpersFile creates a file with shared helper functions used by all normalizers
func generateHelpersFile(outputDir string) error {
	var buf strings.Builder

	buf.WriteString(generateHeader())
	buf.WriteString("package generated\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"strings\"\n")
	buf.WriteString("\t\"time\"\n\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf\"\n")
	buf.WriteString("\t\"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf/objects\"\n")
	buf.WriteString(")\n\n")

	buf.WriteString(`// Shared helper methods for field extraction
// These are used by all generated normalizers

// ExtractString tries multiple field names and returns the first non-empty string
func ExtractString(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := payload[key]; ok {
			if str, ok := val.(string); ok && str != "" {
				return str
			}
		}
	}
	return ""
}

// ExtractUser extracts user information from common field names
func ExtractUser(payload map[string]interface{}) *objects.User {
	name := ExtractString(payload, "user", "username", "user_name", "account", "principal")
	if name == "" {
		return nil
	}
	return &objects.User{
		Name:   name,
		Uid:    ExtractString(payload, "user_id", "uid", "user_uid"),
		Domain: ExtractString(payload, "domain", "user_domain", "realm"),
	}
}

// ExtractTimestamp parses timestamps from various formats
func ExtractTimestamp(payload map[string]interface{}, fallback time.Time) time.Time {
	for _, field := range []string{"timestamp", "time", "@timestamp", "event_time", "datetime"} {
		if val, ok := payload[field]; ok {
			switch v := val.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					return t
				}
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					return t
				}
			case float64:
				return time.Unix(int64(v), 0)
			case int64:
				return time.Unix(v, 0)
			}
		}
	}
	return fallback
}

// ExtractStatus maps status strings to OCSF status codes
func ExtractStatus(payload map[string]interface{}) (int, string) {
	status := strings.ToLower(ExtractString(payload, "status", "result", "outcome"))
	switch {
	case strings.Contains(status, "success") || strings.Contains(status, "ok"):
		return ocsf.StatusSuccess, "Success"
	case strings.Contains(status, "fail") || strings.Contains(status, "error"):
		return ocsf.StatusFailure, "Failure"
	default:
		return ocsf.StatusUnknown, "Unknown"
	}
}

// ExtractSeverity maps severity strings to OCSF severity codes
func ExtractSeverity(payload map[string]interface{}) (int, string) {
	sev := strings.ToLower(ExtractString(payload, "severity", "level", "priority"))
	switch {
	case strings.Contains(sev, "critical") || strings.Contains(sev, "fatal"):
		return ocsf.SeverityCritical, "Critical"
	case strings.Contains(sev, "high") || strings.Contains(sev, "error"):
		return ocsf.SeverityHigh, "High"
	case strings.Contains(sev, "medium") || strings.Contains(sev, "warn"):
		return ocsf.SeverityMedium, "Medium"
	case strings.Contains(sev, "low") || strings.Contains(sev, "info"):
		return ocsf.SeverityLow, "Low"
	default:
		return ocsf.SeverityUnknown, "Unknown"
	}
}

// ExtractNetworkEndpoint extracts network endpoint information
func ExtractNetworkEndpoint(ep map[string]interface{}) *objects.NetworkEndpoint {
	endpoint := &objects.NetworkEndpoint{}

	// IP address (field is "Ip" in OCSF NetworkEndpoint)
	if ip, ok := ep["ip"].(string); ok {
		endpoint.Ip = ip
	}

	// Port - OCSF uses string for port (can be numeric like "443" or named like "https")
	// Handle both integer and string types from input
	if port, ok := ep["port"].(int); ok {
		endpoint.Port = fmt.Sprintf("%d", port)
	} else if port, ok := ep["port"].(float64); ok {
		endpoint.Port = fmt.Sprintf("%d", int(port))
	} else if portStr, ok := ep["port"].(string); ok {
		endpoint.Port = portStr
	}

	// Hostname/name
	if name, ok := ep["name"].(string); ok {
		endpoint.Name = name
	}

	// UID
	if uid, ok := ep["uid"].(string); ok {
		endpoint.Uid = uid
	}

	return endpoint
}
`)

	filename := filepath.Join(outputDir, "helpers.go")
	return os.WriteFile(filename, []byte(buf.String()), 0644)
}
