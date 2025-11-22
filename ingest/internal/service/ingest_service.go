package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/common/ocsf"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/ack"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/authclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/dlq"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/metrics"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/storageclient"
)

type IngestService struct {
	stats         models.IngestionStats
	statsMutex    sync.RWMutex
	eventQueue    chan *models.Event
	stopChan      chan struct{}
	pipeline      *pipeline.Pipeline
	dlq           *dlq.Queue
	storageClient StorageClient
	authClient    AuthClient
	ackManager    *ack.Manager
	queueCapacity int
}

type StorageClient interface {
	Ingest(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error)
}

type AuthClient interface {
	ValidateHECToken(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error)
}

func NewIngestService(pipeline *pipeline.Pipeline, dlq *dlq.Queue, storageClient StorageClient, authClient AuthClient) *IngestService {
	queueCap := 10000
	s := &IngestService{
		eventQueue:    make(chan *models.Event, queueCap),
		stopChan:      make(chan struct{}),
		queueCapacity: queueCap,
		stats: models.IngestionStats{
			LastEvent: time.Now(),
		},
		pipeline:      pipeline,
		dlq:           dlq,
		storageClient: storageClient,
		authClient:    authClient,
	}

	// Initialize metrics
	metrics.QueueCapacity.Set(float64(queueCap))

	// Start event processor
	go s.processEvents()

	return s
}

// SetAckManager configures ack management for the service
func (s *IngestService) SetAckManager(manager *ack.Manager) {
	s.ackManager = manager
}

func (s *IngestService) IngestEvent(event *models.HECEvent, sourceIP string, tokenInfo *TokenInfo) (string, error) {
	// Determine source_type with fallback to default
	sourceType := event.SourceType
	if sourceType == "" {
		sourceType = s.determineSourceType(event.Event)
	}

	// Extract token details
	var hecTokenID, tenantID string
	if tokenInfo != nil {
		hecTokenID = tokenInfo.TokenID
		tenantID = tokenInfo.TenantID
	}

	// Convert HEC event to internal event
	internalEvent := &models.Event{
		ID:         uuid.New().String(),
		Timestamp:  s.parseTime(event.Time),
		Host:       event.Host,
		Source:     event.Source,
		SourceType: sourceType,
		SourceIP:   sourceIP,
		Index:      s.getIndex(event.Index),
		Event:      event.Event,
		Fields:     event.Fields,
		HECTokenID: hecTokenID,
		TenantID:   tenantID,
	}

	// Serialize raw event
	raw, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	internalEvent.Raw = raw

	// Sign event for nonrepudiation
	internalEvent.Signature = s.signEvent(internalEvent)

	// Create ack if manager is configured (before queueing)
	var ackID string
	if s.ackManager != nil {
		ackID = s.ackManager.Create([]string{internalEvent.ID})
		internalEvent.AckID = ackID
	}

	// Queue for processing
	select {
	case s.eventQueue <- internalEvent:
		s.updateStats(len(raw), true)
		metrics.EventsTotal.WithLabelValues("event", "accepted").Inc()
		metrics.EventBytesTotal.Add(float64(len(raw)))
		metrics.QueueDepth.Set(float64(len(s.eventQueue)))
		return ackID, nil
	default:
		// If queue is full, fail the ack
		if s.ackManager != nil && ackID != "" {
			s.ackManager.Fail(ackID)
		}
		s.updateStats(0, false)
		metrics.EventsTotal.WithLabelValues("event", "queue_full").Inc()
		return "", fmt.Errorf("event queue full")
	}
}

func (s *IngestService) IngestRaw(data []byte, sourceIP string, tokenInfo *TokenInfo, source, sourceType, host string) (string, error) {
	// Extract token details
	var hecTokenID, tenantID string
	if tokenInfo != nil {
		hecTokenID = tokenInfo.TokenID
		tenantID = tokenInfo.TenantID
	}

	event := &models.Event{
		ID:         uuid.New().String(),
		Timestamp:  time.Now(),
		Source:     source,
		SourceType: sourceType,
		Host:       host,
		SourceIP:   sourceIP,
		Index:      "main",
		Event:      string(data),
		Raw:        data,
		HECTokenID: hecTokenID,
		TenantID:   tenantID,
	}

	event.Signature = s.signEvent(event)

	// Create ack if manager is configured (before queueing)
	var ackID string
	if s.ackManager != nil {
		ackID = s.ackManager.Create([]string{event.ID})
		event.AckID = ackID
	}

	select {
	case s.eventQueue <- event:
		s.updateStats(len(data), true)
		metrics.EventsTotal.WithLabelValues("raw", "accepted").Inc()
		metrics.EventBytesTotal.Add(float64(len(data)))
		metrics.QueueDepth.Set(float64(len(s.eventQueue)))
		return ackID, nil
	default:
		// If queue is full, fail the ack
		if s.ackManager != nil && ackID != "" {
			s.ackManager.Fail(ackID)
		}
		s.updateStats(0, false)
		metrics.EventsTotal.WithLabelValues("raw", "queue_full").Inc()
		return "", fmt.Errorf("event queue full")
	}
}

