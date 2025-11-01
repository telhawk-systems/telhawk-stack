package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/coreclient"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
)

type IngestService struct {
	stats      models.IngestionStats
	statsMutex sync.RWMutex
	eventQueue chan *models.Event
	stopChan   chan struct{}
	coreClient NormalizationClient
}

type NormalizationClient interface {
	Normalize(ctx context.Context, event *models.Event) (*coreclient.NormalizationResult, error)
}

func NewIngestService(coreClient NormalizationClient) *IngestService {
	s := &IngestService{
		eventQueue: make(chan *models.Event, 10000),
		stopChan:   make(chan struct{}),
		stats: models.IngestionStats{
			LastEvent: time.Now(),
		},
		coreClient: coreClient,
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
			s.normalizeEvent(event)
			// TODO: Forward to storage service
			// TODO: Handle acks if enabled

		case <-s.stopChan:
			log.Println("Stopping event processor")
			return
		}
	}
}

func (s *IngestService) normalizeEvent(event *models.Event) {
	if s.coreClient == nil {
		log.Printf("core client not configured; skipping normalization for event %s", event.ID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.coreClient.Normalize(ctx, event); err != nil {
		log.Printf("failed to normalize event %s: %v", event.ID, err)
		return
	}
	log.Printf("event %s normalized via core service", event.ID)
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
