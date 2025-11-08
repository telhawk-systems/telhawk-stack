package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type DashboardHandler struct {
	queryURL      string
	cacheMutex    sync.RWMutex
	cachedData    []byte
	cacheTime     time.Time
	cacheDuration time.Duration
}

func NewDashboardHandler(queryURL string) *DashboardHandler {
	cacheDuration := 300 * time.Second // Default 5 minutes

	if envSeconds := os.Getenv("DASHBOARD_CACHE_SECONDS"); envSeconds != "" {
		if seconds, err := strconv.Atoi(envSeconds); err == nil && seconds >= 0 {
			cacheDuration = time.Duration(seconds) * time.Second
			log.Printf("Dashboard cache duration set to %d seconds", seconds)
		} else {
			log.Printf("Invalid DASHBOARD_CACHE_SECONDS value '%s', using default 300 seconds", envSeconds)
		}
	}

	return &DashboardHandler{
		queryURL:      queryURL,
		cacheDuration: cacheDuration,
	}
}

func (h *DashboardHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// Check cache first
	h.cacheMutex.RLock()
	if h.cachedData != nil && time.Since(h.cacheTime) < h.cacheDuration {
		// Serve from cache
		cachedData := h.cachedData
		h.cacheMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("X-Cache-Age", fmt.Sprintf("%d", int(time.Since(h.cacheTime).Seconds())))
		w.Write(cachedData)
		return
	}
	h.cacheMutex.RUnlock()

	// Cache miss or expired - fetch from query service
	dashboardQuery := map[string]interface{}{
		"query": "*",
		"limit": 0,
		"aggregations": map[string]interface{}{
			"severity_count": map[string]interface{}{
				"type":  "terms",
				"field": "severity",
				"size":  10,
			},
			"events_by_class": map[string]interface{}{
				"type":  "terms",
				"field": "class_name",
				"size":  10,
			},
			"timeline": map[string]interface{}{
				"type":  "date_histogram",
				"field": "time",
				"opts": map[string]interface{}{
					"interval": "1h",
				},
			},
			"unique_users": map[string]interface{}{
				"type":  "cardinality",
				"field": "actor.user.name.keyword",
			},
			"unique_ips": map[string]interface{}{
				"type":  "cardinality",
				"field": "src_endpoint.ip.keyword",
			},
		},
	}

	queryBody, err := json.Marshal(dashboardQuery)
	if err != nil {
		log.Printf("Failed to marshal dashboard query: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest("POST", h.queryURL+"/api/v1/search", bytes.NewReader(queryBody))
	if err != nil {
		log.Printf("Failed to create query request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Forward auth cookies
	for _, cookie := range r.Cookies() {
		req.AddCookie(cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to query metrics: %v", err)
		http.Error(w, "Failed to fetch dashboard metrics", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Query service returned status %d: %s", resp.StatusCode, string(body))
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Update cache
	h.cacheMutex.Lock()
	h.cachedData = body
	h.cacheTime = time.Now()
	h.cacheMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(body)
}