func (s *IngestService) processEvents() {
	for {
		select {
		case event := <-s.eventQueue:
			if event == nil {
				continue
			}
			metrics.QueueDepth.Set(float64(len(s.eventQueue)))
			log.Printf("Processing event: id=%s source=%s", event.ID, event.Source)

			// Normalize event via Core service
			startTime := time.Now()
			normalizedEvent, err := s.normalizeEvent(event)
			metrics.NormalizationDuration.Observe(time.Since(startTime).Seconds())

			if err != nil {
				log.Printf("failed to normalize event %s: %v", event.ID, err)
				metrics.NormalizationErrors.Inc()
				if s.ackManager != nil && event.AckID != "" {
					s.ackManager.Fail(event.AckID)
				}
				continue
			}

			// Forward to Storage service
			startTime = time.Now()
			err = s.forwardToStorage(normalizedEvent)
			metrics.StorageDuration.Observe(time.Since(startTime).Seconds())

			if err != nil {
				log.Printf("failed to forward event %s to storage: %v", event.ID, err)
				metrics.StorageErrors.Inc()
				if s.ackManager != nil && event.AckID != "" {
					s.ackManager.Fail(event.AckID)
				}
				continue
			}

			log.Printf("event %s successfully stored", event.ID)

			// Complete ack if manager is configured
			if s.ackManager != nil && event.AckID != "" {
				s.ackManager.Complete(event.AckID)
			}

		case <-s.stopChan:
			log.Println("Stopping event processor")
			return
		}
	}
}

