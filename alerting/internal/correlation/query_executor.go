package correlation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/pkg/model"
)

// QueryExecutor executes canonical JSON queries against OpenSearch
// It wraps the existing query translator to ensure consistency
type QueryExecutor struct {
	storageURL string
	username   string
	password   string
	insecure   bool
	httpClient *http.Client
	translator *SimpleTranslator
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(storageURL, username, password string, insecure bool) *QueryExecutor {
	// Create HTTP client with optional TLS skip verification
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &QueryExecutor{
		storageURL: storageURL,
		username:   username,
		password:   password,
		insecure:   insecure,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		translator: &SimpleTranslator{},
	}
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Events     []*Event
	TotalCount int64
	Took       int64 // Milliseconds
}

// ExecuteQuery executes a canonical JSON query with time window
func (qe *QueryExecutor) ExecuteQuery(ctx context.Context, query *model.Query, timeWindow time.Duration) (*QueryResult, error) {
	// Apply time window if not already set
	if query.TimeRange == nil {
		query.TimeRange = &model.TimeRangeDef{
			Last: fmt.Sprintf("%dms", timeWindow.Milliseconds()),
		}
	}

	// Translate to OpenSearch DSL
	osQuery, err := qe.translator.Translate(query)
	if err != nil {
		return nil, fmt.Errorf("failed to translate query: %w", err)
	}

	// Execute against OpenSearch
	return qe.executeOpenSearchQuery(ctx, osQuery)
}

// ExecuteCountQuery executes a count aggregation query
func (qe *QueryExecutor) ExecuteCountQuery(ctx context.Context, query *model.Query, timeWindow time.Duration, groupBy []string) (map[string]int64, error) {
	// Apply time window
	if query.TimeRange == nil {
		query.TimeRange = &model.TimeRangeDef{
			Last: fmt.Sprintf("%dms", timeWindow.Milliseconds()),
		}
	}

	// Add aggregations for grouping
	if len(groupBy) > 0 {
		query.Aggregations = []model.Aggregation{
			{
				Type:  model.AggTypeTerms,
				Field: groupBy[0], // TODO: Support nested grouping
				Name:  "groups",
				Size:  1000,
			},
		}
	}

	// Set size to 0 since we only want aggregations
	query.Limit = 0

	// Translate and execute
	osQuery, err := qe.translator.Translate(query)
	if err != nil {
		return nil, fmt.Errorf("failed to translate count query: %w", err)
	}

	result, err := qe.executeOpenSearchAggQuery(ctx, osQuery)
	if err != nil {
		return nil, err
	}

	// Parse aggregation results
	counts := make(map[string]int64)
	if len(groupBy) == 0 {
		// No grouping - return total count
		if hits, ok := result["hits"].(map[string]interface{}); ok {
			if total, ok := hits["total"].(map[string]interface{}); ok {
				if value, ok := total["value"].(float64); ok {
					counts["_total"] = int64(value)
				}
			}
		}
	} else {
		// Parse grouped results
		counts = parseAggregationBuckets(result, "groups")
	}

	return counts, nil
}

// ExecuteCardinalityQuery executes a cardinality (distinct count) aggregation
func (qe *QueryExecutor) ExecuteCardinalityQuery(ctx context.Context, query *model.Query, timeWindow time.Duration, field string, groupBy []string) (map[string]int64, error) {
	// Apply time window
	if query.TimeRange == nil {
		query.TimeRange = &model.TimeRangeDef{
			Last: fmt.Sprintf("%dms", timeWindow.Milliseconds()),
		}
	}

	// Add cardinality aggregation
	if len(groupBy) > 0 {
		query.Aggregations = []model.Aggregation{
			{
				Type:  model.AggTypeTerms,
				Field: groupBy[0],
				Name:  "groups",
				Size:  1000,
				Aggregations: []model.Aggregation{
					{
						Type:  model.AggTypeCardinality,
						Field: field,
						Name:  "distinct_count",
					},
				},
			},
		}
	} else {
		query.Aggregations = []model.Aggregation{
			{
				Type:  model.AggTypeCardinality,
				Field: field,
				Name:  "distinct_count",
			},
		}
	}

	query.Limit = 0

	// Translate and execute
	osQuery, err := qe.translator.Translate(query)
	if err != nil {
		return nil, fmt.Errorf("failed to translate cardinality query: %w", err)
	}

	result, err := qe.executeOpenSearchAggQuery(ctx, osQuery)
	if err != nil {
		return nil, err
	}

	// Parse results
	counts := make(map[string]int64)
	if len(groupBy) == 0 {
		// No grouping
		if aggs, ok := result["aggregations"].(map[string]interface{}); ok {
			if distinctCount, ok := aggs["distinct_count"].(map[string]interface{}); ok {
				if value, ok := distinctCount["value"].(float64); ok {
					counts["_total"] = int64(value)
				}
			}
		}
	} else {
		// Grouped results
		if aggs, ok := result["aggregations"].(map[string]interface{}); ok {
			if groups, ok := aggs["groups"].(map[string]interface{}); ok {
				if buckets, ok := groups["buckets"].([]interface{}); ok {
					for _, bucket := range buckets {
						b := bucket.(map[string]interface{})
						key := fmt.Sprintf("%v", b["key"])
						if distinctCount, ok := b["distinct_count"].(map[string]interface{}); ok {
							if value, ok := distinctCount["value"].(float64); ok {
								counts[key] = int64(value)
							}
						}
					}
				}
			}
		}
	}

	return counts, nil
}

// executeOpenSearchQuery executes a query and returns events
func (qe *QueryExecutor) executeOpenSearchQuery(ctx context.Context, query map[string]interface{}) (*QueryResult, error) {
	// Serialize query
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/telhawk-events-*/_search", qe.storageURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(qe.username, qe.password)

	// Execute request
	resp, err := qe.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("query failed with status %d (failed to read response body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract events
	hits := result["hits"].(map[string]interface{})
	total := hits["total"].(map[string]interface{})["value"].(float64)
	took := result["took"].(float64)
	hitsList := hits["hits"].([]interface{})

	events := make([]*Event, 0, len(hitsList))
	for _, hit := range hitsList {
		h := hit.(map[string]interface{})
		source := h["_source"].(map[string]interface{})

		event := &Event{
			RawSource: source,
			Fields:    source,
		}

		// Extract time field (handle both Unix timestamp and RFC3339 string)
		if timeVal, ok := source["time"]; ok {
			if timeInt, ok := timeVal.(float64); ok {
				// Unix timestamp in milliseconds
				event.Time = time.UnixMilli(int64(timeInt))
			} else if timeStr, ok := timeVal.(string); ok {
				// RFC3339 format string
				if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
					event.Time = parsedTime
				}
			}
		}

		events = append(events, event)
	}

	return &QueryResult{
		Events:     events,
		TotalCount: int64(total),
		Took:       int64(took),
	}, nil
}

// executeOpenSearchAggQuery executes an aggregation query and returns raw results
func (qe *QueryExecutor) executeOpenSearchAggQuery(ctx context.Context, query map[string]interface{}) (map[string]interface{}, error) {
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/telhawk-events-*/_search", qe.storageURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(qe.username, qe.password)

	resp, err := qe.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("query failed with status %d (failed to read response body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// parseAggregationBuckets parses aggregation buckets into a count map
func parseAggregationBuckets(result map[string]interface{}, aggName string) map[string]int64 {
	counts := make(map[string]int64)

	aggs, ok := result["aggregations"].(map[string]interface{})
	if !ok {
		return counts
	}

	groups, ok := aggs[aggName].(map[string]interface{})
	if !ok {
		return counts
	}

	buckets, ok := groups["buckets"].([]interface{})
	if !ok {
		return counts
	}

	for _, bucket := range buckets {
		b := bucket.(map[string]interface{})
		key := fmt.Sprintf("%v", b["key"])
		docCount := b["doc_count"].(float64)
		counts[key] = int64(docCount)
	}

	return counts
}
