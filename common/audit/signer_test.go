package audit

import (
	"testing"
	"time"
)

func TestNewEventSigner(t *testing.T) {
	secretKey := "test-secret-key"
	signer := NewEventSigner(secretKey)

	if signer == nil {
		t.Fatal("expected non-nil signer")
	}

	if string(signer.secretKey) != secretKey {
		t.Errorf("expected secret key %q, got %q", secretKey, string(signer.secretKey))
	}
}

func TestEventSigner_Sign(t *testing.T) {
	signer := NewEventSigner("test-secret")
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	eventID := "event-123"
	sourceIP := "192.168.1.100"
	data := []byte(`{"test": "data"}`)

	signature := signer.Sign(eventID, timestamp, sourceIP, data)

	// Signature should not be empty
	if signature == "" {
		t.Error("expected non-empty signature")
	}

	// Signature should be deterministic
	signature2 := signer.Sign(eventID, timestamp, sourceIP, data)
	if signature != signature2 {
		t.Error("expected deterministic signatures for same input")
	}

	// Different inputs should produce different signatures
	signature3 := signer.Sign("different-event", timestamp, sourceIP, data)
	if signature == signature3 {
		t.Error("expected different signatures for different event IDs")
	}
}

func TestEventSigner_Verify(t *testing.T) {
	signer := NewEventSigner("test-secret")
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	eventID := "event-456"
	sourceIP := "10.0.0.1"
	data := []byte(`{"user": "admin", "action": "login"}`)

	// Generate signature
	signature := signer.Sign(eventID, timestamp, sourceIP, data)

	// Verify with correct data
	if !signer.Verify(eventID, timestamp, sourceIP, data, signature) {
		t.Error("expected verification to succeed with correct data")
	}

	tests := []struct {
		name      string
		eventID   string
		timestamp time.Time
		sourceIP  string
		data      []byte
		wantValid bool
	}{
		{
			name:      "valid signature",
			eventID:   eventID,
			timestamp: timestamp,
			sourceIP:  sourceIP,
			data:      data,
			wantValid: true,
		},
		{
			name:      "wrong event ID",
			eventID:   "wrong-event",
			timestamp: timestamp,
			sourceIP:  sourceIP,
			data:      data,
			wantValid: false,
		},
		{
			name:      "wrong timestamp",
			eventID:   eventID,
			timestamp: timestamp.Add(1 * time.Hour),
			sourceIP:  sourceIP,
			data:      data,
			wantValid: false,
		},
		{
			name:      "wrong source IP",
			eventID:   eventID,
			timestamp: timestamp,
			sourceIP:  "192.168.1.1",
			data:      data,
			wantValid: false,
		},
		{
			name:      "wrong data",
			eventID:   eventID,
			timestamp: timestamp,
			sourceIP:  sourceIP,
			data:      []byte(`{"tampered": "data"}`),
			wantValid: false,
		},
		{
			name:      "empty data",
			eventID:   eventID,
			timestamp: timestamp,
			sourceIP:  sourceIP,
			data:      []byte{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := signer.Verify(tt.eventID, tt.timestamp, tt.sourceIP, tt.data, signature)
			if result != tt.wantValid {
				t.Errorf("Verify() = %v, want %v", result, tt.wantValid)
			}
		})
	}
}

func TestEventSigner_Verify_WrongSignature(t *testing.T) {
	signer := NewEventSigner("test-secret")
	timestamp := time.Now()
	eventID := "event-789"
	sourceIP := "10.0.0.5"
	data := []byte(`{"data": "value"}`)

	// Use a completely wrong signature
	wrongSignature := "0000000000000000000000000000000000000000000000000000000000000000"

	if signer.Verify(eventID, timestamp, sourceIP, data, wrongSignature) {
		t.Error("expected verification to fail with wrong signature")
	}
}

func TestEventSigner_DifferentSecrets(t *testing.T) {
	signer1 := NewEventSigner("secret-1")
	signer2 := NewEventSigner("secret-2")

	timestamp := time.Now()
	eventID := "event-abc"
	sourceIP := "10.0.0.10"
	data := []byte(`{"test": "data"}`)

	// Generate signature with signer1
	signature1 := signer1.Sign(eventID, timestamp, sourceIP, data)

	// Try to verify with signer2 (different secret)
	if signer2.Verify(eventID, timestamp, sourceIP, data, signature1) {
		t.Error("expected verification to fail with different secret key")
	}

	// Generate signature with signer2
	signature2 := signer2.Sign(eventID, timestamp, sourceIP, data)

	// Signatures should be different
	if signature1 == signature2 {
		t.Error("expected different signatures with different secret keys")
	}

	// Each signer can verify its own signature
	if !signer1.Verify(eventID, timestamp, sourceIP, data, signature1) {
		t.Error("signer1 should verify its own signature")
	}
	if !signer2.Verify(eventID, timestamp, sourceIP, data, signature2) {
		t.Error("signer2 should verify its own signature")
	}
}

