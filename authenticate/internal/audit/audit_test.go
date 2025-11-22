package audit

import (
	"context"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
)

// mockRepository implements the Repository interface for testing
type mockRepository struct {
	logs         []*models.AuditLogEntry
	logAuditErr  error
	logAuditFunc func(*models.AuditLogEntry) error
}

func (m *mockRepository) LogAudit(ctx context.Context, entry *models.AuditLogEntry) error {
	if m.logAuditFunc != nil {
		return m.logAuditFunc(entry)
	}
	if m.logAuditErr != nil {
		return m.logAuditErr
	}
	m.logs = append(m.logs, entry)
	return nil
}

// ============================================================================
// Logger Tests
// ============================================================================

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-secret")

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	if string(logger.secretKey) != "test-secret" {
		t.Errorf("Expected secret key 'test-secret', got %s", string(logger.secretKey))
	}

	if logger.logs == nil {
		t.Error("Logger logs slice is nil")
	}

	if logger.repo != nil {
		t.Error("Logger repo should be nil")
	}
}

func TestNewLoggerWithRepo(t *testing.T) {
	repo := &mockRepository{}
	logger := NewLoggerWithRepo("test-secret", repo)

	if logger == nil {
		t.Fatal("NewLoggerWithRepo returned nil")
	}

	if logger.repo != repo {
		t.Error("Logger repo not set correctly")
	}
}

func TestNewLoggerWithRepoAndIngest(t *testing.T) {
	repo := &mockRepository{}
	ingestClient := &IngestClient{} // Just for testing, not functional
	logger := NewLoggerWithRepoAndIngest("test-secret", repo, ingestClient)

	if logger == nil {
		t.Fatal("NewLoggerWithRepoAndIngest returned nil")
	}

	if logger.repo != repo {
		t.Error("Logger repo not set correctly")
	}

	if logger.ingestClient != ingestClient {
		t.Error("Logger ingestClient not set correctly")
	}
}

func TestLogger_Log(t *testing.T) {
	logger := NewLogger("test-secret-key")

	metadata := map[string]interface{}{
		"ip":         "192.168.1.1",
		"user_agent": "Mozilla/5.0",
	}

	log := logger.Log(
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
	)

	if log == nil {
		t.Fatal("Log returned nil")
	}

	// Verify log fields
	if log.ActorType != "user" {
		t.Errorf("Expected ActorType 'user', got %s", log.ActorType)
	}

	if log.ActorID != "user-123" {
		t.Errorf("Expected ActorID 'user-123', got %s", log.ActorID)
	}

	if log.ActorUsername != "testuser" {
		t.Errorf("Expected ActorUsername 'testuser', got %s", log.ActorUsername)
	}

	if log.Action != "login" {
		t.Errorf("Expected Action 'login', got %s", log.Action)
	}

	if log.Result != "success" {
		t.Errorf("Expected Result 'success', got %s", log.Result)
	}

	if log.Signature == "" {
		t.Error("Log signature should not be empty")
	}

	// Verify log was added to logger
	logs := logger.GetLogs()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}

	if logs[0] != log {
		t.Error("Log not added to logger correctly")
	}
}

func TestLogger_LogWithRepository(t *testing.T) {
	repo := &mockRepository{logs: make([]*models.AuditLogEntry, 0)}
	logger := NewLoggerWithRepo("test-secret", repo)

	log := logger.Log(
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
		nil,
	)

	if log == nil {
		t.Fatal("Log returned nil")
	}

	// Verify log was persisted to repository
	if len(repo.logs) != 1 {
		t.Fatalf("Expected 1 log in repository, got %d", len(repo.logs))
	}

	entry := repo.logs[0]
	if entry.ActorID != "user-123" {
		t.Errorf("Expected ActorID 'user-123', got %s", entry.ActorID)
	}

	if entry.Action != "login" {
		t.Errorf("Expected Action 'login', got %s", entry.Action)
	}
}

func TestLogger_LogWithRepositoryError(t *testing.T) {
	// Logger should continue even if repository fails
	repo := &mockRepository{
		logAuditErr: context.DeadlineExceeded,
	}
	logger := NewLoggerWithRepo("test-secret", repo)

	// Should not panic even with repository error
	log := logger.Log(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"192.168.1.1",
		"",
		"success",
		"",
		nil,
	)

	if log == nil {
		t.Fatal("Log should still be created even if repository fails")
	}

	// Verify log was still added to in-memory logs
	logs := logger.GetLogs()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log in memory, got %d", len(logs))
	}
}

