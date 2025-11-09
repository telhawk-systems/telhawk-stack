package main

import (
	"encoding/json"
	"testing"
)

func TestGenerateAuthEvent(t *testing.T) {
	// Run multiple times to test randomization
	loginFailures := 0
	logoutFailures := 0
	totalLogins := 0
	totalLogouts := 0

	for i := 0; i < 1000; i++ {
		event := generateAuthEvent()

		// Verify required fields
		if event["class_uid"] != 3002 {
			t.Errorf("Expected class_uid 3002, got %v", event["class_uid"])
		}
		if event["class_name"] != "Authentication" {
			t.Errorf("Expected class_name Authentication, got %v", event["class_name"])
		}

		// Verify actor.user object exists (OCSF compliant)
		actor, ok := event["actor"].(map[string]interface{})
		if !ok {
			t.Fatal("actor field missing or wrong type")
		}
		user, ok := actor["user"].(map[string]interface{})
		if !ok {
			t.Fatal("actor.user field missing or wrong type")
		}
		if user["name"] == nil || user["uid"] == nil || user["email"] == nil {
			t.Error("actor.user missing required fields")
		}

		// Verify src_endpoint exists
		endpoint, ok := event["src_endpoint"].(map[string]interface{})
		if !ok {
			t.Fatal("src_endpoint field missing or wrong type")
		}
		if endpoint["ip"] == nil || endpoint["port"] == nil {
			t.Error("src_endpoint missing required fields")
		}

		// Track failure rates by action type
		action := event["activity_name"].(string)
		status := event["status"].(string)
		
		if action == "login" {
			totalLogins++
			if status == "Failure" {
				loginFailures++
			}
		} else if action == "logout" {
			totalLogouts++
			if status == "Failure" {
				logoutFailures++
			}
		}

		// Verify status_detail only present on failures
		if status == "Failure" {
			if event["status_detail"] == nil {
				t.Error("status_detail should be present on failures")
			}
		}
	}

	// Verify login failure rate is around 15%
	if totalLogins > 0 {
		loginFailureRate := float64(loginFailures) / float64(totalLogins)
		if loginFailureRate < 0.10 || loginFailureRate > 0.20 {
			t.Errorf("Login failure rate %.2f%% outside expected range (10-20%%)", loginFailureRate*100)
		}
	}

	// Verify logout failure rate is very low (around 2%)
	if totalLogouts > 0 {
		logoutFailureRate := float64(logoutFailures) / float64(totalLogouts)
		if logoutFailureRate > 0.05 {
			t.Errorf("Logout failure rate %.2f%% too high (expected <5%%)", logoutFailureRate*100)
		}
	}
}

func TestGenerateNetworkEvent(t *testing.T) {
	event := generateNetworkEvent()

	if event["class_uid"] != 4001 {
		t.Errorf("Expected class_uid 4001, got %v", event["class_uid"])
	}
	if event["class_name"] != "Network Activity" {
		t.Errorf("Expected class_name 'Network Activity', got %v", event["class_name"])
	}

	// Verify endpoints
	srcEndpoint, ok := event["src_endpoint"].(map[string]interface{})
	if !ok {
		t.Fatal("src_endpoint missing or wrong type")
	}
	if srcEndpoint["ip"] == nil || srcEndpoint["port"] == nil {
		t.Error("src_endpoint missing required fields")
	}

	dstEndpoint, ok := event["dst_endpoint"].(map[string]interface{})
	if !ok {
		t.Fatal("dst_endpoint missing or wrong type")
	}
	if dstEndpoint["ip"] == nil || dstEndpoint["port"] == nil {
		t.Error("dst_endpoint missing required fields")
	}

	// Verify connection_info
	connInfo, ok := event["connection_info"].(map[string]interface{})
	if !ok {
		t.Fatal("connection_info missing or wrong type")
	}
	if connInfo["protocol_name"] == nil {
		t.Error("connection_info missing protocol_name")
	}

	// Verify traffic
	traffic, ok := event["traffic"].(map[string]interface{})
	if !ok {
		t.Fatal("traffic missing or wrong type")
	}
	if traffic["bytes"] == nil || traffic["packets"] == nil {
		t.Error("traffic missing required fields")
	}
}

