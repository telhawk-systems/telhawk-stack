// Package storage provides OpenSearch client for the respond service.
package storage

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/telhawk-systems/telhawk-stack/common/config"
	"github.com/telhawk-systems/telhawk-stack/respond/internal/models"
)

// OpenSearchStorage provides access to alerts stored in OpenSearch.
type OpenSearchStorage struct {
	client *opensearch.Client
	index  string // Base index pattern (e.g., "telhawk-alerts")
}

// NewOpenSearchStorage creates a new OpenSearch storage client.
func NewOpenSearchStorage() (*OpenSearchStorage, error) {
	cfg := config.GetConfig()
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.OpenSearch.Insecure,
			},
		},
	}

	osCfg := opensearch.Config{
		Addresses: []string{cfg.OpenSearch.URL},
		Username:  cfg.OpenSearch.Username,
		Password:  cfg.OpenSearch.Password,
		Transport: httpClient.Transport,
	}

	client, err := opensearch.NewClient(osCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	// Test connection
	info, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to ping opensearch: %w", err)
	}
	defer info.Body.Close()

	if info.IsError() {
		return nil, fmt.Errorf("opensearch returned error: %s", info.Status())
	}

	return &OpenSearchStorage{
		client: client,
		index:  "telhawk-alerts-*", // Search all alerts indices
	}, nil
}

// ListAlerts queries OpenSearch for security findings (OCSF class_uid 2001).
func (s *OpenSearchStorage) ListAlerts(ctx context.Context, req *models.ListAlertsRequest) (*models.ListAlertsResponse, error) {
	// Build query
	query := s.buildAlertsQuery(req)

	// Calculate pagination
	from := (req.Page - 1) * req.Limit
	if from < 0 {
		from = 0
	}

	searchBody := map[string]interface{}{
		"query": query,
		"from":  from,
		"size":  req.Limit,
		"sort": []map[string]interface{}{
			{"time": map[string]string{"order": "desc"}},
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search body: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(s.index),
		s.client.Search.WithBody(bytes.NewReader(bodyBytes)),
		s.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search alerts: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("opensearch error: %s - %s", res.Status(), string(body))
	}

	// Parse response
	var searchResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string          `json:"_id"`
				Index  string          `json:"_index"`
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Convert to response format
	alerts := make([]*models.Alert, 0, len(searchResult.Hits.Hits))
	for _, hit := range searchResult.Hits.Hits {
		alert, err := s.parseAlertFromOCSF(hit.ID, hit.Source)
		if err != nil {
			// Log and skip malformed alerts
			continue
		}
		alerts = append(alerts, alert)
	}

	total := searchResult.Hits.Total.Value
	totalPages := (total + req.Limit - 1) / req.Limit
	if totalPages < 1 {
		totalPages = 1
	}

	return &models.ListAlertsResponse{
		Alerts: alerts,
		Pagination: models.Pagination{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetAlertByID retrieves a single alert by ID with client_id filter for data isolation.
func (s *OpenSearchStorage) GetAlertByID(ctx context.Context, id string, clientID string) (*models.Alert, error) {
	// Build query with ID and client_id filter for data isolation
	must := []map[string]interface{}{
		{"ids": map[string]interface{}{"values": []string{id}}},
	}

	// CRITICAL: Client ID filter for data isolation (multi-tenant security)
	if clientID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]string{"client_id": clientID},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
	}

	bodyBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(s.index),
		s.client.Search.WithBody(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search for alert: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("opensearch error: %s", res.Status())
	}

	var searchResult struct {
		Hits struct {
			Hits []struct {
				ID     string          `json:"_id"`
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResult.Hits.Hits) == 0 {
		return nil, fmt.Errorf("alert not found: %s", id)
	}

	return s.parseAlertFromOCSF(searchResult.Hits.Hits[0].ID, searchResult.Hits.Hits[0].Source)
}

// buildAlertsQuery constructs the OpenSearch query for alerts.
func (s *OpenSearchStorage) buildAlertsQuery(req *models.ListAlertsRequest) map[string]interface{} {
	must := []map[string]interface{}{
		// Only Security Findings (OCSF class_uid 2001)
		{"term": map[string]interface{}{"class_uid": 2001}},
	}

	// CRITICAL: Client ID filter for data isolation (multi-tenant security)
	if req.ClientID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]string{"client_id": req.ClientID},
		})
	}

	// Add filters
	if req.Severity != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]string{"severity": strings.ToLower(req.Severity)},
		})
	}

	if req.DetectionSchemaID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]string{"detection_schema_id": req.DetectionSchemaID},
		})
	}

	if req.DetectionSchemaVersionID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]string{"detection_schema_version_id": req.DetectionSchemaVersionID},
		})
	}

	// Time range filter
	if req.From != nil || req.To != nil {
		rangeFilter := map[string]interface{}{}
		if req.From != nil {
			rangeFilter["gte"] = req.From.Format(time.RFC3339)
		}
		if req.To != nil {
			rangeFilter["lte"] = req.To.Format(time.RFC3339)
		}
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{"time": rangeFilter},
		})
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"must": must,
		},
	}
}

// parseAlertFromOCSF converts an OCSF Security Finding to our Alert model.
func (s *OpenSearchStorage) parseAlertFromOCSF(id string, source json.RawMessage) (*models.Alert, error) {
	var ocsf struct {
		Time                     string `json:"time"`
		Severity                 string `json:"severity"`
		SeverityID               int    `json:"severity_id"`
		DetectionSchemaID        string `json:"detection_schema_id"`
		DetectionSchemaVersionID string `json:"detection_schema_version_id"`
		FindingInfo              struct {
			UID   string `json:"uid"`
			Title string `json:"title"`
			Desc  string `json:"desc"`
		} `json:"finding_info"`
		Resources []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"resources"`
		RawData       map[string]interface{} `json:"raw_data"`
		MatchedEvents []interface{}          `json:"matched_events"`
	}

	if err := json.Unmarshal(source, &ocsf); err != nil {
		return nil, fmt.Errorf("failed to parse OCSF alert: %w", err)
	}

	// Parse time
	triggeredAt, _ := time.Parse(time.RFC3339, ocsf.Time)
	if triggeredAt.IsZero() {
		triggeredAt = time.Now()
	}

	// Get rule name from resources
	ruleName := "Unknown Rule"
	if len(ocsf.Resources) > 0 {
		ruleName = ocsf.Resources[0].Name
	}

	// Count events
	eventCount := len(ocsf.MatchedEvents)
	if ec, ok := ocsf.RawData["event_count"].(float64); ok {
		eventCount = int(ec)
	}

	alert := &models.Alert{
		AlertID:                  id,
		DetectionSchemaID:        ocsf.DetectionSchemaID,
		DetectionSchemaVersionID: ocsf.DetectionSchemaVersionID,
		DetectionSchemaTitle:     ruleName,
		Title:                    ocsf.FindingInfo.Title,
		Description:              ocsf.FindingInfo.Desc,
		Severity:                 ocsf.Severity,
		Status:                   "open", // Default status
		TriggeredAt:              triggeredAt,
		EventCount:               eventCount,
		Fields:                   ocsf.RawData,
	}

	return alert, nil
}

// Close closes the OpenSearch client connection.
func (s *OpenSearchStorage) Close() error {
	// OpenSearch client doesn't have explicit close
	return nil
}
