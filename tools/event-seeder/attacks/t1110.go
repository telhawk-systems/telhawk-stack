package attacks

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// T1110 implements MITRE ATT&CK T1110.001 - Brute Force: Password Guessing
// Generates failed login attempts from multiple source IPs to simulate credential stuffing or distributed brute force
type T1110 struct{}

func init() {
	Register(&T1110{})
}

func (a *T1110) Name() string {
	return "T1110.001"
}

func (a *T1110) Description() string {
	return "Brute Force: Password Guessing - Multiple failed login attempts from different IPs"
}

func (a *T1110) DefaultParams() map[string]interface{} {
	return map[string]interface{}{
		"ip-count":        15,      // Number of unique source IPs
		"attempts-per-ip": 3,       // Failed login attempts per IP
		"target-user":     "admin", // Username being targeted
	}
}

func (a *T1110) Generate(cfg *Config) ([]HECEvent, error) {
	// Extract parameters with defaults
	ipCount := GetIntParam(cfg, "ip-count", 15)
	attemptsPerIP := GetIntParam(cfg, "attempts-per-ip", 3)
	targetUser := GetStringParam(cfg, "target-user", "admin")

	totalEvents := ipCount * attemptsPerIP
	events := make([]HECEvent, 0, totalEvents)

	// Generate unique source IPs
	sourceIPs := make([]string, ipCount)
	for i := 0; i < ipCount; i++ {
		sourceIPs[i] = gofakeit.IPv4Address()
	}

	eventIndex := 0
	for _, sourceIP := range sourceIPs {
		for attempt := 0; attempt < attemptsPerIP; attempt++ {
			// Use jittered distribution to spread events across time
			eventTime := calculateJitteredTime(cfg.Now, cfg.TimeSpread, eventIndex, totalEvents)

			event := map[string]interface{}{
				"class_uid":     3002, // Authentication
				"class_name":    "Authentication",
				"category_uid":  3,
				"activity_id":   1, // Login
				"activity_name": "login",
				"severity_id":   3, // Medium
				"status_id":     2, // Failure
				"status":        "Failure",
				"status_detail": selectRandomStatus(),
				"actor": map[string]interface{}{
					"user": map[string]interface{}{
						"name":  targetUser,
						"uid":   gofakeit.UUID(),
						"email": fmt.Sprintf("%s@company.local", targetUser),
					},
				},
				"src_endpoint": map[string]interface{}{
					"ip":       sourceIP,
					"port":     rand.Intn(65535-1024) + 1024,
					"hostname": gofakeit.DomainName(),
				},
				"auth_protocol": "LDAP",
				"message":       fmt.Sprintf("Failed login attempt for user %s from %s", targetUser, sourceIP),
				"metadata": map[string]interface{}{
					"product": map[string]interface{}{
						"vendor_name": "TelHawk",
						"name":        "Event Seeder - Attack Pattern T1110.001",
						"version":     "1.0.0",
					},
					"tags": []string{"attack-simulation", "T1110.001", "brute-force"},
				},
			}

			events = append(events, HECEvent{
				Time:       float64(eventTime.Unix()) + float64(eventTime.Nanosecond())/1e9,
				Event:      event,
				SourceType: "ocsf:authentication",
			})

			eventIndex++
		}
	}

	return events, nil
}

// selectRandomStatus returns a random failure status detail
func selectRandomStatus() string {
	statuses := []string{
		"Invalid credentials",
		"Account locked",
		"Password expired",
		"Invalid username or password",
		"Too many failed attempts",
	}
	return statuses[rand.Intn(len(statuses))]
}

// calculateJitteredTime calculates event time with jittered distribution
// This ensures events are spread evenly but naturally across the time window
func calculateJitteredTime(now time.Time, timeSpread time.Duration, index, total int) time.Time {
	if timeSpread == 0 {
		return now
	}

	// Calculate base interval between events
	baseInterval := float64(timeSpread) / float64(total)

	// Calculate base timestamp for this event (evenly spaced)
	baseOffset := time.Duration(float64(index) * baseInterval)

	// Add jitter: ±40% of base interval to make distribution look natural
	jitterRange := baseInterval * 0.4
	jitter := time.Duration((rand.Float64()*2.0 - 1.0) * jitterRange)

	// Calculate final timestamp, ensuring we stay within bounds
	totalOffset := baseOffset + jitter
	if totalOffset < 0 {
		totalOffset = 0
	}
	if totalOffset > timeSpread {
		totalOffset = timeSpread
	}

	// Events are placed going backwards from now
	return now.Add(-(timeSpread - totalOffset))
}