func TestGenerateProcessEvent(t *testing.T) {
	event := generateProcessEvent()

	if event["class_uid"] != 1007 {
		t.Errorf("Expected class_uid 1007, got %v", event["class_uid"])
	}
	if event["class_name"] != "Process Activity" {
		t.Errorf("Expected class_name 'Process Activity', got %v", event["class_name"])
	}

	// Verify process object
	process, ok := event["process"].(map[string]interface{})
	if !ok {
		t.Fatal("process missing or wrong type")
	}
	if process["pid"] == nil || process["name"] == nil || process["cmd_line"] == nil {
		t.Error("process missing required fields")
	}

	// Verify parent_process exists
	parentProc, ok := process["parent_process"].(map[string]interface{})
	if !ok {
		t.Fatal("parent_process missing or wrong type")
	}
	if parentProc["pid"] == nil || parentProc["name"] == nil {
		t.Error("parent_process missing required fields")
	}

	// Verify actor
	actor, ok := event["actor"].(map[string]interface{})
	if !ok {
		t.Fatal("actor missing or wrong type")
	}
	if actor["user"] == nil {
		t.Error("actor missing user field")
	}
}

func TestGenerateFileEvent(t *testing.T) {
	event := generateFileEvent()

	if event["class_uid"] != 4006 {
		t.Errorf("Expected class_uid 4006, got %v", event["class_uid"])
	}
	if event["class_name"] != "File Activity" {
		t.Errorf("Expected class_name 'File Activity', got %v", event["class_name"])
	}

	// Verify file object
	file, ok := event["file"].(map[string]interface{})
	if !ok {
		t.Fatal("file missing or wrong type")
	}
	if file["path"] == nil || file["type"] == nil || file["size"] == nil {
		t.Error("file missing required fields")
	}

	// Verify actor
	actor, ok := event["actor"].(map[string]interface{})
	if !ok {
		t.Fatal("actor missing or wrong type")
	}
	if actor["user"] == nil {
		t.Error("actor missing user field")
	}
}

func TestGenerateDNSEvent(t *testing.T) {
	event := generateDNSEvent()

	if event["class_uid"] != 4003 {
		t.Errorf("Expected class_uid 4003, got %v", event["class_uid"])
	}
	if event["class_name"] != "DNS Activity" {
		t.Errorf("Expected class_name 'DNS Activity', got %v", event["class_name"])
	}

	// Verify query object
	query, ok := event["query"].(map[string]interface{})
	if !ok {
		t.Fatal("query missing or wrong type")
	}
	if query["hostname"] == nil || query["type"] == nil {
		t.Error("query missing required fields")
	}

	// Verify answers
	answers, ok := event["answers"].([]map[string]interface{})
	if !ok {
		t.Fatal("answers missing or wrong type")
	}
	if len(answers) == 0 {
		t.Error("answers array is empty")
	}

	// Verify src_endpoint
	if event["src_endpoint"] == nil {
		t.Error("src_endpoint missing")
	}
}

func TestGenerateHTTPEvent(t *testing.T) {
	event := generateHTTPEvent()

	if event["class_uid"] != 4002 {
		t.Errorf("Expected class_uid 4002, got %v", event["class_uid"])
	}
	if event["class_name"] != "HTTP Activity" {
		t.Errorf("Expected class_name 'HTTP Activity', got %v", event["class_name"])
	}

	// Verify http_request
	req, ok := event["http_request"].(map[string]interface{})
	if !ok {
		t.Fatal("http_request missing or wrong type")
	}
	if req["method"] == nil || req["url"] == nil {
		t.Error("http_request missing required fields")
	}

	// Verify http_response
	resp, ok := event["http_response"].(map[string]interface{})
	if !ok {
		t.Fatal("http_response missing or wrong type")
	}
	if resp["code"] == nil {
		t.Error("http_response missing code field")
	}

	// Verify severity_id is set appropriately for status codes
	statusCode := resp["code"].(int)
	severityID := event["severity_id"].(int)
	
	if statusCode >= 500 && severityID < 3 {
		t.Error("5xx errors should have severity_id >= 3")
	}
}

