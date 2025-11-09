package service

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/dlq"
	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/storage"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// Processor wraps the pipeline and captures basic telemetry.
type Processor struct {
	pipeline      *pipeline.Pipeline
	storageClient *storage.Client
	dlq           *dlq.Queue
	startedAt     time.Time
	processed     atomic.Uint64
	failed        atomic.Uint64
	stored        atomic.Uint64
	dlqWritten    atomic.Uint64
}

// NewProcessor creates a new Processor instance.
func NewProcessor(p *pipeline.Pipeline, storageClient *storage.Client, dlqQueue *dlq.Queue) *Processor {
	return &Processor{
		pipeline:      p,
		storageClient: storageClient,
		dlq:           dlqQueue,
		startedAt:     time.Now().UTC(),
	}
}

// Process runs the configured pipeline against the envelope.
func (p *Processor) Process(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	event, err := p.pipeline.Process(ctx, envelope)
	if err != nil {
		p.failed.Add(1)

		// Log detailed normalization/validation failure information
		payloadPreview := string(envelope.Payload)
		if len(payloadPreview) > 500 {
			payloadPreview = payloadPreview[:500] + "..."
		}
		log.Printf("ERROR: Normalization/validation failed for event [id=%s, source=%s, source_type=%s, format=%s]: %v\nPayload preview: %s",
			envelope.ID, envelope.Source, envelope.SourceType, envelope.Format, err, payloadPreview)

		// Write to DLQ for normalization failures
		if p.dlq != nil {
			if dlqErr := p.dlq.Write(ctx, envelope, err, "normalization_failed"); dlqErr != nil {
				log.Printf("ERROR: failed to write to DLQ: %v", dlqErr)
			} else {
				p.dlqWritten.Add(1)
			}
		}

		return nil, err
	}

	// Persist normalized event to storage
	if p.storageClient != nil {
		if err := p.storageClient.StoreEvent(ctx, event); err != nil {
			p.failed.Add(1)
			
			// Write to DLQ for storage failures
			if p.dlq != nil {
				if dlqErr := p.dlq.Write(ctx, envelope, err, "storage_failed"); dlqErr != nil {
					log.Printf("ERROR: failed to write to DLQ: %v", dlqErr)
				} else {
					p.dlqWritten.Add(1)
				}
			}
			
			log.Printf("ERROR: failed to persist event to storage: %v", err)
			return nil, err
		}
		p.stored.Add(1)
	}

	p.processed.Add(1)
	return event, nil
}

// Stats returns a snapshot of processor metrics.
type Stats struct {
	UptimeSeconds int64                  `json:"uptime_seconds"`
	Processed     uint64                 `json:"processed"`
	Failed        uint64                 `json:"failed"`
	Stored        uint64                 `json:"stored"`
	DLQWritten    uint64                 `json:"dlq_written"`
	DLQStats      map[string]interface{} `json:"dlq_stats,omitempty"`
}

// Health returns live status for health checks.
func (p *Processor) Health() Stats {
	stats := Stats{
		UptimeSeconds: int64(time.Since(p.startedAt).Seconds()),
		Processed:     p.processed.Load(),
		Failed:        p.failed.Load(),
		Stored:        p.stored.Load(),
		DLQWritten:    p.dlqWritten.Load(),
	}
	
	if p.dlq != nil {
		stats.DLQStats = p.dlq.Stats()
	}
	
	return stats
}

// DLQ returns the dead-letter queue instance.
func (p *Processor) DLQ() *dlq.Queue {
	return p.dlq
}

