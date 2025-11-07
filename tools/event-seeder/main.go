package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

var (
	hecURL      = flag.String("hec-url", "http://localhost:8088", "HEC endpoint URL")
	hecToken    = flag.String("token", "", "HEC authentication token (required)")
	count       = flag.Int("count", 100, "Number of events to generate")
	interval    = flag.Duration("interval", 100*time.Millisecond, "Interval between events")
	eventTypes  = flag.String("types", "auth,network,process,file,dns,http,detection", "Comma-separated list of event types")
	timeSpread  = flag.Duration("time-spread", 24*time.Hour, "Spread events over this time period (0 for real-time)")
	batchSize   = flag.Int("batch-size", 10, "Number of events per batch")
)

type HECEvent struct {
	Time       float64                `json:"time"`
	Event      map[string]interface{} `json:"event"`
	SourceType string                 `json:"sourcetype"`
	Index      string                 `json:"index,omitempty"`
}

func main() {
	flag.Parse()

	if *hecToken == "" {
		log.Fatal("HEC token is required. Use -token flag")
	}

	gofakeit.Seed(time.Now().UnixNano())

	log.Printf("Starting event seeder:")
	log.Printf("  HEC URL: %s", *hecURL)
	log.Printf("  Event count: %d", *count)
	log.Printf("  Interval: %v", *interval)
	log.Printf("  Batch size: %d", *batchSize)
	log.Printf("  Time spread: %v", *timeSpread)

	types := parseEventTypes(*eventTypes)
	log.Printf("  Event types: %v", types)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	successCount := 0
	failCount := 0

	batch := make([]HECEvent, 0, *batchSize)
	
	for i := 0; i < *count; i++ {
		eventType := types[rand.Intn(len(types))]
		event := generateEvent(eventType, i)
		batch = append(batch, event)

		if len(batch) >= *batchSize || i == *count-1 {
			if err := sendBatch(client, *hecURL, *hecToken, batch); err != nil {
				log.Printf("Failed to send batch: %v", err)
				failCount += len(batch)
			} else {
				successCount += len(batch)
				if successCount%50 == 0 {
					log.Printf("Progress: %d/%d events sent", successCount, *count)
				}
			}
			batch = batch[:0]
		}

		if *interval > 0 && i < *count-1 {
			time.Sleep(*interval)
		}
	}

	log.Printf("\nSeeding complete:")
	log.Printf("  Success: %d events", successCount)
	log.Printf("  Failed: %d events", failCount)
}

func parseEventTypes(types string) []string {
	result := []string{}
	current := ""
	for _, c := range types {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func generateEvent(eventType string, index int) HECEvent {
	now := time.Now()
	
	var eventTime time.Time
	if *timeSpread > 0 {
		offset := time.Duration(rand.Int63n(int64(*timeSpread)))
		eventTime = now.Add(-offset)
	} else {
		eventTime = now
	}

	var event map[string]interface{}
	var sourcetype string

	switch eventType {
	case "auth":
		event = generateAuthEvent()
		sourcetype = "ocsf:authentication"
	case "network":
		event = generateNetworkEvent()
		sourcetype = "ocsf:network_activity"
	case "process":
		event = generateProcessEvent()
		sourcetype = "ocsf:process_activity"
	case "file":
		event = generateFileEvent()
		sourcetype = "ocsf:file_activity"
	case "dns":
		event = generateDNSEvent()
		sourcetype = "ocsf:dns_activity"
	case "http":
		event = generateHTTPEvent()
		sourcetype = "ocsf:http_activity"
	case "detection":
		event = generateDetectionEvent()
		sourcetype = "ocsf:detection_finding"
	default:
		event = generateAuthEvent()
		sourcetype = "ocsf:authentication"
	}

	return HECEvent{
		Time:       float64(eventTime.Unix()) + float64(eventTime.Nanosecond())/1e9,
		Event:      event,
		SourceType: sourcetype,
	}
}

func generateAuthEvent() map[string]interface{} {
	actions := []string{"login", "logout", "mfa_verify", "password_change"}
	action := actions[rand.Intn(len(actions))]
	
	success := rand.Float32() > 0.15 // 85% success rate
	
	event := map[string]interface{}{
		"class_uid":  3002,
		"class_name": "Authentication",
		"activity_id": func() int {
			if action == "login" {
				return 1
			} else if action == "logout" {
				return 2
			}
			return 99
		}(),
		"activity_name": action,
		"severity_id": func() int {
			if !success {
				return 3 // Medium
			}
			return 1 // Informational
		}(),
		"status": func() string {
			if success {
				return "Success"
			}
			return "Failure"
		}(),
		"user": map[string]interface{}{
			"name":   gofakeit.Username(),
			"uid":    gofakeit.UUID(),
			"email":  gofakeit.Email(),
		},
		"src_endpoint": map[string]interface{}{
			"ip":       gofakeit.IPv4Address(),
			"port":     rand.Intn(65535-1024) + 1024,
			"hostname": gofakeit.DomainName(),
		},
		"auth_protocol": "LDAP",
		"message":       fmt.Sprintf("User %s attempt", action),
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
				"version":     "1.0.0",
			},
		},
	}

	if !success {
		event["status_detail"] = []string{"Invalid credentials", "Account locked", "Expired password"}[rand.Intn(3)]
	}

	return event
}