func TestGenerateDetectionEvent(t *testing.T) {
	event := generateDetectionEvent()

	if event["class_uid"] != 2004 {
		t.Errorf("Expected class_uid 2004, got %v", event["class_uid"])
	}
	if event["class_name"] != "Detection Finding" {
		t.Errorf("Expected class_name 'Detection Finding', got %v", event["class_name"])
	}

	// Verify finding object
	finding, ok := event["finding"].(map[string]interface{})
	if !ok {
		t.Fatal("finding missing or wrong type")
	}
	if finding["title"] == nil || finding["uid"] == nil {
		t.Error("finding missing required fields")
	}

	// Verify attacks array
	attacks, ok := event["attacks"].([]map[string]interface{})
	if !ok {
		t.Fatal("attacks missing or wrong type")
	}
	if len(attacks) == 0 {
		t.Error("attacks array is empty")
	}

	// Verify resources array
	resources, ok := event["resources"].([]map[string]interface{})
	if !ok {
		t.Fatal("resources missing or wrong type")
	}
	if len(resources) == 0 {
		t.Error("resources array is empty")
	}

	// Verify severity is reasonable for security findings
	severityID := event["severity_id"].(int)
	if severityID < 3 || severityID > 5 {
		t.Errorf("Detection finding severity_id %d outside expected range (3-5)", severityID)
	}
}

func TestGenerateEventAllTypes(t *testing.T) {
	types := []string{"auth", "network", "process", "file", "dns", "http", "detection"}
	
	for _, eventType := range types {
		event := generateEvent(eventType, 0)
		
		// Verify HECEvent structure
		if event.Time == 0 {
			t.Errorf("Event type %s has zero time", eventType)
		}
		if event.SourceType == "" {
			t.Errorf("Event type %s has empty sourcetype", eventType)
		}
		if event.Event == nil || len(event.Event) == 0 {
			t.Errorf("Event type %s has empty event payload", eventType)
		}

		// Verify event can be marshaled to JSON
		_, err := json.Marshal(event)
		if err != nil {
			t.Errorf("Event type %s failed to marshal to JSON: %v", eventType, err)
		}

		// Verify sourcetype matches expected pattern
		expectedSourcetype := "ocsf:" + eventType
		if eventType == "auth" {
			expectedSourcetype = "ocsf:authentication"
		}
		// Handle other type name variations
		if eventType == "network" {
			expectedSourcetype = "ocsf:network_activity"
		}
		if eventType == "process" {
			expectedSourcetype = "ocsf:process_activity"
		}
		if eventType == "file" {
			expectedSourcetype = "ocsf:file_activity"
		}
		if eventType == "dns" {
			expectedSourcetype = "ocsf:dns_activity"
		}
		if eventType == "http" {
			expectedSourcetype = "ocsf:http_activity"
		}
		if eventType == "detection" {
			expectedSourcetype = "ocsf:detection_finding"
		}
		
		if event.SourceType != expectedSourcetype {
			t.Errorf("Event type %s has wrong sourcetype: expected %s, got %s", 
				eventType, expectedSourcetype, event.SourceType)
		}
	}
}

func TestParseEventTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"auth", []string{"auth"}},
		{"auth,network", []string{"auth", "network"}},
		{"auth,network,process,file", []string{"auth", "network", "process", "file"}},
		{"", []string{}}, // Empty string should return empty slice
		{"single", []string{"single"}},
	}

	for _, tt := range tests {
		result := parseEventTypes(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseEventTypes(%q) returned %d items, expected %d", 
				tt.input, len(result), len(tt.expected))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("parseEventTypes(%q)[%d] = %q, expected %q", 
					tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestEventMetadataConsistency(t *testing.T) {
	// Test that all event types have consistent metadata
	generators := map[string]func() map[string]interface{}{
		"auth":      generateAuthEvent,
		"network":   generateNetworkEvent,
		"process":   generateProcessEvent,
		"file":      generateFileEvent,
		"dns":       generateDNSEvent,
		"http":      generateHTTPEvent,
		"detection": generateDetectionEvent,
	}

	for eventType, generator := range generators {
		event := generator()
		
		// Verify metadata exists
		metadata, ok := event["metadata"].(map[string]interface{})
		if !ok {
			t.Errorf("%s event missing metadata", eventType)
			continue
		}

		// Verify product info
		product, ok := metadata["product"].(map[string]interface{})
		if !ok {
			t.Errorf("%s event metadata missing product", eventType)
			continue
		}

		if product["vendor_name"] != "TelHawk" {
			t.Errorf("%s event has wrong vendor_name: %v", eventType, product["vendor_name"])
		}
		if product["name"] != "Event Seeder" {
			t.Errorf("%s event has wrong product name: %v", eventType, product["name"])
		}

		// Verify class_uid is set
		if event["class_uid"] == nil {
			t.Errorf("%s event missing class_uid", eventType)
		}

		// Verify class_name is set
		if event["class_name"] == nil {
			t.Errorf("%s event missing class_name", eventType)
		}

		// Verify severity_id is set
		if event["severity_id"] == nil {
			t.Errorf("%s event missing severity_id", eventType)
		}
	}
}
