package service

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/storage"
	"github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf"
)

// Processor wraps the pipeline and captures basic telemetry.
type Processor struct {
	pipeline      *pipeline.Pipeline
	storageClient *storage.Client
	startedAt     time.Time
	processed     atomic.Uint64
	failed        atomic.Uint64
}

// NewProcessor creates a new Processor instance.
func NewProcessor(p *pipeline.Pipeline, storageClient *storage.Client) *Processor {
	return &Processor{
		pipeline:      p,
		storageClient: storageClient,
		startedAt:     time.Now().UTC(),
	}
}

// Process runs the configured pipeline against the envelope.
func (p *Processor) Process(ctx context.Context, envelope *model.RawEventEnvelope) (*ocsf.Event, error) {
	event, err := p.pipeline.Process(ctx, envelope)
	if err != nil {
		p.failed.Add(1)
		return nil, err
	}

	if p.storageClient != nil {
		if err := p.storageClient.StoreEvent(ctx, event); err != nil {
			log.Printf("failed to persist event to storage: %v", err)
		}
	}

	p.processed.Add(1)
	return event, nil
}

// Stats returns a snapshot of processor metrics.
type Stats struct {
	UptimeSeconds int64  `json:"uptime_seconds"`
	Processed     uint64 `json:"processed"`
	Failed        uint64 `json:"failed"`
}

// Health returns live status for health checks.
func (p *Processor) Health() Stats {
	return Stats{
		UptimeSeconds: int64(time.Since(p.startedAt).Seconds()),
		Processed:     p.processed.Load(),
		Failed:        p.failed.Load(),
	}
}
