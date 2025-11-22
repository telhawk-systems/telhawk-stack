// Package nats provides NATS messaging integration for the web backend.
package nats

import (
	"context"
	"encoding/json"
	"log"

	"github.com/telhawk-systems/telhawk-stack/common/messaging"
	"github.com/telhawk-systems/telhawk-stack/web/backend/internal/handlers"
)

// SearchResultMessage is the message format received from the search service.
type SearchResultMessage struct {
	JobID     string                   `json:"job_id"`
	Success   bool                     `json:"success"`
	Error     string                   `json:"error,omitempty"`
	TotalHits int64                    `json:"total_hits"`
	Events    []map[string]interface{} `json:"events"`
	TookMs    int64                    `json:"took_ms"`
}

// ResultSubscriber subscribes to search results and updates the async query handler.
type ResultSubscriber struct {
	client       messaging.Client
	queryHandler *handlers.AsyncQueryHandler
	sub          messaging.Subscription
}

// NewResultSubscriber creates a new result subscriber.
func NewResultSubscriber(client messaging.Client, queryHandler *handlers.AsyncQueryHandler) *ResultSubscriber {
	return &ResultSubscriber{
		client:       client,
		queryHandler: queryHandler,
	}
}

// Start subscribes to search result subjects.
func (s *ResultSubscriber) Start() error {
	// Subscribe to all query results using wildcard
	// Subject pattern: search.results.query.{job_id}
	sub, err := s.client.Subscribe("search.results.query.>", s.handleResult)
	if err != nil {
		return err
	}
	s.sub = sub
	log.Printf("Subscribed to search.results.query.>")
	return nil
}

// Stop unsubscribes from search results.
func (s *ResultSubscriber) Stop() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

// handleResult processes incoming search results from NATS.
func (s *ResultSubscriber) handleResult(ctx context.Context, msg *messaging.Message) error {
	var result SearchResultMessage
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		log.Printf("Failed to unmarshal search result: %v", err)
		return err
	}

	// Determine status
	status := handlers.QueryStatusComplete
	errMsg := ""
	if !result.Success {
		status = handlers.QueryStatusFailed
		errMsg = result.Error
	}

	// Build result data for the cache
	data, err := json.Marshal(map[string]interface{}{
		"total_hits": result.TotalHits,
		"events":     result.Events,
		"took_ms":    result.TookMs,
	})
	if err != nil {
		log.Printf("Failed to marshal result data: %v", err)
		return err
	}

	// Update the handler's cache
	s.queryHandler.SetQueryResult(result.JobID, status, data, errMsg)

	log.Printf("Received result for query %s (success=%v, hits=%d)",
		result.JobID, result.Success, result.TotalHits)

	return nil
}
