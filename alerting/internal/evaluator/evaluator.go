package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/telhawk-systems/telhawk-stack/alerting/internal/correlation"
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
	rulesClient         RulesClient
	storageClient       StorageClient
	lastEvalTime        time.Time
	stateManager        *correlation.StateManager
	queryExecutor       *correlation.QueryExecutor
	eventCountEval      *correlation.EventCountEvaluator
	valueCountEval      *correlation.ValueCountEvaluator
	temporalEval        *correlation.TemporalEvaluator
	temporalOrderedEval *correlation.TemporalOrderedEvaluator
	joinEval            *correlation.JoinEvaluator
}

// NewEvaluator creates a new evaluator instance
func NewEvaluator(rulesClient RulesClient, storageClient StorageClient, stateManager *correlation.StateManager, queryExecutor *correlation.QueryExecutor) *Evaluator {
	return &Evaluator{
		rulesClient:         rulesClient,
		storageClient:       storageClient,
		lastEvalTime:        time.Now().Add(-5 * time.Minute), // Start 5 minutes in the past
		stateManager:        stateManager,
		queryExecutor:       queryExecutor,
		eventCountEval:      correlation.NewEventCountEvaluator(queryExecutor, stateManager),
		valueCountEval:      correlation.NewValueCountEvaluator(queryExecutor, stateManager),
		temporalEval:        correlation.NewTemporalEvaluator(queryExecutor, stateManager),
		temporalOrderedEval: correlation.NewTemporalOrderedEvaluator(queryExecutor, stateManager),
		joinEval:            correlation.NewJoinEvaluator(queryExecutor, stateManager),
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
	// Check if this is a correlation rule
	correlationType, ok := schema.Model["correlation_type"].(string)
	if ok && correlationType != "" {
		return e.evaluateCorrelationRule(ctx, schema)
	}

	// Fall back to simple rule evaluation
	return e.evaluateSimpleRule(ctx, schema, events)
}

// evaluateCorrelationRule evaluates a correlation-based detection schema
func (e *Evaluator) evaluateCorrelationRule(ctx context.Context, schema *DetectionSchema) []*Alert {
	// Convert to correlation schema format
	corrSchema := &correlation.DetectionSchema{
		ID:         schema.ID,
		VersionID:  schema.VersionID,
		Model:      schema.Model,
		View:       schema.View,
		Controller: schema.Controller,
	}

	correlationType := correlation.CorrelationType(schema.Model["correlation_type"].(string))

	var alerts []*correlation.Alert
	var err error

	// Route to appropriate evaluator
	switch correlationType {
	case correlation.TypeEventCount:
		alerts, err = e.eventCountEval.Evaluate(ctx, corrSchema)
	case correlation.TypeValueCount:
		alerts, err = e.valueCountEval.Evaluate(ctx, corrSchema)
	case correlation.TypeTemporal:
		alerts, err = e.temporalEval.Evaluate(ctx, corrSchema)
	case correlation.TypeTemporalOrdered:
		alerts, err = e.temporalOrderedEval.Evaluate(ctx, corrSchema)
	case correlation.TypeJoin:
		alerts, err = e.joinEval.Evaluate(ctx, corrSchema)
	// TODO: Add remaining correlation types as they're implemented
	// case correlation.TypeBaselineDeviation:
	// 	alerts, err = e.baselineDeviationEval.Evaluate(ctx, corrSchema)
	// case correlation.TypeMissingEvent:
	// 	alerts, err = e.missingEventEval.Evaluate(ctx, corrSchema)
	default:
		log.Printf("Unsupported correlation type '%s' for schema %s", correlationType, schema.ID)
		return []*Alert{}
	}

	if err != nil {
		log.Printf("Error evaluating correlation rule %s: %v", schema.ID, err)
		return []*Alert{}
	}

	// Apply suppression if configured
	alerts = e.applySuppression(ctx, schema, alerts)

	// Convert correlation alerts to evaluator alerts
	return e.convertCorrelationAlerts(alerts)
}

// applySuppression applies suppression logic to alerts
func (e *Evaluator) applySuppression(ctx context.Context, schema *DetectionSchema, alerts []*correlation.Alert) []*correlation.Alert {
	// Check if suppression is configured in controller
	suppressionConfig, ok := schema.Controller["suppression"].(map[string]interface{})
	if !ok {
		return alerts // No suppression configured
	}

	enabled, _ := suppressionConfig["enabled"].(bool)
	if !enabled {
		return alerts // Suppression disabled
	}

	// Extract suppression parameters
	var params correlation.SuppressionParams
	data, err := json.Marshal(suppressionConfig)
	if err != nil {
		log.Printf("Failed to marshal suppression config: %v", err)
		return alerts
	}
	if err := json.Unmarshal(data, &params); err != nil {
		log.Printf("Failed to parse suppression params: %v", err)
		return alerts
	}

	if err := params.Validate(); err != nil {
		log.Printf("Invalid suppression params: %v", err)
		return alerts
	}

	// Parse suppression window
	window, err := time.ParseDuration(params.Window)
	if err != nil {
		log.Printf("Invalid suppression window: %v", err)
		return alerts
	}

	maxAlerts := params.MaxAlerts
	if maxAlerts == 0 {
		maxAlerts = 1 // Default to 1 alert per window
	}

	// Filter alerts through suppression
	unsuppressedAlerts := make([]*correlation.Alert, 0)
	for _, alert := range alerts {
		// Build suppression key from alert metadata
		suppressionKey := make(map[string]string)
		for _, keyField := range params.Key {
			if val, ok := alert.Metadata[keyField]; ok {
				suppressionKey[keyField] = fmt.Sprintf("%v", val)
			}
		}

		// Check if suppressed
		suppressed, err := e.stateManager.IsSuppressed(ctx, schema.ID, suppressionKey)
		if err != nil {
			log.Printf("Error checking suppression: %v", err)
			unsuppressedAlerts = append(unsuppressedAlerts, alert)
			continue
		}

		if suppressed {
			log.Printf("Alert suppressed for rule %s with key %v", schema.ID, suppressionKey)
			continue
		}

		// Record alert for suppression tracking
		if err := e.stateManager.RecordAlert(ctx, schema.ID, suppressionKey, window, maxAlerts); err != nil {
			log.Printf("Error recording alert for suppression: %v", err)
		}

		unsuppressedAlerts = append(unsuppressedAlerts, alert)
	}

	return unsuppressedAlerts
}

// evaluateSimpleRule evaluates a non-correlation detection schema
func (e *Evaluator) evaluateSimpleRule(ctx context.Context, schema *DetectionSchema, events []*Event) []*Alert {
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

// convertCorrelationAlerts converts correlation alerts to evaluator alerts
func (e *Evaluator) convertCorrelationAlerts(corrAlerts []*correlation.Alert) []*Alert {
	alerts := make([]*Alert, 0, len(corrAlerts))
	for _, ca := range corrAlerts {
		alert := &Alert{
			ID:                       fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			DetectionSchemaID:        ca.RuleID,
			DetectionSchemaVersionID: ca.RuleVersionID,
			Severity:                 ca.Severity,
			Title:                    ca.Title,
			Description:              ca.Description,
			MatchedEvents:            []string{}, // Correlation alerts may not have individual event IDs
			Metadata:                 ca.Metadata,
			Timestamp:                ca.Time,
		}
		alerts = append(alerts, alert)
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