func TestLogger_Sign(t *testing.T) {
	logger := NewLogger("test-secret-key")

	now := time.Now()
	log := &models.AuditLog{
		ID:        "log-123",
		Timestamp: now,
		ActorID:   "user-123",
		Action:    "login",
		Resource:  "user",
		Result:    "success",
	}

	signature := logger.sign(log)

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	// Signature should be deterministic - same log should produce same signature
	signature2 := logger.sign(log)
	if signature != signature2 {
		t.Error("Same log should produce same signature")
	}

	// Different log should produce different signature
	log.ActorID = "user-456"
	signature3 := logger.sign(log)
	if signature == signature3 {
		t.Error("Different log should produce different signature")
	}
}

func TestLogger_Verify(t *testing.T) {
	logger := NewLogger("test-secret-key")

	log := logger.Log(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"192.168.1.1",
		"",
		"success",
		"",
		nil,
	)

	// Original log should verify
	if !logger.Verify(log) {
		t.Error("Valid log should verify successfully")
	}

	// Tampered log should fail verification
	log.ActorID = "user-456"
	if logger.Verify(log) {
		t.Error("Tampered log should fail verification")
	}
}

func TestLogger_VerifyWithDifferentSecret(t *testing.T) {
	logger1 := NewLogger("secret-1")
	logger2 := NewLogger("secret-2")

	log := logger1.Log(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"192.168.1.1",
		"",
		"success",
		"",
		nil,
	)

	// Logger with same secret should verify
	if !logger1.Verify(log) {
		t.Error("Logger with same secret should verify")
	}

	// Logger with different secret should fail
	if logger2.Verify(log) {
		t.Error("Logger with different secret should not verify")
	}
}

func TestLogger_LogJSON(t *testing.T) {
	logger := NewLogger("test-secret")

	log := logger.Log(
		"user",
		"user-123",
		"testuser",
		"login",
		"user",
		"user-123",
		"192.168.1.1",
		"",
		"success",
		"",
		nil,
	)

	jsonBytes, err := logger.LogJSON(log)
	if err != nil {
		t.Fatalf("LogJSON failed: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("LogJSON returned empty bytes")
	}

	// Should be valid JSON with expected fields
	jsonStr := string(jsonBytes)
	if jsonStr == "" {
		t.Error("JSON string is empty")
	}

	// Basic JSON validation - should contain expected fields
	expectedFields := []string{
		"\"id\":",
		"\"timestamp\":",
		"\"actor_type\":\"user\"",
		"\"actor_id\":\"user-123\"",
		"\"action\":\"login\"",
		"\"signature\":",
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON missing expected field: %s", field)
		}
	}
}

func TestLogger_GetLogs(t *testing.T) {
	logger := NewLogger("test-secret")

	// Initially empty
	logs := logger.GetLogs()
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs initially, got %d", len(logs))
	}

	// Add logs
	logger.Log("user", "user-1", "user1", "login", "user", "user-1", "", "", "success", "", nil)
	logger.Log("user", "user-2", "user2", "logout", "user", "user-2", "", "", "success", "", nil)
	logger.Log("user", "user-3", "user3", "login", "user", "user-3", "", "", "failure", "invalid password", nil)

	logs = logger.GetLogs()
	if len(logs) != 3 {
		t.Errorf("Expected 3 logs, got %d", len(logs))
	}

	// Verify logs are in order
	if logs[0].ActorID != "user-1" {
		t.Errorf("Expected first log ActorID 'user-1', got %s", logs[0].ActorID)
	}

	if logs[1].ActorID != "user-2" {
		t.Errorf("Expected second log ActorID 'user-2', got %s", logs[1].ActorID)
	}

	if logs[2].ActorID != "user-3" {
		t.Errorf("Expected third log ActorID 'user-3', got %s", logs[2].ActorID)
	}
}

func TestLogger_SignatureConsistency(t *testing.T) {
	// Test that signatures remain consistent across logger instances with same secret
	secret := "consistent-secret"

	logger1 := NewLogger(secret)
	logger2 := NewLogger(secret)

	now := time.Now().UTC()
	log := &models.AuditLog{
		ID:        "log-123",
		Timestamp: now,
		ActorID:   "user-123",
		Action:    "login",
		Resource:  "user",
		Result:    "success",
	}

	sig1 := logger1.sign(log)
	sig2 := logger2.sign(log)

	if sig1 != sig2 {
		t.Error("Signatures should be consistent across logger instances with same secret")
	}
}

func TestLogger_MultipleLogs(t *testing.T) {
	logger := NewLogger("test-secret")

	// Add multiple logs
	for i := 0; i < 10; i++ {
		logger.Log(
			"user",
			"user-123",
			"testuser",
			"login",
			"user",
			"user-123",
			"192.168.1.1",
			"",
			"success",
			"",
			nil,
		)
	}

	logs := logger.GetLogs()
	if len(logs) != 10 {
		t.Errorf("Expected 10 logs, got %d", len(logs))
	}

	// Each log should have unique ID
	ids := make(map[string]bool)
	for _, log := range logs {
		if ids[log.ID] {
			t.Errorf("Duplicate log ID found: %s", log.ID)
		}
		ids[log.ID] = true
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
