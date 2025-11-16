package evaluator

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPStorageClient implements StorageClient using OpenSearch HTTP API
type HTTPStorageClient struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewHTTPStorageClient creates a new HTTP storage client
func NewHTTPStorageClient(baseURL, username, password string, insecure bool) *HTTPStorageClient {
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &HTTPStorageClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// FetchEvents retrieves events from OpenSearch since the given timestamp
func (c *HTTPStorageClient) FetchEvents(ctx context.Context, since time.Time) ([]*Event, error) {
	// Build query to fetch events since timestamp
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"time": map[string]interface{}{
					"gte": since.UnixMilli(),
				},
			},
		},
		"sort": []map[string]interface{}{
			{"time": map[string]string{"order": "asc"}},
		},
		"size": 1000, // Limit to 1000 events per poll
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Search across all event indices
	url := fmt.Sprintf("%s/telhawk-events-*/_search", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("search failed with status %d (failed to read response body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Index  string                 `json:"_index"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	events := make([]*Event, 0, len(searchResp.Hits.Hits))
	for _, hit := range searchResp.Hits.Hits {
		// Extract timestamp from event
		var timestamp time.Time
		if timeVal, ok := hit.Source["time"].(float64); ok {
			timestamp = time.UnixMilli(int64(timeVal))
		} else {
			timestamp = time.Now()
		}

		events = append(events, &Event{
			ID:        hit.ID,
			Index:     hit.Index,
			Source:    hit.Source,
			Timestamp: timestamp,
		})
	}

	return events, nil
}

// StoreAlert stores an alert in OpenSearch
func (c *HTTPStorageClient) StoreAlert(ctx context.Context, alert *Alert) error {
	// Convert alert to OCSF Detection Finding (class_uid: 2004)
	ocsfAlert := map[string]interface{}{
		"class_uid":    2004,
		"category_uid": 2,
		"type_uid":     200401,
		"activity_id":  1,
		"severity_id":  c.getSeverityID(alert.Severity),
		"severity":     alert.Severity,
		"time":         alert.Timestamp.UnixMilli(),
		"metadata": map[string]interface{}{
			"version":     "1.1.0",
			"product":     map[string]string{"name": "TelHawk", "vendor_name": "TelHawk Systems"},
			"logged_time": alert.Timestamp.UnixMilli(),
		},
		"finding_info": map[string]interface{}{
			"title":         alert.Title,
			"desc":          alert.Description,
			"uid":           alert.ID,
			"created_time":  alert.Timestamp.UnixMilli(),
			"modified_time": alert.Timestamp.UnixMilli(),
			"types":         []string{"Detection"},
		},
		"resources": []map[string]interface{}{
			{
				"uid":  alert.DetectionSchemaID,
				"name": fmt.Sprintf("Detection Schema v%s", alert.DetectionSchemaVersionID),
				"type": "Detection Schema",
			},
		},
		"detection_schema_id":         alert.DetectionSchemaID,
		"detection_schema_version_id": alert.DetectionSchemaVersionID,
		"matched_events":              alert.MatchedEvents,
		"raw_data":                    alert.Metadata,
	}

	alertJSON, err := json.Marshal(ocsfAlert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	// Index into telhawk-alerts-YYYY.MM.DD
	indexName := fmt.Sprintf("telhawk-alerts-%s", time.Now().Format("2006.01.02"))
	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, indexName, alert.ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(alertJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to index alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("indexing failed with status %d (failed to read response body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("indexing failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Query performs a generic OpenSearch query
func (c *HTTPStorageClient) Query(method, path string, body []byte) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// getSeverityID maps severity string to OCSF severity_id
func (c *HTTPStorageClient) getSeverityID(severity string) int {
	severityMap := map[string]int{
		"info":     1,
		"low":      2,
		"medium":   3,
		"high":     4,
		"critical": 5,
	}
	if id, ok := severityMap[severity]; ok {
		return id
	}
	return 0 // Unknown
}
