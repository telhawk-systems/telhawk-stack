package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ============================================================================
// IngestClient Tests
// ============================================================================

func TestNewIngestClient(t *testing.T) {
	client := NewIngestClient("http://localhost:8088", "test-token")

	if client == nil {
		t.Fatal("NewIngestClient returned nil")
	}

	if client.ingestURL != "http://localhost:8088" {
		t.Errorf("Expected ingestURL 'http://localhost:8088', got %s", client.ingestURL)
	}

	if client.hecToken != "test-token" {
		t.Errorf("Expected hecToken 'test-token', got %s", client.hecToken)
	}

	if client.client == nil {
		t.Error("HTTP client should not be nil")
	}

	if client.client.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", client.client.Timeout)
	}
}

func TestForwardEvent_Success(t *testing.T) {
	// Create test server
	var receivedRequest *http.Request
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		receivedBody = make([]byte, r.ContentLength)
		r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text":"Success","code":0}`))
	}))
	defer server.Close()

	client := NewIngestClient(server.URL, "test-hec-token")

	timestamp := time.Now().UTC()
	metadata := map[string]interface{}{
		"test_key": "test_value",
	}

	err := client.ForwardEvent(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"192.168.1.1",
		"Mozilla/5.0",
		"success",
		"",
		metadata,
		timestamp,
	)

	if err != nil {
		t.Fatalf("ForwardEvent failed: %v", err)
	}

	// Verify request
	if receivedRequest.Method != "POST" {
		t.Errorf("Expected POST request, got %s", receivedRequest.Method)
	}

	if receivedRequest.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", receivedRequest.Header.Get("Content-Type"))
	}

	authHeader := receivedRequest.Header.Get("Authorization")
	if authHeader != "Splunk test-hec-token" {
		t.Errorf("Expected Authorization 'Splunk test-hec-token', got %s", authHeader)
	}

	// Verify body
	var hecEvent HECEvent
	if err := json.Unmarshal(receivedBody, &hecEvent); err != nil {
		t.Fatalf("Failed to unmarshal HEC event: %v", err)
	}

	if hecEvent.Source != "telhawk:auth" {
		t.Errorf("Expected source 'telhawk:auth', got %s", hecEvent.Source)
	}

	if hecEvent.Sourcetype != "ocsf:authentication" {
		t.Errorf("Expected sourcetype 'ocsf:authentication', got %s", hecEvent.Sourcetype)
	}

	if hecEvent.Host != "auth-service" {
		t.Errorf("Expected host 'auth-service', got %s", hecEvent.Host)
	}

	// Verify OCSF event structure
	eventBytes, _ := json.Marshal(hecEvent.Event)
	var ocsfEvent OCSFAuthEvent
	if err := json.Unmarshal(eventBytes, &ocsfEvent); err != nil {
		t.Fatalf("Failed to unmarshal OCSF event: %v", err)
	}

	if ocsfEvent.ClassUID != 3002 {
		t.Errorf("Expected ClassUID 3002, got %d", ocsfEvent.ClassUID)
	}

	if ocsfEvent.CategoryUID != 3 {
		t.Errorf("Expected CategoryUID 3, got %d", ocsfEvent.CategoryUID)
	}

	if ocsfEvent.User.Name != "testuser" {
		t.Errorf("Expected user name 'testuser', got %s", ocsfEvent.User.Name)
	}

	if ocsfEvent.User.UID != "user-123" {
		t.Errorf("Expected user UID 'user-123', got %s", ocsfEvent.User.UID)
	}

	if ocsfEvent.Status != "success" {
		t.Errorf("Expected status 'success', got %s", ocsfEvent.Status)
	}

	if ocsfEvent.StatusID != 1 {
		t.Errorf("Expected StatusID 1, got %d", ocsfEvent.StatusID)
	}

	if ocsfEvent.SrcEndpoint == nil {
		t.Fatal("Expected SrcEndpoint to be set")
	}

	if ocsfEvent.SrcEndpoint.IP != "192.168.1.1" {
		t.Errorf("Expected source IP '192.168.1.1', got %s", ocsfEvent.SrcEndpoint.IP)
	}

	// Verify observables
	if len(ocsfEvent.Observables) != 1 {
		t.Fatalf("Expected 1 observable, got %d", len(ocsfEvent.Observables))
	}

	if ocsfEvent.Observables[0].Value != "192.168.1.1" {
		t.Errorf("Expected observable value '192.168.1.1', got %s", ocsfEvent.Observables[0].Value)
	}
}

