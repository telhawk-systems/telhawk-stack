package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/common/hecstats"
	"github.com/telhawk-systems/telhawk-stack/common/httputil"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/metrics"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/ratelimit"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/service"
	"github.com/telhawk-systems/telhawk-stack/ingest/pkg/hec"
)

type IngestServiceInterface interface {
	IngestEvent(event *models.HECEvent, sourceIP string, tokenInfo *service.TokenInfo) (string, error)
	IngestRaw(data []byte, sourceIP string, tokenInfo *service.TokenInfo, source, sourceType, host string) (string, error)
	ValidateHECToken(ctx context.Context, token string) (*service.TokenInfo, error)
	GetStats() models.IngestionStats
	QueryAcks(ackIDs []string) map[string]bool
}

type HECHandler struct {
	service        IngestServiceInterface
	rateLimiter    ratelimit.RateLimiter
	statsCollector *hecstats.Collector
}

func NewHECHandler(service IngestServiceInterface, rateLimiter ratelimit.RateLimiter, statsCollector *hecstats.Collector) *HECHandler {
	return &HECHandler{
		service:        service,
		rateLimiter:    rateLimiter,
		statsCollector: statsCollector,
	}
}

func (h *HECHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, hec.ErrInvalidEvent, http.StatusMethodNotAllowed)
		return
	}

	// Get client IP for rate limiting
	sourceIP := httputil.GetClientIP(r)

	// Apply IP-based rate limiting BEFORE expensive operations
	if h.rateLimiter != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		allowed, err := h.rateLimiter.Allow(ctx, "ip:"+sourceIP)
		if err != nil {
			log.Printf("rate limit check error: %v", err)
		} else if !allowed {
			metrics.EventsTotal.WithLabelValues("event", "rate_limited").Inc()
			h.sendError(w, hec.ErrServerBusy, http.StatusTooManyRequests)
			return
		}
	}

	// Authenticate HEC token
	token := hec.ExtractToken(r.Header.Get("Authorization"))
	if token == "" {
		h.sendError(w, hec.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Validate token with auth service
	tokenInfo, err := h.service.ValidateHECToken(r.Context(), token)
	if err != nil {
		log.Printf("HEC token validation failed: %v", err)
		h.sendError(w, hec.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Optional: Apply per-token rate limiting after authentication
	if h.rateLimiter != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		allowed, err := h.rateLimiter.Allow(ctx, "token:"+token)
		if err != nil {
			log.Printf("rate limit check error: %v", err)
		} else if !allowed {
			metrics.EventsTotal.WithLabelValues("event", "token_rate_limited").Inc()
			h.sendError(w, hec.ErrServerBusy, http.StatusTooManyRequests)
			return
		}
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendError(w, hec.ErrInvalidEvent, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		h.sendError(w, hec.ErrNoData, http.StatusBadRequest)
		return
	}

	// Try to parse as single event or batch
	var events []models.HECEvent

	// Try single event first
	var singleEvent models.HECEvent
	if err := json.Unmarshal(body, &singleEvent); err == nil {
		events = append(events, singleEvent)
	} else {
		// Try as newline-delimited JSON (NDJSON)
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var event models.HECEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				h.sendError(w, hec.ErrInvalidEvent, http.StatusBadRequest)
				return
			}
			events = append(events, event)
		}
	}

	// Ingest events
	var ackID string
	for _, event := range events {
		eventAckID, err := h.service.IngestEvent(&event, sourceIP, tokenInfo)
		if err != nil {
			h.sendError(w, hec.ErrServerBusy, http.StatusServiceUnavailable)
			return
		}
		// Use the first ackID for the response (in batch, all share same ack)
		if ackID == "" {
			ackID = eventAckID
		}
	}

	// Record HEC token usage stats
	if h.statsCollector != nil && tokenInfo != nil {
		h.statsCollector.Record(tokenInfo.TokenID, int64(len(events)), net.ParseIP(sourceIP))
	}

	// Check if client requested acknowledgement
	channelID := r.Header.Get("X-Splunk-Request-Channel")
	if channelID != "" && ackID != "" {
		// Return ackID in response
		h.sendSuccessWithAck(w, ackID)
	} else {
		// Send standard success response
		h.sendSuccess(w)
	}
}

func (h *HECHandler) HandleRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, hec.ErrInvalidEvent, http.StatusMethodNotAllowed)
		return
	}

	// Get client IP for rate limiting
	sourceIP := httputil.GetClientIP(r)

	// Apply IP-based rate limiting BEFORE expensive operations
	if h.rateLimiter != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		allowed, err := h.rateLimiter.Allow(ctx, "ip:"+sourceIP)
		if err != nil {
			log.Printf("rate limit check error: %v", err)
		} else if !allowed {
			metrics.EventsTotal.WithLabelValues("raw", "rate_limited").Inc()
			h.sendError(w, hec.ErrServerBusy, http.StatusTooManyRequests)
			return
		}
	}

	// Authenticate HEC token
	token := hec.ExtractToken(r.Header.Get("Authorization"))
	if token == "" {
		h.sendError(w, hec.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Validate token with auth service
	tokenInfo, err := h.service.ValidateHECToken(r.Context(), token)
	if err != nil {
		log.Printf("HEC token validation failed: %v", err)
		h.sendError(w, hec.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Optional: Apply per-token rate limiting after authentication
	if h.rateLimiter != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		allowed, err := h.rateLimiter.Allow(ctx, "token:"+token)
		if err != nil {
			log.Printf("rate limit check error: %v", err)
		} else if !allowed {
			metrics.EventsTotal.WithLabelValues("raw", "token_rate_limited").Inc()
			h.sendError(w, hec.ErrServerBusy, http.StatusTooManyRequests)
			return
		}
	}

	// Get metadata from query params or headers
	source := r.URL.Query().Get("source")
	if source == "" {
		source = r.Header.Get("X-Splunk-Request-Source")
	}

	sourceType := r.URL.Query().Get("sourcetype")
	if sourceType == "" {
		sourceType = r.Header.Get("X-Splunk-Request-Sourcetype")
	}

	host := r.URL.Query().Get("host")
	if host == "" {
		host = r.Header.Get("X-Splunk-Request-Host")
	}

	// Read raw data
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendError(w, hec.ErrInvalidEvent, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		h.sendError(w, hec.ErrNoData, http.StatusBadRequest)
		return
	}

	// Ingest raw event
	ackID, err := h.service.IngestRaw(body, sourceIP, tokenInfo, source, sourceType, host)
	if err != nil {
		h.sendError(w, hec.ErrServerBusy, http.StatusServiceUnavailable)
		return
	}

	// Record HEC token usage stats (raw endpoint = 1 event)
	if h.statsCollector != nil && tokenInfo != nil {
		h.statsCollector.Record(tokenInfo.TokenID, 1, net.ParseIP(sourceIP))
	}

	// Check if client requested acknowledgement
	channelID := r.Header.Get("X-Splunk-Request-Channel")
	if channelID != "" && ackID != "" {
		// Return ackID in response
		h.sendSuccessWithAck(w, ackID)
	} else {
		h.sendSuccess(w)
	}
}

func (h *HECHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func (h *HECHandler) Ready(w http.ResponseWriter, r *http.Request) {
	stats := h.service.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ready",
		"stats":  stats,
	})
}

func (h *HECHandler) Ack(w http.ResponseWriter, r *http.Request) {
	// Parse ack IDs from request body
	var req struct {
		Acks []string `json:"acks"`
	}

	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.sendError(w, hec.ErrInvalidEvent, http.StatusBadRequest)
			return
		}
	}

	// Query ack status if service has ack manager
	result := h.service.QueryAcks(req.Acks)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"acks": result,
	})
}

func (h *HECHandler) sendSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	httputil.WriteJSON(w, http.StatusOK, models.HECResponse{
		Text: "Success",
		Code: 0,
	})
}

func (h *HECHandler) sendSuccessWithAck(w http.ResponseWriter, ackID string) {
	w.Header().Set("Content-Type", "application/json")
	httputil.WriteJSON(w, http.StatusOK, models.HECResponse{
		Text:  "Success",
		Code:  0,
		AckID: ackID,
	})
}

func (h *HECHandler) sendError(w http.ResponseWriter, hecErr *hec.HECError, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	httputil.WriteJSON(w, httpStatus, models.HECResponse{
		Text: hecErr.Text,
		Code: hecErr.Code,
	})
}
