package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/telhawk-systems/telhawk-stack/common/httputil"

	"github.com/opensearch-project/opensearch-go/v2/opensearchutil"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/client"
	"github.com/telhawk-systems/telhawk-stack/storage/internal/indexmgr"
)

type StorageHandler struct {
	client       *client.OpenSearchClient
	indexManager *indexmgr.IndexManager
}

func NewStorageHandler(client *client.OpenSearchClient, indexManager *indexmgr.IndexManager) *StorageHandler {
	return &StorageHandler{
		client:       client,
		indexManager: indexManager,
	}
}

type IngestRequest struct {
	Events []map[string]interface{} `json:"events"`
}

type IngestResponse struct {
	Indexed int      `json:"indexed"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

func (h *StorageHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Events) == 0 {
		http.Error(w, "No events provided", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := h.indexEvents(ctx, req.Events)

	w.Header().Set("Content-Type", "application/json")
	if resp.Failed > 0 {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *StorageHandler) BulkIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}

	events := make([]map[string]interface{}, 0)
	lines := strings.Split(string(body), "\n")

	for i := 0; i < len(lines); i += 2 {
		if i+1 >= len(lines) || strings.TrimSpace(lines[i]) == "" {
			break
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(lines[i+1]), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		http.Error(w, "No valid events in bulk request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	resp := h.indexEvents(ctx, events)

	w.Header().Set("Content-Type", "application/json")
	if resp.Failed > 0 {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *StorageHandler) indexEvents(ctx context.Context, events []map[string]interface{}) IngestResponse {
	resp := IngestResponse{}
	indexName := h.indexManager.GetWriteAlias()

	bi, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client: h.client.Client(),
		Index:  indexName,
	})
	if err != nil {
		resp.Failed = len(events)
		resp.Errors = []string{fmt.Sprintf("Failed to create bulk indexer: %v", err)}
		return resp
	}

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("Failed to marshal event: %v", err))
			continue
		}

		// DEBUG: Log JSON being sent to OpenSearch (controlled by DEBUG_DUMP_JSON env var)
		if os.Getenv("DEBUG_DUMP_JSON") == "true" {
			log.Printf("=== DEBUG: JSON being indexed to OpenSearch ===\n%s\n=== END DEBUG ===", string(data))
		}

		err = bi.Add(ctx, opensearchutil.BulkIndexerItem{
			Action: "index",
			Body:   bytes.NewReader(data),
			OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem) {
				resp.Indexed++
			},
			OnFailure: func(ctx context.Context, item opensearchutil.BulkIndexerItem, res opensearchutil.BulkIndexerResponseItem, err error) {
				resp.Failed++
				if err != nil {
					resp.Errors = append(resp.Errors, err.Error())
				} else {
					resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %s", res.Error.Type, res.Error.Reason))
				}
			},
		})

		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("Failed to add to bulk indexer: %v", err))
		}
	}

	if err := bi.Close(ctx); err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("Bulk indexer close error: %v", err))
	}

	return resp
}

func (h *StorageHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *StorageHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	info, err := h.client.Client().Info(h.client.Client().Info.WithContext(ctx))
	if err != nil || info.IsError() {
		httputil.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "reason": "opensearch unavailable"})
		return
	}
	defer info.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