func generateNetworkEvent() map[string]interface{} {
	actions := []int{1, 2, 5, 6} // open, close, traffic, refuse
	actionNames := map[int]string{1: "Open", 2: "Close", 5: "Traffic", 6: "Refuse"}
	action := actions[rand.Intn(len(actions))]
	
	protocols := []string{"TCP", "UDP", "ICMP"}
	protocol := protocols[rand.Intn(len(protocols))]

	return map[string]interface{}{
		"class_uid":     4001,
		"class_name":    "Network Activity",
		"activity_id":   action,
		"activity_name": actionNames[action],
		"severity_id":   1,
		"src_endpoint": map[string]interface{}{
			"ip":   gofakeit.IPv4Address(),
			"port": rand.Intn(65535-1024) + 1024,
		},
		"dst_endpoint": map[string]interface{}{
			"ip":   gofakeit.IPv4Address(),
			"port": []int{80, 443, 22, 3389, 445, 3306, 5432}[rand.Intn(7)],
		},
		"connection_info": map[string]interface{}{
			"protocol_name": protocol,
			"direction":     []string{"inbound", "outbound"}[rand.Intn(2)],
			"boundary":      []string{"internal", "external"}[rand.Intn(2)],
		},
		"traffic": map[string]interface{}{
			"bytes": rand.Intn(1000000),
			"packets": rand.Intn(10000),
		},
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
			},
		},
	}
}

func generateProcessEvent() map[string]interface{} {
	commands := []string{
		"/bin/bash -c 'ls -la'",
		"python3 /opt/scripts/backup.py",
		"/usr/bin/curl https://api.example.com",
		"docker run -d nginx",
		"systemctl restart nginx",
		"npm install express",
	}
	
	return map[string]interface{}{
		"class_uid":     1007,
		"class_name":    "Process Activity",
		"activity_id":   1, // Launch
		"activity_name": "Launch",
		"severity_id":   1,
		"process": map[string]interface{}{
			"pid":          rand.Intn(65535),
			"name":         []string{"bash", "python3", "curl", "docker", "systemctl", "npm"}[rand.Intn(6)],
			"cmd_line":     commands[rand.Intn(len(commands))],
			"uid":          gofakeit.UUID(),
			"parent_process": map[string]interface{}{
				"pid":  rand.Intn(65535),
				"name": "systemd",
			},
		},
		"actor": map[string]interface{}{
			"user": map[string]interface{}{
				"name": gofakeit.Username(),
				"uid":  gofakeit.UUID(),
			},
		},
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
			},
		},
	}
}

func generateFileEvent() map[string]interface{} {
	actions := []int{1, 2, 3, 4, 5} // create, read, update, delete, rename
	actionNames := map[int]string{1: "Create", 2: "Read", 3: "Update", 4: "Delete", 5: "Rename"}
	action := actions[rand.Intn(len(actions))]
	
	paths := []string{
		"/var/log/app.log",
		"/etc/nginx/nginx.conf",
		"/home/user/documents/report.pdf",
		"/tmp/upload_" + gofakeit.UUID()[:8],
		"/opt/data/config.json",
	}

	return map[string]interface{}{
		"class_uid":     4006,
		"class_name":    "File Activity",
		"activity_id":   action,
		"activity_name": actionNames[action],
		"severity_id":   1,
		"file": map[string]interface{}{
			"path":      paths[rand.Intn(len(paths))],
			"type":      []string{"Regular File", "Directory", "Symbolic Link"}[rand.Intn(3)],
			"size":      rand.Intn(10000000),
			"modified_time": time.Now().Unix(),
		},
		"actor": map[string]interface{}{
			"user": map[string]interface{}{
				"name": gofakeit.Username(),
				"uid":  gofakeit.UUID(),
			},
		},
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
			},
		},
	}
}