func TestForwardEvent_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var hecEvent HECEvent
		json.NewDecoder(r.Body).Decode(&hecEvent)

		eventBytes, _ := json.Marshal(hecEvent.Event)
		var ocsfEvent OCSFAuthEvent
		json.Unmarshal(eventBytes, &ocsfEvent)

		// Verify failure is properly mapped
		if ocsfEvent.StatusID != 2 {
			t.Errorf("Expected StatusID 2 for failure, got %d", ocsfEvent.StatusID)
		}

		if ocsfEvent.SeverityID != 3 {
			t.Errorf("Expected SeverityID 3 for failed login, got %d", ocsfEvent.SeverityID)
		}

		if ocsfEvent.Severity != "Medium" {
			t.Errorf("Expected Severity 'Medium' for failed login, got %s", ocsfEvent.Severity)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL, "test-token")

	err := client.ForwardEvent(
		"user",
		"user-123",
		"baduser",
		"login",
		"user",
		"user-123",
		"192.168.1.1",
		"",
		"failure",
		"invalid password",
		nil,
		time.Now(),
	)

	if err != nil {
		t.Fatalf("ForwardEvent failed: %v", err)
	}
}

func TestForwardEvent_WithErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var hecEvent HECEvent
		json.NewDecoder(r.Body).Decode(&hecEvent)

		eventBytes, _ := json.Marshal(hecEvent.Event)
		var ocsfEvent OCSFAuthEvent
		json.Unmarshal(eventBytes, &ocsfEvent)

		if ocsfEvent.Message != "login failure by testuser: invalid credentials" {
			t.Errorf("Expected message to include error, got: %s", ocsfEvent.Message)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL, "test-token")

	err := client.ForwardEvent(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"",
		"",
		"failure",
		"invalid credentials",
		nil,
		time.Now(),
	)

	if err != nil {
		t.Fatalf("ForwardEvent failed: %v", err)
	}
}

func TestForwardEvent_WithoutIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var hecEvent HECEvent
		json.NewDecoder(r.Body).Decode(&hecEvent)

		eventBytes, _ := json.Marshal(hecEvent.Event)
		var ocsfEvent OCSFAuthEvent
		json.Unmarshal(eventBytes, &ocsfEvent)

		if ocsfEvent.SrcEndpoint != nil {
			t.Error("Expected SrcEndpoint to be nil when no IP provided")
		}

		if len(ocsfEvent.Observables) != 0 {
			t.Errorf("Expected 0 observables without IP, got %d", len(ocsfEvent.Observables))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL, "test-token")

	err := client.ForwardEvent(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"", // No IP
		"",
		"success",
		"",
		nil,
		time.Now(),
	)

	if err != nil {
		t.Fatalf("ForwardEvent failed: %v", err)
	}
}

func TestForwardEvent_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewIngestClient(server.URL, "test-token")

	err := client.ForwardEvent(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"",
		"",
		"success",
		"",
		nil,
		time.Now(),
	)

	if err == nil {
		t.Fatal("Expected error when ingest returns 500, got nil")
	}

	expectedErr := "ingest returned status 500"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestForwardEvent_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewIngestClient("http://invalid-host-that-does-not-exist:9999", "test-token")

	err := client.ForwardEvent(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"",
		"",
		"success",
		"",
		nil,
		time.Now(),
	)

	if err == nil {
		t.Fatal("Expected network error, got nil")
	}
}

// ============================================================================
// Mapping Function Tests
// ============================================================================

func TestMapActionToActivityID(t *testing.T) {
	tests := []struct {
		action     string
		expectedID int
	}{
		{"login", 1},
		{"logout", 2},
		{"register", 1},
		{"token_revoke", 2},
		{"hec_token_create", 3},
		{"hec_token_revoke", 99},
		{"user_update", 99},
		{"user_delete", 99},
		{"password_change", 99},
		{"unknown_action", 99},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := mapActionToActivityID(tt.action)
			if result != tt.expectedID {
				t.Errorf("Expected activity ID %d for action '%s', got %d", tt.expectedID, tt.action, result)
			}
		})
	}
}

