package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/models"
)

type Logger struct {
	secretKey []byte
	logs      []*models.AuditLog
}

func NewLogger(secretKey string) *Logger {
	return &Logger{
		secretKey: []byte(secretKey),
		logs:      make([]*models.AuditLog, 0),
	}
}

func (l *Logger) Log(actorType, actorID, actorUsername, action, resource, resourceID, ipAddress, userAgent, result, reason string, metadata map[string]interface{}) *models.AuditLog {
	log := &models.AuditLog{
		ID:            uuid.New().String(),
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
