package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/telhawk-systems/telhawk-stack/ingest/internal/models"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/service"
	"github.com/telhawk-systems/telhawk-stack/ingest/pkg/hec"
)

type HECHandler struct {
	service *service.IngestService
}

func NewHECHandler(service *service.IngestService) *HECHandler {
	return &HECHandler{
		service: service,
	}
}

func (h *HECHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, hec.ErrInvalidEvent, http.StatusMethodNotAllowed)
		return
	}

	// Authenticate HEC token
	token := hec.ExtractToken(r.Header.Get("Authorization"))
	if token == "" {
		h.sendError(w, hec.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// TODO: Validate token with auth service
	// For now, accept any non-empty token

	// Get client IP
	sourceIP := getClientIP(r)

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
	for _, event := range events {
		if err := h.service.IngestEvent(&event, sourceIP, token); err != nil {
			h.sendError(w, hec.ErrServerBusy, http.StatusServiceUnavailable)
			return
		}
	}

	// Send success response
	h.sendSuccess(w)
}

func (h *HECHandler) HandleRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, hec.ErrInvalidEvent, http.StatusMethodNotAllowed)
		return
	}

	// Authenticate HEC token
	token := hec.ExtractToken(r.Header.Get("Authorization"))
	if token == "" {
		h.sendError(w, hec.ErrUnauthorized, http.StatusUnauthorized)
		return
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

	// Get client IP
	sourceIP := getClientIP(r)

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
	if err := h.service.IngestRaw(body, sourceIP, token, source, sourceType, host); err != nil {
		h.sendError(w, hec.ErrServerBusy, http.StatusServiceUnavailable)
		return
	}

	h.sendSuccess(w)
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
	// Placeholder for ack support
	// Splunk HEC ack mechanism for guaranteed delivery
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"acks": map[string]bool{},
	})
}

func (h *HECHandler) sendSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.HECResponse{
		Text: "Success",
		Code: 0,
	})
}

func (h *HECHandler) sendError(w http.ResponseWriter, hecErr *hec.HECError, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(models.HECResponse{
		Text: hecErr.Text,
		Code: hecErr.Code,
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
