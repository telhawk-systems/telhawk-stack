package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/service"
)

// ProcessorHandler manages normalization HTTP endpoints.
type ProcessorHandler struct {
	processor *service.Processor
}

// NewProcessorHandler constructs a new handler.
func NewProcessorHandler(p *service.Processor) *ProcessorHandler {
	return &ProcessorHandler{processor: p}
}

// NormalizationRequest represents the inbound payload for normalization.
type NormalizationRequest struct {
	ID         string            `json:"id"`
	Source     string            `json:"source"`
	SourceType string            `json:"source_type"`
	Format     string            `json:"format"`
	Payload    string            `json:"payload"`
	Attributes map[string]string `json:"attributes,omitempty"`
	ReceivedAt string            `json:"received_at"`
}

// NormalizationResponse returns the normalized OCSF event.
type NormalizationResponse struct {
	Event json.RawMessage `json:"event"`
}

// Normalize handles POST /api/v1/normalize requests.
func (h *ProcessorHandler) Normalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req NormalizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	payload, err := base64.StdEncoding.DecodeString(req.Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", "payload must be base64 encoded")
		return
	}

	receivedAt := time.Now().UTC()
	if req.ReceivedAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, req.ReceivedAt); err == nil {
			receivedAt = parsed
		}
	}

	envelope := &model.RawEventEnvelope{
		ID:         req.ID,
		Source:     req.Source,
		SourceType: req.SourceType,
		Format:     req.Format,
		Payload:    payload,
		Attributes: req.Attributes,
		ReceivedAt: receivedAt,
	}

	event, err := h.processor.Process(r.Context(), envelope)
	if err != nil {
		writeError(w, http.StatusBadRequest, "normalization_failed", err.Error())
		return
	}

	serialized, err := pipeline.MarshalResult(event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "serialization_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, NormalizationResponse{Event: serialized})
}

// Health handles GET /healthz.
func (h *ProcessorHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, h.processor.Health())
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	type errorBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	writeJSON(w, status, errorBody{Code: code, Message: message})
}

func methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
}
