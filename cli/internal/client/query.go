package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// QueryClient interacts with the query service via the web backend.
type QueryClient struct {
	baseURL string
	client  *http.Client
}

// NewQueryClient creates a QueryClient pointing at the given base URL.
func NewQueryClient(baseURL string) *QueryClient {
	return &QueryClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Client exposes the underlying http.Client for specialized calls.
func (c *QueryClient) Client() *http.Client { return c.client }

// TimeRange bounds a query using RFC3339 timestamps.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// SearchRequest captures the SPL query and optional constraints.
type SearchRequest struct {
	Query     string     `json:"query"`
	TimeRange *TimeRange `json:"time_range,omitempty"`
	Limit     int        `json:"limit,omitempty"`
}

// Search executes an SPL-compatible search query via the web backend.
func (c *QueryClient) Search(accessToken, query, earliest, latest, last string) ([]map[string]interface{}, error) {
	// Build the search request
	req := SearchRequest{
		Query: query,
		Limit: 100, // Default limit
	}

	// Parse time range from earliest/latest or last
	timeRange, err := parseTimeRange(earliest, latest, last)
	if err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}
	req.TimeRange = timeRange

	// Wrap in JSON:API format
	payload := jsonAPIRequest{
		Data: jsonAPIRequestData{
			Type:       "search",
			Attributes: req,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/query/v1/search", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/vnd.api+json")
	httpReq.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse JSON:API response
	var apiResp jsonAPIResponse
	apiResp.Data = &jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if len(apiResp.Errors) > 0 {
		e := apiResp.Errors[0]
		return nil, fmt.Errorf("%s: %s", e.Code, e.Detail)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
	}

	// Extract results from attributes
	res := apiResp.Data.(*jsonAPIResource)
	results, ok := res.Attributes["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: missing results")
	}

	// Convert to []map[string]interface{}
	out := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		if m, ok := r.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}

	return out, nil
}

// parseTimeRange converts earliest/latest/last strings to a TimeRange.
func parseTimeRange(earliest, latest, last string) (*TimeRange, error) {
	now := time.Now().UTC()

	// If "last" is specified, use it as shorthand (e.g., "1h", "24h", "7d")
	if last != "" {
		duration, err := parseDuration(last)
		if err != nil {
			return nil, fmt.Errorf("invalid 'last' duration: %w", err)
		}
		return &TimeRange{
			From: now.Add(-duration),
			To:   now,
		}, nil
	}

	// Parse earliest/latest
	var tr TimeRange
	tr.To = now // Default to now

	if earliest != "" {
		t, err := parseTimeSpec(earliest, now)
		if err != nil {
			return nil, fmt.Errorf("invalid 'earliest': %w", err)
		}
		tr.From = t
	} else {
		// Default to last 24 hours
		tr.From = now.Add(-24 * time.Hour)
	}

	if latest != "" {
		t, err := parseTimeSpec(latest, now)
		if err != nil {
			return nil, fmt.Errorf("invalid 'latest': %w", err)
		}
		tr.To = t
	}

	return &tr, nil
}

// parseDuration parses duration strings like "1h", "24h", "7d".
func parseDuration(s string) (time.Duration, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty duration")
	}

	// Handle day suffix
	if s[len(s)-1] == 'd' {
		days, err := parseInt(s[:len(s)-1])
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Handle week suffix
	if s[len(s)-1] == 'w' {
		weeks, err := parseInt(s[:len(s)-1])
		if err != nil {
			return 0, err
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}

	// Standard Go duration
	return time.ParseDuration(s)
}

// parseTimeSpec parses time specifications like "-1h", "now", or RFC3339.
func parseTimeSpec(s string, now time.Time) (time.Time, error) {
	if s == "now" {
		return now, nil
	}

	// Relative time (e.g., "-1h", "-7d")
	if len(s) > 0 && s[0] == '-' {
		d, err := parseDuration(s[1:])
		if err != nil {
			return time.Time{}, err
		}
		return now.Add(-d), nil
	}

	// Try RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try common date formats
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %s", s)
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// RawQuery sends a raw JSON query to the /api/query/v1/query endpoint.
// The rawJSON should be a valid JSON object that will be wrapped in JSON:API format.
func (c *QueryClient) RawQuery(accessToken string, rawJSON []byte) ([]map[string]interface{}, error) {
	// Parse the raw JSON to validate it
	var queryObj map[string]interface{}
	if err := json.Unmarshal(rawJSON, &queryObj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Wrap in JSON:API format
	payload := jsonAPIRequest{
		Data: jsonAPIRequestData{
			Type:       "query",
			Attributes: queryObj,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/api/query/v1/query", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/vnd.api+json")
	httpReq.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse JSON:API response
	var apiResp jsonAPIResponse
	apiResp.Data = &jsonAPIResource{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if len(apiResp.Errors) > 0 {
		e := apiResp.Errors[0]
		return nil, fmt.Errorf("%s: %s", e.Code, e.Detail)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query failed with status %d", resp.StatusCode)
	}

	// Extract results from attributes
	res := apiResp.Data.(*jsonAPIResource)
	results, ok := res.Attributes["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: missing results")
	}

	// Convert to []map[string]interface{}
	out := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		if m, ok := r.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}

	return out, nil
}
