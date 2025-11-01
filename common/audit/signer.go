package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type EventSigner struct {
	secretKey []byte
}

func NewEventSigner(secretKey string) *EventSigner {
	return &EventSigner{
		secretKey: []byte(secretKey),
	}
}

func (s *EventSigner) Sign(eventID string, timestamp time.Time, sourceIP string, data []byte) string {
	payload := eventID + timestamp.Format(time.RFC3339Nano) + sourceIP + string(data)
	h := hmac.New(sha256.New, s.secretKey)
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *EventSigner) Verify(eventID string, timestamp time.Time, sourceIP string, data []byte, signature string) bool {
	expected := s.Sign(eventID, timestamp, sourceIP, data)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *EventSigner) SignIngestion(hecTokenID, sourceIP string, eventCount int, bytesReceived int64, timestamp time.Time) string {
	payload := uuid.New().String() + timestamp.Format(time.RFC3339Nano) + hecTokenID + sourceIP
	h := hmac.New(sha256.New, s.secretKey)
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