func TestMapResultToSeverity(t *testing.T) {
	tests := []struct {
		result           string
		action           string
		expectedSeverity string
	}{
		{"success", "login", "Informational"},
		{"success", "logout", "Informational"},
		{"failure", "login", "Medium"},
		{"failure", "logout", "Low"},
		{"failure", "user_update", "Low"},
	}

	for _, tt := range tests {
		t.Run(tt.result+"_"+tt.action, func(t *testing.T) {
			result := mapResultToSeverity(tt.result, tt.action)
			if result != tt.expectedSeverity {
				t.Errorf("Expected severity '%s' for result=%s action=%s, got '%s'",
					tt.expectedSeverity, tt.result, tt.action, result)
			}
		})
	}
}

func TestMapResultToSeverityID(t *testing.T) {
	tests := []struct {
		result     string
		action     string
		expectedID int
	}{
		{"success", "login", 1},
		{"success", "logout", 1},
		{"failure", "login", 3},
		{"failure", "logout", 2},
		{"failure", "user_update", 2},
	}

	for _, tt := range tests {
		t.Run(tt.result+"_"+tt.action, func(t *testing.T) {
			result := mapResultToSeverityID(tt.result, tt.action)
			if result != tt.expectedID {
				t.Errorf("Expected severity ID %d for result=%s action=%s, got %d",
					tt.expectedID, tt.result, tt.action, result)
			}
		})
	}
}

func TestMapActorTypeToID(t *testing.T) {
	tests := []struct {
		actorType  string
		expectedID int
	}{
		{"user", 1},
		{"service", 3},
		{"system", 3},
		{"unknown", 99},
	}

	for _, tt := range tests {
		t.Run(tt.actorType, func(t *testing.T) {
			result := mapActorTypeToID(tt.actorType)
			if result != tt.expectedID {
				t.Errorf("Expected actor type ID %d for '%s', got %d", tt.expectedID, tt.actorType, result)
			}
		})
	}
}

// ============================================================================
// OCSF Event Structure Tests
// ============================================================================

func TestOCSFEventSerialization(t *testing.T) {
	event := OCSFAuthEvent{
		ActivityID:  1,
		CategoryUID: 3,
		ClassUID:    3002,
		Time:        time.Now().UnixMilli(),
		TypeUID:     300201,
		Severity:    "Informational",
		SeverityID:  1,
		Status:      "success",
		StatusID:    1,
		User: OCSFUser{
			Name:   "testuser",
			UID:    "user-123",
			Type:   "user",
			TypeID: 1,
		},
		SrcEndpoint: &OCSFEndpoint{
			IP: "192.168.1.1",
		},
		Metadata: OCSFMetadata{
			Version: "1.1.0",
			Product: OCSFProduct{
				Name:    "TelHawk Auth",
				Vendor:  "TelHawk Systems",
				Version: "1.0.0",
			},
		},
		Message: "login success by testuser",
	}

	// Test serialization
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal OCSF event: %v", err)
	}

	// Test deserialization
	var decoded OCSFAuthEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal OCSF event: %v", err)
	}

	if decoded.ClassUID != event.ClassUID {
		t.Errorf("Expected ClassUID %d, got %d", event.ClassUID, decoded.ClassUID)
	}

	if decoded.User.Name != event.User.Name {
		t.Errorf("Expected user name '%s', got '%s'", event.User.Name, decoded.User.Name)
	}
}

func TestHECEventSerialization(t *testing.T) {
	ocsfEvent := OCSFAuthEvent{
		ClassUID: 3002,
		User: OCSFUser{
			Name: "testuser",
		},
	}

	hecEvent := HECEvent{
		Time:       time.Now().Unix(),
		Source:     "telhawk:auth",
		Sourcetype: "ocsf:authentication",
		Host:       "auth-service",
		Event:      ocsfEvent,
	}

	// Test serialization
	data, err := json.Marshal(hecEvent)
	if err != nil {
		t.Fatalf("Failed to marshal HEC event: %v", err)
	}

	// Test deserialization
	var decoded HECEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal HEC event: %v", err)
	}

	if decoded.Source != hecEvent.Source {
		t.Errorf("Expected source '%s', got '%s'", hecEvent.Source, decoded.Source)
	}
}