func (s *IngestService) normalizeEvent(event *models.Event) (map[string]interface{}, error) {
	if s.pipeline == nil {
		log.Printf("normalization pipeline not configured; skipping normalization for event %s", event.ID)
		// Return basic event structure without normalization
		return s.eventToMap(event), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create raw event envelope for pipeline
	envelope := &models.RawEventEnvelope{
		ID:         event.ID,
		Source:     event.Source,
		SourceType: event.SourceType,
		Format:     "json",
		Payload:    event.Raw, // Raw JSON bytes, not base64 encoded
		Attributes: map[string]string{
			"host":      event.Host,
			"index":     event.Index,
			"source_ip": event.SourceIP,
		},
		ReceivedAt: event.Timestamp,
	}

	// Process through normalization pipeline
	normalizedEvent, err := s.pipeline.Process(ctx, envelope)
	if err != nil {
		// Write to DLQ if available
		if s.dlq != nil {
			if dlqErr := s.dlq.Write(ctx, envelope, err, "normalization_failed"); dlqErr != nil {
				log.Printf("failed to write to DLQ for event %s: %v", event.ID, dlqErr)
			}
		}
		return nil, fmt.Errorf("normalization failed: %w", err)
	}

	// Convert OCSF event to map for storage
	eventMap, err := s.ocsfEventToMap(normalizedEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to convert OCSF event: %w", err)
	}

	// Add tenant_id for data isolation (critical for multi-tenant)
	if event.TenantID != "" {
		eventMap["tenant_id"] = event.TenantID
	}

	log.Printf("event %s normalized via pipeline", event.ID)
	return eventMap, nil
}

// ocsfEventToMap converts an OCSF event to a map for storage
func (s *IngestService) ocsfEventToMap(event *ocsf.Event) (map[string]interface{}, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *IngestService) forwardToStorage(event map[string]interface{}) error {
	if s.storageClient == nil {
		log.Println("storage client not configured; skipping storage")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events := []map[string]interface{}{event}
	resp, err := s.storageClient.Ingest(ctx, events)
	if err != nil {
		return fmt.Errorf("storage ingest failed: %w", err)
	}

	if resp.Failed > 0 {
		return fmt.Errorf("storage reported %d failures: %v", resp.Failed, resp.Errors)
	}

	return nil
}

func (s *IngestService) eventToMap(event *models.Event) map[string]interface{} {
	return map[string]interface{}{
		"id":           event.ID,
		"timestamp":    event.Timestamp,
		"host":         event.Host,
		"source":       event.Source,
		"sourcetype":   event.SourceType,
		"source_ip":    event.SourceIP,
		"index":        event.Index,
		"event":        event.Event,
		"fields":       event.Fields,
		"hec_token_id": event.HECTokenID,
		"tenant_id":    event.TenantID,
		"signature":    event.Signature,
	}
}

// TokenInfo contains validated HEC token information
type TokenInfo struct {
	TokenID  string
	UserID   string
	TenantID string
}

func (s *IngestService) ValidateHECToken(ctx context.Context, token string) (*TokenInfo, error) {
	if s.authClient == nil {
		log.Println("auth client not configured; skipping token validation")
		// Return empty token info when no auth client (dev mode)
		return &TokenInfo{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := s.authClient.ValidateHECToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !resp.Valid {
		return nil, fmt.Errorf("invalid or expired HEC token")
	}

	log.Printf("HEC token validated: token_id=%s user_id=%s tenant_id=%s", resp.TokenID, resp.UserID, resp.TenantID)
	return &TokenInfo{
		TokenID:  resp.TokenID,
		UserID:   resp.UserID,
		TenantID: resp.TenantID,
	}, nil
}

func (s *IngestService) parseTime(t *float64) time.Time {
	if t == nil {
		return time.Now()
	}

	// Splunk HEC time format: seconds.milliseconds since epoch
	sec := int64(*t)
	nsec := int64((*t - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}

func (s *IngestService) getIndex(index string) string {
	if index == "" {
		return "main"
	}
	return index
}

func (s *IngestService) signEvent(event *models.Event) string {
	// Placeholder signature
	// In production, use EventSigner from common/audit
	return uuid.New().String()
}

func (s *IngestService) updateStats(bytes int, success bool) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	s.stats.TotalEvents++
	s.stats.TotalBytes += int64(bytes)
	s.stats.LastEvent = time.Now()

	if success {
		s.stats.SuccessfulEvents++
	} else {
		s.stats.FailedEvents++
	}
}

func (s *IngestService) GetStats() models.IngestionStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()
	return s.stats
}

func (s *IngestService) QueryAcks(ackIDs []string) map[string]bool {
	if s.ackManager == nil {
		return make(map[string]bool)
	}
	return s.ackManager.Query(ackIDs)
}

// determineSourceType determines OCSF source type based on event content
// This provides a fallback when source_type is not explicitly provided in the HEC event
func (s *IngestService) determineSourceType(event interface{}) string {
	// Try to extract class_uid if event is a map
	eventMap, ok := event.(map[string]interface{})
	if !ok {
		return "ocsf:generic"
	}

	classUID, ok := eventMap["class_uid"]
	if !ok {
		return "ocsf:generic"
	}

	// Convert to int (handle both float64 from JSON and int)
	var uid int
	switch v := classUID.(type) {
	case float64:
		uid = int(v)
	case int:
		uid = v
	default:
		return "ocsf:generic"
	}

	// Map OCSF class_uid to appropriate source_type
	// Based on OCSF 1.1.0 class definitions
	switch uid {
	case 3001, 3002, 3003, 3004, 3005, 3006:
		return "ocsf:iam" // Identity & Access Management (3000s)
	case 4001, 4002, 4003, 4004, 4005, 4006, 4007, 4008, 4009, 4010, 4011, 4012:
		return "ocsf:network" // Network Activity (4000s)
	case 1001, 1002, 1003, 1004, 1005, 1006, 1007:
		return "ocsf:system" // System Activity (1000s)
	case 2001, 2002, 2003, 2004:
		return "ocsf:findings" // Findings (2000s)
	case 5001, 5002, 5003:
		return "ocsf:discovery" // Discovery (5000s)
	case 6001, 6002, 6003, 6004:
		return "ocsf:application" // Application Activity (6000s)
	default:
		return "ocsf:generic"
	}
}

func (s *IngestService) Stop() {
	if s.ackManager != nil {
		s.ackManager.Close()
	}
	close(s.stopChan)
	close(s.eventQueue)
}