func generateDNSEvent() map[string]interface{} {
	domains := []string{
		"example.com",
		"api.github.com",
		"malicious-site.ru",
		"cdn.cloudflare.net",
		"login.microsoft.com",
		"updates.ubuntu.com",
	}
	
	rTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT"}
	rType := rTypes[rand.Intn(len(rTypes))]
	
	return map[string]interface{}{
		"class_uid":     4003,
		"class_name":    "DNS Activity",
		"activity_id":   1, // Query
		"activity_name": "Query",
		"severity_id":   1,
		"query": map[string]interface{}{
			"hostname": domains[rand.Intn(len(domains))],
			"type":     rType,
			"class":    "IN",
		},
		"answers": []map[string]interface{}{
			{
				"type":  rType,
				"rdata": gofakeit.IPv4Address(),
				"ttl":   300,
			},
		},
		"src_endpoint": map[string]interface{}{
			"ip":   gofakeit.IPv4Address(),
			"port": rand.Intn(65535-1024) + 1024,
		},
		"response_code": []string{"NOERROR", "NXDOMAIN", "SERVFAIL"}[rand.Intn(3)],
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
			},
		},
	}
}

func generateHTTPEvent() map[string]interface{} {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	statusCodes := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503}
	paths := []string{
		"/api/v1/users",
		"/api/v1/auth/login",
		"/api/v1/events",
		"/health",
		"/metrics",
		"/admin/dashboard",
	}
	
	statusCode := statusCodes[rand.Intn(len(statusCodes))]
	
	return map[string]interface{}{
		"class_uid":     4002,
		"class_name":    "HTTP Activity",
		"activity_id":   1,
		"activity_name": "HTTP Request",
		"severity_id": func() int {
			if statusCode >= 500 {
				return 3 // Medium
			} else if statusCode >= 400 {
				return 2 // Low
			}
			return 1 // Informational
		}(),
		"http_request": map[string]interface{}{
			"method":     methods[rand.Intn(len(methods))],
			"url":        map[string]interface{}{
				"path":   paths[rand.Intn(len(paths))],
				"scheme": "https",
				"hostname": gofakeit.DomainName(),
			},
			"user_agent": gofakeit.UserAgent(),
		},
		"http_response": map[string]interface{}{
			"code":   statusCode,
			"length": rand.Intn(100000),
		},
		"src_endpoint": map[string]interface{}{
			"ip":   gofakeit.IPv4Address(),
			"port": rand.Intn(65535-1024) + 1024,
		},
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
			},
		},
	}
}

func generateDetectionEvent() map[string]interface{} {
	findings := []struct {
		title    string
		severity int
		tactic   string
	}{
		{"Suspicious PowerShell Execution", 4, "Execution"},
		{"Multiple Failed Login Attempts", 3, "Credential Access"},
		{"Unusual Network Traffic Pattern", 3, "Command and Control"},
		{"Potential Data Exfiltration", 4, "Exfiltration"},
		{"Privilege Escalation Attempt", 4, "Privilege Escalation"},
		{"Malware Detected", 5, "Malware"},
	}
	
	finding := findings[rand.Intn(len(findings))]
	
	return map[string]interface{}{
		"class_uid":     2004,
		"class_name":    "Detection Finding",
		"activity_id":   1, // Create
		"activity_name": "Create",
		"severity_id":   finding.severity,
		"finding": map[string]interface{}{
			"title":       finding.title,
			"uid":         gofakeit.UUID(),
			"types":       []string{"Threat Detection"},
			"created_time": time.Now().Unix(),
		},
		"attacks": []map[string]interface{}{
			{
				"tactic": map[string]interface{}{
					"name": finding.tactic,
				},
			},
		},
		"resources": []map[string]interface{}{
			{
				"name": gofakeit.DomainName(),
				"type": "endpoint",
			},
		},
		"metadata": map[string]interface{}{
			"product": map[string]interface{}{
				"vendor_name": "TelHawk",
				"name":        "Event Seeder",
			},
		},
	}
}

func sendBatch(client *http.Client, hecURL, token string, events []HECEvent) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to encode event: %w", err)
		}
	}

	req, err := http.NewRequest("POST", hecURL+"/services/collector/event", &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Splunk "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HEC returned status %d", resp.StatusCode)
	}

	return nil
}
