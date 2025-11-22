// Package scheduler provides periodic execution of detection rules.
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
	respondnats "github.com/telhawk-systems/telhawk-stack/respond/internal/nats"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/repository"
)

// Scheduler periodically evaluates detection schemas by requesting
// correlation jobs from the search service.
type Scheduler struct {
	repo      repository.Repository
	publisher *respondnats.Publisher
	interval  time.Duration
	stop      chan struct{}
	stopped   chan struct{}
}

// NewScheduler creates a new correlation scheduler.
func NewScheduler(repo repository.Repository, publisher *respondnats.Publisher, interval time.Duration) *Scheduler {
	return &Scheduler{
		repo:      repo,
		publisher: publisher,
		interval:  interval,
		stop:      make(chan struct{}),
		stopped:   make(chan struct{}),
	}
}

// Start begins the scheduler loop. This should be called in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	defer close(s.stopped)

	log.Printf("Correlation scheduler started (interval: %v)", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.runCorrelations(ctx)

	for {
		select {
		case <-ticker.C:
			s.runCorrelations(ctx)
		case <-s.stop:
			log.Println("Correlation scheduler stopped")
			return
		case <-ctx.Done():
			log.Println("Correlation scheduler context cancelled")
			return
		}
	}
}

// Stop signals the scheduler to stop and waits for it to finish.
func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.stopped
}

// runCorrelations fetches all active schemas and publishes correlation job requests.
func (s *Scheduler) runCorrelations(ctx context.Context) {
	// Get all active schemas
	schemas, _, err := s.repo.ListSchemas(ctx, &models.ListSchemasRequest{
		Page:            1,
		Limit:           1000,
		IncludeDisabled: false,
		IncludeHidden:   false,
	})
	if err != nil {
		log.Printf("Failed to list schemas for correlation: %v", err)
		return
	}

	if len(schemas) == 0 {
		log.Println("No active schemas for correlation")
		return
	}

	log.Printf("Running correlation for %d active schemas", len(schemas))

	for _, schema := range schemas {
		if err := s.runSchemaCorrelation(ctx, schema); err != nil {
			log.Printf("Failed to run correlation for schema %s: %v", schema.ID, err)
		}
	}
}

// runSchemaCorrelation publishes a correlation job request for a single schema.
func (s *Scheduler) runSchemaCorrelation(ctx context.Context, schema *models.DetectionSchema) error {
	// Extract correlation config from controller
	controller := schema.Controller
	if controller == nil {
		return nil // No controller defined
	}

	query := getStringFromMap(controller, "query")
	if query == "" {
		return nil // No query defined
	}

	// Get time window from controller (default 5 minutes)
	windowStr := getStringFromMap(controller, "time_window")
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		window = 5 * time.Minute
	}

	// Get threshold (default 1)
	threshold := 1
	if t, ok := controller["threshold"].(float64); ok {
		threshold = int(t)
	}

	// Get aggregation key if specified
	aggregationKey := getStringFromMap(controller, "aggregation_key")

	// Generate job ID
	jobID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	now := time.Now()
	req := &respondnats.CorrelationJobRequest{
		JobID:           jobID.String(),
		SchemaID:        schema.ID,
		SchemaVersionID: schema.VersionID,
		Query:           query,
		TimeRange: respondnats.TimeRange{
			From: now.Add(-window),
			To:   now,
		},
		AggregationKey: aggregationKey,
		Threshold:      threshold,
	}

	return s.publisher.RequestCorrelation(ctx, req)
}

// getStringFromMap safely extracts a string value from a map.
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
