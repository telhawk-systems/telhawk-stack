package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/telhawk-systems/telhawk-stack/common/hecstats"
)

// HECStatsHandler provides endpoints for HEC token usage statistics.
type HECStatsHandler struct {
	statsClient *hecstats.Client
}

// NewHECStatsHandler creates a new HEC stats handler.
// Returns nil if statsClient is nil (Redis unavailable).
func NewHECStatsHandler(statsClient *hecstats.Client) *HECStatsHandler {
	if statsClient == nil {
		return nil
	}
	return &HECStatsHandler{
		statsClient: statsClient,
	}
}

// GetStats returns usage statistics for a single HEC token.
// GET /api/hec/stats/{id}
func (h *HECStatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tokenID := r.PathValue("id")
	if tokenID == "" {
		http.Error(w, `{"error":"token ID required"}`, http.StatusBadRequest)
		return
	}

	stats, err := h.statsClient.GetStats(r.Context(), tokenID)
	if err != nil {
		log.Printf("Failed to get HEC stats for token %s: %v", tokenID, err)
		http.Error(w, `{"error":"failed to retrieve stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetMultiStats returns usage statistics for multiple HEC tokens.
// POST /api/hec/stats/batch
// Request body: {"token_ids": ["id1", "id2", ...]}
func (h *HECStatsHandler) GetMultiStats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenIDs []string `json:"token_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.TokenIDs) == 0 {
		http.Error(w, `{"error":"token_ids required"}`, http.StatusBadRequest)
		return
	}

	// Limit batch size to prevent abuse
	if len(req.TokenIDs) > 100 {
		http.Error(w, `{"error":"maximum 100 tokens per request"}`, http.StatusBadRequest)
		return
	}

	stats, err := h.statsClient.GetMultiStats(r.Context(), req.TokenIDs)
	if err != nil {
		log.Printf("Failed to get HEC stats for multiple tokens: %v", err)
		http.Error(w, `{"error":"failed to retrieve stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
