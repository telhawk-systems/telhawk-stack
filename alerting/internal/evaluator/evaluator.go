package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// DetectionSchema represents a detection rule from the Rules service
type DetectionSchema struct {
	ID         string                 `json:"id"`
	VersionID  string                 `json:"version_id"`
	Version    int                    `json:"version"`
	Model      map[string]interface{} `json:"model"`
	View       map[string]interface{} `json:"view"`
	Controller map[string]interface{} `json:"controller"`
	CreatedAt  time.Time              `json:"created_at"`
	Disabled   bool                   `json:"disabled"`
}

// Event represents an OCSF event from OpenSearch
type Event struct {
	ID        string                 `json:"_id"`
	Index     string                 `json:"_index"`
	Source    map[string]interface{} `json:"_source"`
	Timestamp time.Time              `json:"@timestamp"`
}

// Alert represents a detection finding
type Alert struct {
	ID                       string                 `json:"id"`
	DetectionSchemaID        string                 `json:"detection_schema_id"`
	DetectionSchemaVersionID string                 `json:"detection_schema_version_id"`
	Severity                 string                 `json:"severity"`
	Title                    string                 `json:"title"`
	Description              string                 `json:"description"`
	MatchedEvents            []string               `json:"matched_events"` // Event IDs
	Metadata                 map[string]interface{} `json:"metadata"`
	Timestamp                time.Time              `json:"@timestamp"`
}

// RulesClient defines the interface for fetching detection schemas
type RulesClient interface {
	ListSchemas(ctx context.Context) ([]*DetectionSchema, error)
}

// StorageClient defines the interface for OpenSearch operations
type StorageClient interface {
	FetchEvents(ctx context.Context, since time.Time) ([]*Event, error)
	StoreAlert(ctx context.Context, alert *Alert) error
}

// Evaluator handles event evaluation against detection schemas
type Evaluator struct {
	rulesClient   RulesClient
	storageClient StorageClient
	lastEvalTime  time.Time
}

// NewEvaluator creates a new evaluator instance
func NewEvaluator(rulesClient RulesClient, storageClient StorageClient) *Evaluator {
	return &Evaluator{
		rulesClient:   rulesClient,
		storageClient: storageClient,
		lastEvalTime:  time.Now().Add(-5 * time.Minute), // Start 5 minutes in the past
	}
}

// Run starts the evaluation loop
func (e *Evaluator) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Evaluation engine started (interval: %s)", interval)

	// Run immediately on start
	e.evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Evaluation engine stopped")
			return
		case <-ticker.C:
			e.evaluate(ctx)
		}
	}
}

// evaluate performs a single evaluation cycle
func (e *Evaluator) evaluate(ctx context.Context) {
	now := time.Now()
	log.Printf("Starting evaluation cycle (since: %s)", e.lastEvalTime.Format(time.RFC3339))

	// Fetch new events since last evaluation
	events, err := e.storageClient.FetchEvents(ctx, e.lastEvalTime)
	if err != nil {
		log.Printf("Error fetching events: %v", err)
		return
	}

	if len(events) == 0 {
		log.Printf("No new events to evaluate")
		e.lastEvalTime = now
		return
	}

	log.Printf("Fetched %d events for evaluation", len(events))

	// Fetch active detection schemas
	schemas, err := e.rulesClient.ListSchemas(ctx)
	if err != nil {
		log.Printf("Error fetching detection schemas: %v", err)
		return
	}

	log.Printf("Evaluating against %d detection schemas", len(schemas))

	// Evaluate events against schemas
	alertCount := 0
	for _, schema := range schemas {
		if schema.Disabled {
			continue
		}

		alerts := e.evaluateSchema(ctx, schema, events)
		for _, alert := range alerts {
			if err := e.storageClient.StoreAlert(ctx, alert); err != nil {
				log.Printf("Error storing alert: %v", err)
			} else {
				alertCount++
			}
		}
	}

	log.Printf("Evaluation cycle complete: %d alerts generated from %d events", alertCount, len(events))
	e.lastEvalTime = now
}

// evaluateSchema evaluates events against a single detection schema
func (e *Evaluator) evaluateSchema(ctx context.Context, schema *DetectionSchema, events []*Event) []*Alert {
	alerts := []*Alert{}

	// Extract controller logic
	controller, ok := schema.Controller["detection"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid controller format for schema %s", schema.ID)
		return alerts
	}

	// Extract model (aggregation settings)
	model := schema.Model

	// Extract view (presentation settings)
	view := schema.View

	// Get title and description from view
	title, _ := view["title"].(string)
	description, _ := view["description"].(string)
	severity, _ := view["severity"].(string)
	if severity == "" {
		severity = "medium"
	}

	// Simple pattern matching evaluation
	// In production, this would use a proper query engine
	for _, event := range events {
		if e.matchesController(controller, event.Source) {
			alert := &Alert{
				ID:                       fmt.Sprintf("alert-%d", time.Now().UnixNano()),
				DetectionSchemaID:        schema.ID,
				DetectionSchemaVersionID: schema.VersionID,
				Severity:                 severity,
				Title:                    title,
				Description:              description,
				MatchedEvents:            []string{event.ID},
				Metadata: map[string]interface{}{
					"event_index":    event.Index,
					"schema_version": schema.Version,
					"model":          model,
				},
				Timestamp: time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// matchesController performs simple pattern matching
// In production, this would be a full query evaluation engine
func (e *Evaluator) matchesController(controller map[string]interface{}, eventData map[string]interface{}) bool {
	// Simple field matching example
	// Real implementation would parse the query language in controller["query"]

	query, ok := controller["query"].(string)
	if !ok {
		return false
	}

	// For now, just do a simple string match on serialized event
	eventJSON, _ := json.Marshal(eventData)
	// This is a placeholder - real implementation would parse and execute the query
	_ = eventJSON
	_ = query

	// TODO: Implement proper query language evaluation
	// For now, return false to avoid generating spurious alerts
	return false
}
