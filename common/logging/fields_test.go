package logging

import (
	"errors"
	"log/slog"
	"testing"
)

func TestService(t *testing.T) {
	attr := Service("auth-service")
	if attr.Key != FieldService {
		t.Errorf("expected key %q, got %q", FieldService, attr.Key)
	}
	if attr.Value.String() != "auth-service" {
		t.Errorf("expected value %q, got %q", "auth-service", attr.Value.String())
	}
}

func TestUserID(t *testing.T) {
	attr := UserID("user-123")
	if attr.Key != FieldUserID {
		t.Errorf("expected key %q, got %q", FieldUserID, attr.Key)
	}
	if attr.Value.String() != "user-123" {
		t.Errorf("expected value %q, got %q", "user-123", attr.Value.String())
	}
}

func TestUsername(t *testing.T) {
	attr := Username("admin")
	if attr.Key != FieldUsername {
		t.Errorf("expected key %q, got %q", FieldUsername, attr.Key)
	}
	if attr.Value.String() != "admin" {
		t.Errorf("expected value %q, got %q", "admin", attr.Value.String())
	}
}

func TestIP(t *testing.T) {
	attr := IP("192.168.1.1")
	if attr.Key != FieldIP {
		t.Errorf("expected key %q, got %q", FieldIP, attr.Key)
	}
	if attr.Value.String() != "192.168.1.1" {
		t.Errorf("expected value %q, got %q", "192.168.1.1", attr.Value.String())
	}
}

func TestMethod(t *testing.T) {
	attr := Method("POST")
	if attr.Key != FieldMethod {
		t.Errorf("expected key %q, got %q", FieldMethod, attr.Key)
	}
	if attr.Value.String() != "POST" {
		t.Errorf("expected value %q, got %q", "POST", attr.Value.String())
	}
}

func TestPath(t *testing.T) {
	attr := Path("/api/v1/events")
	if attr.Key != FieldPath {
		t.Errorf("expected key %q, got %q", FieldPath, attr.Key)
	}
	if attr.Value.String() != "/api/v1/events" {
		t.Errorf("expected value %q, got %q", "/api/v1/events", attr.Value.String())
	}
}

func TestStatus(t *testing.T) {
	attr := Status(200)
	if attr.Key != FieldStatus {
		t.Errorf("expected key %q, got %q", FieldStatus, attr.Key)
	}
	if attr.Value.Int64() != 200 {
		t.Errorf("expected value %d, got %d", 200, attr.Value.Int64())
	}
}

func TestDuration(t *testing.T) {
	attr := Duration(1234)
	if attr.Key != FieldDuration {
		t.Errorf("expected key %q, got %q", FieldDuration, attr.Key)
	}
	if attr.Value.Int64() != 1234 {
		t.Errorf("expected value %d, got %d", 1234, attr.Value.Int64())
	}
}

func TestError(t *testing.T) {
	err := errors.New("something went wrong")
	attr := Error(err)
	if attr.Key != FieldError {
		t.Errorf("expected key %q, got %q", FieldError, attr.Key)
	}
	if attr.Value.String() != "something went wrong" {
		t.Errorf("expected value %q, got %q", "something went wrong", attr.Value.String())
	}
}

func TestTokenID(t *testing.T) {
	attr := TokenID("token-abc-123")
	if attr.Key != FieldTokenID {
		t.Errorf("expected key %q, got %q", FieldTokenID, attr.Key)
	}
	if attr.Value.String() != "token-abc-123" {
		t.Errorf("expected value %q, got %q", "token-abc-123", attr.Value.String())
	}
}

func TestEventID(t *testing.T) {
	attr := EventID("event-xyz-789")
	if attr.Key != FieldEventID {
		t.Errorf("expected key %q, got %q", FieldEventID, attr.Key)
	}
	if attr.Value.String() != "event-xyz-789" {
		t.Errorf("expected value %q, got %q", "event-xyz-789", attr.Value.String())
	}
}

func TestQuery(t *testing.T) {
	attr := Query("SELECT * FROM events")
	if attr.Key != FieldQuery {
		t.Errorf("expected key %q, got %q", FieldQuery, attr.Key)
	}
	if attr.Value.String() != "SELECT * FROM events" {
		t.Errorf("expected value %q, got %q", "SELECT * FROM events", attr.Value.String())
	}
}

func TestFieldConstants(t *testing.T) {
	// Verify all field constants are defined and non-empty
	fields := map[string]string{
		"FieldService":  FieldService,
		"FieldUserID":   FieldUserID,
		"FieldUsername": FieldUsername,
		"FieldIP":       FieldIP,
		"FieldMethod":   FieldMethod,
		"FieldPath":     FieldPath,
		"FieldStatus":   FieldStatus,
		"FieldDuration": FieldDuration,
		"FieldError":    FieldError,
		"FieldTokenID":  FieldTokenID,
		"FieldEventID":  FieldEventID,
		"FieldQuery":    FieldQuery,
	}

	for name, value := range fields {
		if value == "" {
			t.Errorf("%s constant is empty", name)
		}
	}
}

func TestFieldHelpers_ReturnsSlogAttr(t *testing.T) {
	// Verify all helper functions return slog.Attr type
	tests := []struct {
		name string
		attr slog.Attr
	}{
		{"Service", Service("test")},
		{"UserID", UserID("test")},
		{"Username", Username("test")},
		{"IP", IP("test")},
		{"Method", Method("test")},
		{"Path", Path("test")},
		{"Status", Status(200)},
		{"Duration", Duration(100)},
		{"Error", Error(errors.New("test"))},
		{"TokenID", TokenID("test")},
		{"EventID", EventID("test")},
		{"Query", Query("test")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If this compiles and runs, the types are correct
			_ = tt.attr.Key
			_ = tt.attr.Value
		})
	}
}
