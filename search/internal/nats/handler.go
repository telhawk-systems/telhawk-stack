package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/telhawk-systems/telhawk-stack/common/messaging"
	natsclient "github.com/telhawk-systems/telhawk-stack/common/messaging/nats"
	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	"github.com/telhawk-systems/telhawk-stack/search/internal/service"
)

// Handler processes NATS messages for search operations.
type Handler struct {
	client *natsclient.Client
	svc    *service.SearchService
	subs   []messaging.Subscription
	logger *slog.Logger
}

// NewHandler creates a new NATS handler for search operations.
func NewHandler(client *natsclient.Client, svc *service.SearchService) *Handler {
	return &Handler{
		client: client,
		svc:    svc,
		subs:   make([]messaging.Subscription, 0),
		logger: slog.Default().With(slog.String("component", "nats-handler")),
	}
}

// Start subscribes to NATS subjects for search operations.
func (h *Handler) Start(ctx context.Context) error {
	// Subscribe to search jobs with queue group for load balancing
	sub1, err := h.client.QueueSubscribe(
		messaging.SubjectSearchJobsQuery,
		messaging.QueueSearchWorkers,
		h.handleSearchJob,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to search jobs: %w", err)
	}
	h.subs = append(h.subs, sub1)

	// Subscribe to correlation jobs
	sub2, err := h.client.QueueSubscribe(
		messaging.SubjectSearchJobsCorrelate,
		messaging.QueueSearchWorkers,
		h.handleCorrelationJob,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to correlation jobs: %w", err)
	}
	h.subs = append(h.subs, sub2)

	h.logger.Info("NATS handler started",
		slog.String("search_subject", messaging.SubjectSearchJobsQuery),
		slog.String("correlate_subject", messaging.SubjectSearchJobsCorrelate),
		slog.String("queue_group", messaging.QueueSearchWorkers))

	return nil
}

// Stop unsubscribes from all NATS subjects.
func (h *Handler) Stop() error {
	h.logger.Info("Stopping NATS handler")
	for _, sub := range h.subs {
		if err := sub.Unsubscribe(); err != nil {
			h.logger.Warn("Failed to unsubscribe", slog.String("error", err.Error()))
		}
	}
	h.subs = nil
	return nil
}

// Client returns the underlying NATS client for health checks.
func (h *Handler) Client() *natsclient.Client {
	return h.client
}

// handleSearchJob processes ad-hoc search requests from NATS.
func (h *Handler) handleSearchJob(ctx context.Context, msg *messaging.Message) error {
	var req SearchJobRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.Error("Failed to unmarshal search job request",
			slog.String("error", err.Error()))
		return h.replyError(ctx, msg.Reply, "", err)
	}

	h.logger.Debug("Processing search job",
		slog.String("job_id", req.JobID),
		slog.String("query", req.Query))

	start := time.Now()

	// Convert NATS request to internal search request
	searchReq := &models.SearchRequest{
		Query: req.Query,
		Limit: req.Limit,
	}
	if !req.TimeRange.From.IsZero() || !req.TimeRange.To.IsZero() {
		searchReq.TimeRange = &models.TimeRange{
			From: req.TimeRange.From,
			To:   req.TimeRange.To,
		}
	}

	// Execute search using existing service logic
	result, err := h.svc.ExecuteSearch(ctx, searchReq)

	resp := SearchJobResponse{
		JobID:  req.JobID,
		TookMs: time.Since(start).Milliseconds(),
	}

	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		h.logger.Error("Search job failed",
			slog.String("job_id", req.JobID),
			slog.String("error", err.Error()))
	} else {
		resp.Success = true
		resp.TotalHits = int64(result.TotalMatches)
		resp.Events = result.Results
		h.logger.Info("Search job completed",
			slog.String("job_id", req.JobID),
			slog.Int64("total_hits", resp.TotalHits),
			slog.Int64("took_ms", resp.TookMs))
	}

	// Reply via NATS request-reply if reply subject specified
	if msg.Reply != "" {
		if err := h.client.PublishJSON(ctx, msg.Reply, resp); err != nil {
			h.logger.Error("Failed to reply to search job",
				slog.String("job_id", req.JobID),
				slog.String("error", err.Error()))
		}
	}

	// Also publish to job-specific result subject for async consumers (web backend)
	resultSubject := messaging.SearchQueryResultSubject(req.JobID)
	return h.client.PublishJSON(ctx, resultSubject, resp)
}

