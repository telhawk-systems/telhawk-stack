package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/common/messaging"
	natsclient "github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/repository"
)

// Handler processes incoming NATS messages for the respond service.
type Handler struct {
	client    *natsclient.Client
	repo      repository.Repository
	publisher *Publisher
	subs      []messaging.Subscription
}

// NewHandler creates a new NATS message handler.
func NewHandler(client *natsclient.Client, repo repository.Repository, publisher *Publisher) *Handler {
	return &Handler{
		client:    client,
		repo:      repo,
		publisher: publisher,
		subs:      make([]messaging.Subscription, 0),
	}
}

// Start begins listening for NATS messages.
func (h *Handler) Start(ctx context.Context) error {
	// Subscribe to correlation results from the search service
	sub, err := h.client.QueueSubscribe(
		messaging.SubjectSearchResultsCorrelate,
		messaging.QueueRespondWorkers,
		h.handleCorrelationResult,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to correlation results: %w", err)
	}
	h.subs = append(h.subs, sub)

	log.Printf("NATS handler started, subscribed to %s", messaging.SubjectSearchResultsCorrelate)
	return nil
}

// Stop gracefully stops the handler and unsubscribes from all subjects.
func (h *Handler) Stop() error {
	for _, sub := range h.subs {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Warning: failed to unsubscribe from %s: %v", sub.Subject(), err)
		}
	}
	h.subs = nil
	log.Println("NATS handler stopped")
	return nil
}

// handleCorrelationResult processes correlation results from the search service.
func (h *Handler) handleCorrelationResult(ctx context.Context, msg *messaging.Message) error {
	var result CorrelationJobResponse
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		log.Printf("Failed to unmarshal correlation result: %v", err)
		return err
	}

	if !result.Success {
		log.Printf("Correlation job %s failed: %s", result.JobID, result.Error)
		return nil
	}

	if !result.Triggered {
		log.Printf("Correlation job %s: no matches (took %dms)", result.JobID, result.TookMs)
		return nil
	}

	log.Printf("Correlation job %s triggered: %d matches (took %dms)",
		result.JobID, result.MatchCount, result.TookMs)

	// Get schema to build alert details
	schema, err := h.repo.GetSchemaByVersionID(ctx, result.SchemaVersionID)
	if err != nil {
		log.Printf("Failed to get schema %s: %v", result.SchemaVersionID, err)
		return err
	}

	// Create alerts for each match
	for _, match := range result.Matches {
		alertID, err := uuid.NewV7()
		if err != nil {
			log.Printf("Failed to generate alert ID: %v", err)
			continue
		}

		// Extract title and severity from schema view
		title := extractString(schema.View, "title", "Detection Alert")
		severity := extractString(schema.View, "severity", "medium")
		description := extractString(schema.View, "description", "")

		// Create and publish alert event
		// Note: In a full implementation, we would also persist the alert to the database
		// For now, we publish the event so other services can consume it
		event := &AlertCreatedEvent{
			AlertID:         alertID.String(),
			SchemaID:        result.SchemaID,
			SchemaVersionID: result.SchemaVersionID,
			Title:           title,
			Description:     description,
			Severity:        severity,
			TriggeredAt:     time.Now(),
			EventCount:      match.EventCount,
			AggregationKey:  match.AggregationKey,
		}

		if err := h.publisher.PublishAlertCreated(ctx, event); err != nil {
			log.Printf("Failed to publish alert created event: %v", err)
			// Continue processing other matches
		} else {
			log.Printf("Published alert %s for schema %s (matched %d events)",
				alertID.String(), result.SchemaID, match.EventCount)
		}
	}

	return nil
}

// extractString safely extracts a string value from a map.
func extractString(m map[string]interface{}, key, defaultVal string) string {
	if m == nil {
		return defaultVal
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}
