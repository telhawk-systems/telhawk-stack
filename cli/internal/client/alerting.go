package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AlertingClient struct {
	baseURL string
	client  *http.Client
}

// Alert represents a security alert from OpenSearch
type Alert struct {
	ID              string                 `json:"id"`
	DetectionName   string                 `json:"detection_name"`
	Severity        string                 `json:"severity"`
	TriggeredAt     time.Time              `json:"triggered_at"`
	EventCount      int                    `json:"event_count,omitempty"`
	DistinctCount   int                    `json:"distinct_count,omitempty"`
	GroupByValues   map[string]interface{} `json:"group_by_values,omitempty"`
	DetectionSchema struct {
		ID        string `json:"id"`
		VersionID string `json:"version_id"`
	} `json:"detection_schema"`
}

// Case represents an investigation case
type Case struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Severity    string     `json:"severity"`
	AssignedTo  string     `json:"assigned_to,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	ClosedBy    string     `json:"closed_by,omitempty"`
	AlertCount  int        `json:"alert_count,omitempty"`
}

type AlertsResponse struct {
	Alerts     []Alert                `json:"alerts"`
	Pagination map[string]interface{} `json:"pagination"`
}

type CasesResponse struct {
	Cases      []Case                 `json:"cases"`
	Pagination map[string]interface{} `json:"pagination"`
}

func NewAlertingClient(baseURL string) *AlertingClient {
	return &AlertingClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *AlertingClient) doRequest(method, path, token string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.client.Do(req)
}

func (c *AlertingClient) ListAlerts(token string, page, limit int, filters map[string]string) (*AlertsResponse, error) {
	path := fmt.Sprintf("/api/v1/alerts?page=%d&limit=%d", page, limit)

	// Add filters to query params
	for key, val := range filters {
		if val != "" {
			path += fmt.Sprintf("&%s=%s", key, val)
		}
	}

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list alerts: %s", string(bodyBytes))
	}

	var alertsResp AlertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertsResp); err != nil {
		return nil, err
	}

	return &alertsResp, nil
}

func (c *AlertingClient) GetAlert(token, id string) (*Alert, error) {
	path := fmt.Sprintf("/api/v1/alerts/%s", id)

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get alert: %s", string(bodyBytes))
	}

	var alert Alert
	if err := json.NewDecoder(resp.Body).Decode(&alert); err != nil {
		return nil, err
	}

	return &alert, nil
}

func (c *AlertingClient) ListCases(token string, page, limit int, status string) (*CasesResponse, error) {
	path := fmt.Sprintf("/api/v1/cases?page=%d&limit=%d", page, limit)
	if status != "" {
		path += fmt.Sprintf("&status=%s", status)
	}

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list cases: %s", string(bodyBytes))
	}

	var casesResp CasesResponse
	if err := json.NewDecoder(resp.Body).Decode(&casesResp); err != nil {
		return nil, err
	}

	return &casesResp, nil
}

func (c *AlertingClient) GetCase(token, id string) (*Case, error) {
	path := fmt.Sprintf("/api/v1/cases/%s", id)

	resp, err := c.doRequest("GET", path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get case: %s", string(bodyBytes))
	}

	var caseData Case
	if err := json.NewDecoder(resp.Body).Decode(&caseData); err != nil {
		return nil, err
	}

	return &caseData, nil
}

func (c *AlertingClient) CreateCase(token, title, description, severity string) (*Case, error) {
	payload := map[string]interface{}{
		"title":       title,
		"description": description,
		"severity":    severity,
	}

	resp, err := c.doRequest("POST", "/api/v1/cases", token, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create case: %s", string(bodyBytes))
	}

	var caseData Case
	if err := json.NewDecoder(resp.Body).Decode(&caseData); err != nil {
		return nil, err
	}

	return &caseData, nil
}

func (c *AlertingClient) UpdateCase(token, id, title, description, assignedTo string) (*Case, error) {
	payload := make(map[string]interface{})
	if title != "" {
		payload["title"] = title
	}
	if description != "" {
		payload["description"] = description
	}
	if assignedTo != "" {
		payload["assigned_to"] = assignedTo
	}

	path := fmt.Sprintf("/api/v1/cases/%s", id)

	resp, err := c.doRequest("PUT", path, token, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update case: %s", string(bodyBytes))
	}

	var caseData Case
	if err := json.NewDecoder(resp.Body).Decode(&caseData); err != nil {
		return nil, err
	}

	return &caseData, nil
}

func (c *AlertingClient) CloseCase(token, id string) error {
	path := fmt.Sprintf("/api/v1/cases/%s/close", id)

	resp, err := c.doRequest("POST", path, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to close case: %s", string(bodyBytes))
	}

	return nil
}

func (c *AlertingClient) ReopenCase(token, id string) error {
	path := fmt.Sprintf("/api/v1/cases/%s/reopen", id)

	resp, err := c.doRequest("POST", path, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reopen case: %s", string(bodyBytes))
	}

	return nil
}