func TestEventSigner_SignIngestion(t *testing.T) {
	signer := NewEventSigner("ingestion-secret")
	hecTokenID := "token-123"
	sourceIP := "192.168.1.200"
	eventCount := 42
	bytesReceived := int64(8192)
	timestamp := time.Date(2024, 2, 15, 10, 30, 0, 0, time.UTC)

	signature := signer.SignIngestion(hecTokenID, sourceIP, eventCount, bytesReceived, timestamp)

	// Signature should not be empty
	if signature == "" {
		t.Error("expected non-empty ingestion signature")
	}

	// Test that different inputs produce different signatures
	signature2 := signer.SignIngestion("different-token", sourceIP, eventCount, bytesReceived, timestamp)
	if signature == signature2 {
		t.Error("expected different signatures for different HEC token IDs")
	}

	signature3 := signer.SignIngestion(hecTokenID, "10.0.0.1", eventCount, bytesReceived, timestamp)
	if signature == signature3 {
		t.Error("expected different signatures for different source IPs")
	}
}

func TestEventSigner_SignIngestion_NonDeterministic(t *testing.T) {
	// SignIngestion uses uuid.New() internally, so signatures should be different each time
	signer := NewEventSigner("ingestion-secret")
	hecTokenID := "token-456"
	sourceIP := "10.0.0.20"
	eventCount := 10
	bytesReceived := int64(1024)
	timestamp := time.Now()

	signature1 := signer.SignIngestion(hecTokenID, sourceIP, eventCount, bytesReceived, timestamp)
	signature2 := signer.SignIngestion(hecTokenID, sourceIP, eventCount, bytesReceived, timestamp)

	// Due to uuid.New(), these should be different
	if signature1 == signature2 {
		t.Error("expected non-deterministic signatures for ingestion (due to UUID)")
	}
}

func TestEventSigner_TimestampPrecision(t *testing.T) {
	signer := NewEventSigner("precision-test")
	eventID := "event-precision"
	sourceIP := "10.0.0.30"
	data := []byte(`{"test": "data"}`)

	// Test with different timestamp precisions
	timestamp1 := time.Date(2024, 3, 1, 15, 30, 45, 123456789, time.UTC)
	signature1 := signer.Sign(eventID, timestamp1, sourceIP, data)

	// Same timestamp with different nanoseconds
	timestamp2 := time.Date(2024, 3, 1, 15, 30, 45, 987654321, time.UTC)
	signature2 := signer.Sign(eventID, timestamp2, sourceIP, data)

	// Signatures should be different (RFC3339Nano includes nanoseconds)
	if signature1 == signature2 {
		t.Error("expected different signatures for different nanosecond precision")
	}

	// Verify each signature with its corresponding timestamp
	if !signer.Verify(eventID, timestamp1, sourceIP, data, signature1) {
		t.Error("failed to verify signature1 with timestamp1")
	}
	if !signer.Verify(eventID, timestamp2, sourceIP, data, signature2) {
		t.Error("failed to verify signature2 with timestamp2")
	}

	// Cross-verification should fail
	if signer.Verify(eventID, timestamp1, sourceIP, data, signature2) {
		t.Error("expected cross-verification to fail")
	}
}

func TestEventSigner_EmptyInputs(t *testing.T) {
	signer := NewEventSigner("")
	timestamp := time.Now()

	// Sign with empty inputs
	signature := signer.Sign("", timestamp, "", []byte{})

	// Should produce a signature (even with empty inputs)
	if signature == "" {
		t.Error("expected signature even with empty inputs")
	}

	// Verify should work with same empty inputs
	if !signer.Verify("", timestamp, "", []byte{}, signature) {
		t.Error("expected verification to succeed with matching empty inputs")
	}
}

func TestEventSigner_SignatureFormat(t *testing.T) {
	signer := NewEventSigner("format-test")
	timestamp := time.Now()
	signature := signer.Sign("event-id", timestamp, "10.0.0.1", []byte("data"))

	// HMAC-SHA256 produces 32 bytes, hex encoded = 64 characters
	if len(signature) != 64 {
		t.Errorf("expected signature length of 64 characters (hex-encoded SHA256), got %d", len(signature))
	}

	// Verify it's valid hex
	for _, c := range signature {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("signature contains non-hex character: %c", c)
		}
	}
}
