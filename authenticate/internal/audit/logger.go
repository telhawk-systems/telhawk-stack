package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
)

type Repository interface {
	LogAudit(ctx context.Context, entry *models.AuditLogEntry) error
}

type Logger struct {
	secretKey    []byte
	logs         []*models.AuditLog
	repo         Repository
	ingestClient *IngestClient
}

func NewLogger(secretKey string) *Logger {
	return &Logger{
		secretKey: []byte(secretKey),
		logs:      make([]*models.AuditLog, 0),
	}
}

func NewLoggerWithRepo(secretKey string, repo Repository) *Logger {
	return &Logger{
		secretKey: []byte(secretKey),
		logs:      make([]*models.AuditLog, 0),
		repo:      repo,
	}
}

func NewLoggerWithRepoAndIngest(secretKey string, repo Repository, ingestClient *IngestClient) *Logger {
	return &Logger{
		secretKey:    []byte(secretKey),
		logs:         make([]*models.AuditLog, 0),
		repo:         repo,
		ingestClient: ingestClient,
	}
}

func (l *Logger) Log(actorType, actorID, actorUsername, action, resource, resourceID, ipAddress, userAgent, result, reason string, metadata map[string]interface{}) *models.AuditLog {
	id, _ := uuid.NewV7()
	log := &models.AuditLog{
		ID:            id.String(),
		Timestamp:     time.Now().UTC(),
		ActorType:     actorType,
		ActorID:       actorID,
		ActorUsername: actorUsername,
		Action:        action,
		Resource:      resource,
		ResourceID:    resourceID,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Result:        result,
		Reason:        reason,
		Metadata:      metadata,
	}

	log.Signature = l.sign(log)
	l.logs = append(l.logs, log)

	// Always persist to PostgreSQL
	// Use Background context to ensure audit logs complete even if parent request is cancelled
	if l.repo != nil {
		entry := &models.AuditLogEntry{
			Timestamp:    log.Timestamp,
			ActorType:    actorType,
			ActorID:      actorID,
			ActorName:    actorUsername,
			Action:       action,
			ResourceType: resource,
			ResourceID:   resourceID,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			Result:       result,
			ErrorMessage: reason,
			Metadata:     metadata,
		}
		if err := l.repo.LogAudit(context.Background(), entry); err != nil {
			// Silently ignore audit persistence errors to not block auth operations
			_ = err
		}
	}

	// Forward security-relevant events to ingest pipeline
	// Skip high-frequency operational events to prevent noise and loops
	if l.ingestClient != nil && models.ShouldForwardToIngest(action) {
		go func() {
			// Async send to not block auth operations
			if err := l.ingestClient.ForwardEvent(
				actorType, actorID, actorUsername, action,
				resource, resourceID, ipAddress, userAgent,
				result, reason, metadata, log.Timestamp,
			); err != nil {
				slog.Warn("Failed to forward audit event to ingest", slog.String("error", err.Error()))
			}
		}()
	}

	return log
}

func (l *Logger) sign(log *models.AuditLog) string {
	data := []byte(log.ID + log.Timestamp.Format(time.RFC3339Nano) + log.ActorID + log.Action + log.Resource + log.Result)
	h := hmac.New(sha256.New, l.secretKey)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (l *Logger) Verify(log *models.AuditLog) bool {
	expectedSig := l.sign(log)
	return hmac.Equal([]byte(expectedSig), []byte(log.Signature))
}

func (l *Logger) LogJSON(log *models.AuditLog) ([]byte, error) {
	return json.Marshal(log)
}

func (l *Logger) GetLogs() []*models.AuditLog {
	return l.logs
}