// handleCorrelationJob processes correlation evaluation requests from NATS.
func (h *Handler) handleCorrelationJob(ctx context.Context, msg *messaging.Message) error {
	var req CorrelationJobRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.Error("Failed to unmarshal correlation job request",
			slog.String("error", err.Error()))
		return h.replyError(ctx, msg.Reply, "", err)
	}

	h.logger.Debug("Processing correlation job",
		slog.String("job_id", req.JobID),
		slog.String("schema_id", req.SchemaID),
		slog.String("query", req.Query))

	start := time.Now()

	// Build search request for correlation query
	searchReq := &models.SearchRequest{
		Query: req.Query,
		Limit: 10000, // High limit for correlation queries
	}
	if !req.TimeRange.From.IsZero() || !req.TimeRange.To.IsZero() {
		searchReq.TimeRange = &models.TimeRange{
			From: req.TimeRange.From,
			To:   req.TimeRange.To,
		}
	}

	// Execute search
	result, err := h.svc.ExecuteSearch(ctx, searchReq)

	resp := CorrelationJobResponse{
		JobID:           req.JobID,
		SchemaID:        req.SchemaID,
		SchemaVersionID: req.SchemaVersionID,
		TookMs:          time.Since(start).Milliseconds(),
	}

	if err != nil {
		resp.Success = false
		resp.Error = err.Error()
		h.logger.Error("Correlation job failed",
			slog.String("job_id", req.JobID),
			slog.String("error", err.Error()))
	} else {
		resp.Success = true
		// Aggregate results by key if specified
		matches := h.aggregateResults(result.Results, req.AggregationKey, req.Threshold)
		resp.Triggered = len(matches) > 0
		resp.MatchCount = len(matches)
		resp.Matches = matches
		h.logger.Info("Correlation job completed",
			slog.String("job_id", req.JobID),
			slog.Bool("triggered", resp.Triggered),
			slog.Int("match_count", resp.MatchCount),
			slog.Int64("took_ms", resp.TookMs))
	}

	// Reply to requester if reply subject specified
	if msg.Reply != "" {
		if err := h.client.PublishJSON(ctx, msg.Reply, resp); err != nil {
			h.logger.Error("Failed to reply to correlation job",
				slog.String("job_id", req.JobID),
				slog.String("error", err.Error()))
		}
	}

	// Always publish to broadcast subject for respond service to consume
	return h.client.PublishJSON(ctx, messaging.SubjectSearchResultsCorrelate, resp)
}

// aggregateResults groups search results by an aggregation key and applies threshold filtering.
func (h *Handler) aggregateResults(events []map[string]interface{}, aggKey string, threshold int) []CorrelationMatch {
	if aggKey == "" {
		// No aggregation - just check if we have enough events
		if len(events) >= threshold {
			var firstSeen, lastSeen time.Time
			if len(events) > 0 {
				firstSeen = h.extractTime(events[len(events)-1])
				lastSeen = h.extractTime(events[0])
			}
			return []CorrelationMatch{{
				AggregationKey: "_all",
				EventCount:     len(events),
				Events:         events,
				FirstSeen:      firstSeen,
				LastSeen:       lastSeen,
			}}
		}
		return nil
	}

	// Group events by aggregation key
	groups := make(map[string][]map[string]interface{})
	for _, event := range events {
		key := h.extractStringField(event, aggKey)
		if key == "" {
			key = "_unknown"
		}
		groups[key] = append(groups[key], event)
	}

	// Filter groups by threshold
	var matches []CorrelationMatch
	for key, groupEvents := range groups {
		if len(groupEvents) >= threshold {
			var firstSeen, lastSeen time.Time
			if len(groupEvents) > 0 {
				firstSeen = h.extractTime(groupEvents[len(groupEvents)-1])
				lastSeen = h.extractTime(groupEvents[0])
			}
			matches = append(matches, CorrelationMatch{
				AggregationKey: key,
				EventCount:     len(groupEvents),
				Events:         groupEvents,
				FirstSeen:      firstSeen,
				LastSeen:       lastSeen,
			})
		}
	}

	return matches
}

// extractStringField extracts a string field from an event, handling nested paths.
func (h *Handler) extractStringField(event map[string]interface{}, field string) string {
	if val, ok := event[field]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// extractTime extracts a timestamp from an event.
func (h *Handler) extractTime(event map[string]interface{}) time.Time {
	// Try common timestamp fields
	for _, field := range []string{"time", "timestamp", "@timestamp", "time_dt"} {
		if val, ok := event[field]; ok {
			switch v := val.(type) {
			case time.Time:
				return v
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					return t
				}
			case float64:
				return time.Unix(int64(v), 0)
			case int64:
				return time.Unix(v, 0)
			}
		}
	}
	return time.Time{}
}

// replyError sends an error response to a reply subject.
func (h *Handler) replyError(ctx context.Context, subject, jobID string, err error) error {
	if subject == "" {
		return nil
	}
	resp := SearchJobResponse{
		JobID:   jobID,
		Success: false,
		Error:   err.Error(),
	}
	return h.client.PublishJSON(ctx, subject, resp)
}
