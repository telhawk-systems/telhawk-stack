package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/authclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/coreclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/storageclient"
)

type IngestService struct {
	stats         models.IngestionStats
	statsMutex    sync.RWMutex
	eventQueue    chan *models.Event
	stopChan      chan struct{}
	coreClient    NormalizationClient
	storageClient StorageClient
	authClient    AuthClient
}

type NormalizationClient interface {
	Normalize(ctx context.Context, event *models.Event) (*coreclient.NormalizationResult, error)
}

type StorageClient interface {
	Ingest(ctx context.Context, events []map[string]interface{}) (*storageclient.IngestResponse, error)
}

type AuthClient interface {
	ValidateHECToken(ctx context.Context, token string) (*authclient.ValidateHECTokenResponse, error)
}

func NewIngestService(coreClient NormalizationClient, storageClient StorageClient, authClient AuthClient) *IngestService {
	s := &IngestService{
		eventQueue: make(chan *models.Event, 10000),
		stopChan:   make(chan struct{}),
		stats: models.IngestionStats{
			LastEvent: time.Now(),
		},
		coreClient:    coreClient,
		storageClient: storageClient,
		authClient:    authClient,
	}

	// Start event processor
	go s.processEvents()

	return s
}

func (s *IngestService) IngestEvent(event *models.HECEvent, sourceIP, hecTokenID string) error {
	// Convert HEC event to internal event
	internalEvent := &models.Event{
		ID:         uuid.New().String(),
		Timestamp:  s.parseTime(event.Time),
		Host:       event.Host,
		Source:     event.Source,
		SourceType: event.SourceType,
		SourceIP:   sourceIP,
		Index:      s.getIndex(event.Index),
		Event:      event.Event,
		Fields:     event.Fields,
		HECTokenID: hecTokenID,
	}

	// Serialize raw event
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	internalEvent.Raw = raw

	// Sign event for nonrepudiation
	internalEvent.Signature = s.signEvent(internalEvent)

	// Queue for processing
	select {
	case s.eventQueue <- internalEvent:
		s.updateStats(len(raw), true)
		return nil
	default:
		s.updateStats(0, false)
		return fmt.Errorf("event queue full")
	}
}

func (s *IngestService) IngestRaw(data []byte, sourceIP, hecTokenID, source, sourceType, host string) error {
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
	}

	event.Signature = s.signEvent(event)

	select {
	case s.eventQueue <- event:
		s.updateStats(len(data), true)
		return nil
	default:
		s.updateStats(0, false)
		return fmt.Errorf("event queue full")
	}
}

func (s *IngestService) processEvents() {
	for {
		select {
		case event := <-s.eventQueue:
			log.Printf("Processing event: id=%s source=%s", event.ID, event.Source)
			
			// Normalize event via Core service
			normalizedEvent, err := s.normalizeEvent(event)
			if err != nil {
				log.Printf("failed to normalize event %s: %v", event.ID, err)
				continue
			}
			
			// Forward to Storage service
			if err := s.forwardToStorage(normalizedEvent); err != nil {
				log.Printf("failed to forward event %s to storage: %v", event.ID, err)
				continue
			}
			
			log.Printf("event %s successfully stored", event.ID)

		case <-s.stopChan:
			log.Println("Stopping event processor")
			return
		}
	}
}

func (s *IngestService) normalizeEvent(event *models.Event) (map[string]interface{}, error) {
	if s.coreClient == nil {
		log.Printf("core client not configured; skipping normalization for event %s", event.ID)
		// Return basic event structure without normalization
		return s.eventToMap(event), nil
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := s.coreClient.Normalize(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("normalization failed: %w", err)
	}
	
	// Parse normalized event
	var normalizedEvent map[string]interface{}
	if err := json.Unmarshal(result.Event, &normalizedEvent); err != nil {
		return nil, fmt.Errorf("failed to parse normalized event: %w", err)
	}
	
	log.Printf("event %s normalized via core service", event.ID)
	return normalizedEvent, nil
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
		"id":          event.ID,
		"timestamp":   event.Timestamp,
		"host":        event.Host,
		"source":      event.Source,
		"sourcetype":  event.SourceType,
		"source_ip":   event.SourceIP,
		"index":       event.Index,
		"event":       event.Event,
		"fields":      event.Fields,
		"hec_token_id": event.HECTokenID,
		"signature":   event.Signature,
	}
}

func (s *IngestService) ValidateHECToken(token string) error {
	if s.authClient == nil {
		log.Println("auth client not configured; skipping token validation")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := s.authClient.ValidateHECToken(ctx, token)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	if !resp.Valid {
		return fmt.Errorf("invalid or expired HEC token")
	}

	log.Printf("HEC token validated: token_id=%s user_id=%s", resp.TokenID, resp.UserID)
	return nil
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

func (s *IngestService) Stop() {
	close(s.stopChan)
	close(s.eventQueue)
}
