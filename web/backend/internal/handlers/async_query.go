// Package handlers provides HTTP handlers for the web backend.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/common/messaging"
)

// QueryStatus represents the status of an async query job.
type QueryStatus string

const (
	QueryStatusPending  QueryStatus = "pending"
	QueryStatusComplete QueryStatus = "complete"
	QueryStatusFailed   QueryStatus = "failed"
	queryResultCacheTTL             = 5 * time.Minute
)

// AsyncQueryHandler handles async query submission and result retrieval.
type AsyncQueryHandler struct {
	publisher messaging.Publisher
	subject   string

	// In-memory cache for query results
	resultsMu sync.RWMutex
	results   map[string]*queryResult
}

type queryResult struct {
	Status    QueryStatus     `json:"status"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// SubmitQueryRequest is the request body for submitting a query.
type SubmitQueryRequest struct {
	Query     string `json:"query"`
	TimeRange string `json:"time_range,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// SubmitQueryResponse is the response for a submitted query.
type SubmitQueryResponse struct {
	QueryID string `json:"query_id"`
}

// QueryStatusResponse is the response for checking query status.
type QueryStatusResponse struct {
	QueryID string          `json:"query_id"`
	Status  QueryStatus     `json:"status"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// QueryJobMessage is the message published to NATS for query execution.
type QueryJobMessage struct {
	QueryID   string `json:"query_id"`
	Query     string `json:"query"`
	TimeRange string `json:"time_range,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	ReplyTo   string `json:"reply_to,omitempty"`
}

// NewAsyncQueryHandler creates a new AsyncQueryHandler.
func NewAsyncQueryHandler(publisher messaging.Publisher, subject string) *AsyncQueryHandler {
	h := &AsyncQueryHandler{
		publisher: publisher,
		subject:   subject,
		results:   make(map[string]*queryResult),
	}

	// Start background cleanup goroutine
	go h.cleanupExpiredResults()

	return h
}

// SubmitQuery handles POST /api/async-query/submit
// It accepts a search query, generates a query ID, publishes to NATS, and returns immediately.
func (h *AsyncQueryHandler) SubmitQuery(w http.ResponseWriter, r *http.Request) {
	var req SubmitQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	// Generate query ID
	queryID := uuid.New().String()

	// Create job message
	jobMsg := QueryJobMessage{
		QueryID:   queryID,
		Query:     req.Query,
		TimeRange: req.TimeRange,
		Limit:     req.Limit,
	}

	// Serialize message
	data, err := json.Marshal(jobMsg)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to serialize query job: %v", err), http.StatusInternalServerError)
		return
	}

	// Publish to NATS
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.publisher.Publish(ctx, h.subject, data); err != nil {
		http.Error(w, fmt.Sprintf("failed to submit query: %v", err), http.StatusInternalServerError)
		return
	}

	// Store pending result in cache
	h.resultsMu.Lock()
	h.results[queryID] = &queryResult{
		Status:    QueryStatusPending,
		CreatedAt: time.Now(),
	}
	h.resultsMu.Unlock()

	// Return query ID
	resp := SubmitQueryResponse{
		QueryID: queryID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

// GetQueryStatus handles GET /api/async-query/status/{id}
// It returns the status and results (if available) for a query.
func (h *AsyncQueryHandler) GetQueryStatus(w http.ResponseWriter, r *http.Request) {
	queryID := r.PathValue("id")
	if queryID == "" {
		http.Error(w, "query ID is required", http.StatusBadRequest)
		return
	}

	h.resultsMu.RLock()
	result, exists := h.results[queryID]
	h.resultsMu.RUnlock()

	if !exists {
		http.Error(w, "query not found", http.StatusNotFound)
		return
	}

	resp := QueryStatusResponse{
		QueryID: queryID,
		Status:  result.Status,
		Data:    result.Data,
		Error:   result.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SetQueryResult updates the result for a query (called when results arrive).
// This method is exposed for the result subscriber to call.
func (h *AsyncQueryHandler) SetQueryResult(queryID string, status QueryStatus, data json.RawMessage, errMsg string) {
	h.resultsMu.Lock()
	defer h.resultsMu.Unlock()

	if result, exists := h.results[queryID]; exists {
		result.Status = status
		result.Data = data
		result.Error = errMsg
	} else {
		// Result arrived before or without initial submit (edge case)
		h.results[queryID] = &queryResult{
			Status:    status,
			Data:      data,
			Error:     errMsg,
			CreatedAt: time.Now(),
		}
	}
}

// cleanupExpiredResults periodically removes old results from the cache.
func (h *AsyncQueryHandler) cleanupExpiredResults() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.resultsMu.Lock()
		now := time.Now()
		for id, result := range h.results {
			if now.Sub(result.CreatedAt) > queryResultCacheTTL {
				delete(h.results, id)
			}
		}
		h.resultsMu.Unlock()
	}
}
