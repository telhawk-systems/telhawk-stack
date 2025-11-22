package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/telhawk-systems/telhawk-stack/search/internal/models"
	"github.com/telhawk-systems/telhawk-stack/search/internal/translator"
	"github.com/telhawk-systems/telhawk-stack/search/internal/validator"
	"github.com/telhawk-systems/telhawk-stack/search/pkg/model"
)

// ExecuteSearch executes a search query against OpenSearch.
func (s *SearchService) ExecuteSearch(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	startTime := time.Now()

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}

	query := s.buildOpenSearchQuery(req)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("encode query: %w", err)
	}

	res, err := s.osClient.Client().Search(
		s.osClient.Client().Search.WithContext(ctx),
		s.osClient.Client().Search.WithIndex(s.osClient.Index()+"*"),
		s.osClient.Client().Search.WithBody(&buf),
		s.osClient.Client().Search.WithSize(limit),
		s.osClient.Client().Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}

	var searchResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
				Sort   []interface{}          `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(searchResult.Hits.Hits))
	var searchAfter []interface{}

	for _, hit := range searchResult.Hits.Hits {
		event := hit.Source
		if req.IncludeFields != nil && len(req.IncludeFields) > 0 {
			filtered := make(map[string]interface{})
			for _, field := range req.IncludeFields {
				if val, ok := event[field]; ok {
					filtered[field] = val
				}
			}
			results = append(results, filtered)
		} else {
			results = append(results, event)
		}
		searchAfter = hit.Sort
	}

	latency := time.Since(startTime).Milliseconds()

	response := &models.SearchResponse{
		RequestID:    generateID(),
		LatencyMS:    int(latency),
		ResultCount:  len(results),
		TotalMatches: searchResult.Hits.Total.Value,
		Results:      results,
	}

	if len(searchAfter) > 0 && len(results) == limit {
		response.SearchAfter = searchAfter
	}

	if searchResult.Aggregations != nil && len(searchResult.Aggregations) > 0 {
		response.Aggregations = searchResult.Aggregations
	}

	return response, nil
}

// ExecuteQuery executes a canonical JSON query and returns search results.
func (s *SearchService) ExecuteQuery(ctx context.Context, q *model.Query) (*models.SearchResponse, error) {
	startTime := time.Now()

	// Validate the query
	validator := validator.NewQueryValidator()
	if err := validator.Validate(q); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Translate the canonical query to OpenSearch DSL
	translator := translator.NewOpenSearchTranslator()
	osQuery, err := translator.Translate(q)
	if err != nil {
		return nil, fmt.Errorf("query translation failed: %w", err)
	}

	// Execute the query
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(osQuery); err != nil {
		return nil, fmt.Errorf("encode query: %w", err)
	}

	res, err := s.osClient.Client().Search(
		s.osClient.Client().Search.WithContext(ctx),
		s.osClient.Client().Search.WithIndex(s.osClient.Index()+"*"),
		s.osClient.Client().Search.WithBody(&buf),
		s.osClient.Client().Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}

	var searchResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
				Sort   []interface{}          `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(searchResult.Hits.Hits))
	var searchAfter []interface{}

	for _, hit := range searchResult.Hits.Hits {
		results = append(results, hit.Source)
		searchAfter = hit.Sort
	}

	latency := time.Since(startTime).Milliseconds()

	// Serialize the OpenSearch query for debugging
	osQueryJSON, _ := json.Marshal(osQuery)

	response := &models.SearchResponse{
		RequestID:       generateID(),
		LatencyMS:       int(latency),
		ResultCount:     len(results),
		TotalMatches:    searchResult.Hits.Total.Value,
		Results:         results,
		OpenSearchQuery: string(osQueryJSON),
	}

	if len(searchAfter) > 0 && len(results) > 0 {
		response.SearchAfter = searchAfter
	}

	if searchResult.Aggregations != nil && len(searchResult.Aggregations) > 0 {
		response.Aggregations = searchResult.Aggregations
	}

	return response, nil
}

// buildOpenSearchQuery constructs an OpenSearch query from a SearchRequest.
func (s *SearchService) buildOpenSearchQuery(req *models.SearchRequest) map[string]interface{} {
	query := make(map[string]interface{})

	boolQuery := make(map[string]interface{})
	must := []interface{}{}
	filter := []interface{}{}

	// CRITICAL: Add client_id filter for data isolation
	// This ensures users can only see events belonging to their client
	if req.ClientID != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{
				"client_id": req.ClientID,
			},
		})
	}

	if req.Query != "" && req.Query != "*" {
		must = append(must, map[string]interface{}{
			"query_string": map[string]interface{}{
				"query":            req.Query,
				"default_operator": "AND",
			},
		})
	}

	if req.TimeRange != nil {
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{
				"time": map[string]interface{}{
					"gte": req.TimeRange.From.Unix(),
					"lte": req.TimeRange.To.Unix(),
				},
			},
		})
	}

	// Build bool query
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	// If we have any bool conditions, use bool query; otherwise match_all
	if len(must) > 0 || len(filter) > 0 {
		query["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	} else {
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	s.addSortAndSearchAfter(query, req)
	s.addAggregations(query, req)

	return query
}

// addSortAndSearchAfter adds sorting and pagination to a query.
func (s *SearchService) addSortAndSearchAfter(query map[string]interface{}, req *models.SearchRequest) {
	if req.Sort != nil {
		query["sort"] = []interface{}{
			map[string]interface{}{
				req.Sort.Field: map[string]interface{}{
					"order": req.Sort.Order,
				},
			},
		}
	}

	if req.SearchAfter != nil && len(req.SearchAfter) > 0 {
		query["search_after"] = req.SearchAfter
	}
}

// addAggregations adds aggregations to a query.
func (s *SearchService) addAggregations(query map[string]interface{}, req *models.SearchRequest) {
	if req.Aggregations == nil || len(req.Aggregations) == 0 {
		return
	}

	aggs := make(map[string]interface{})
	for name, aggReq := range req.Aggregations {
		switch aggReq.Type {
		case "terms":
			termsAgg := map[string]interface{}{
				"field": aggReq.Field,
			}
			if aggReq.Size > 0 {
				termsAgg["size"] = aggReq.Size
			} else {
				termsAgg["size"] = 10
			}
			for k, v := range aggReq.Opts {
				termsAgg[k] = v
			}
			aggs[name] = map[string]interface{}{
				"terms": termsAgg,
			}
		case "date_histogram":
			histAgg := map[string]interface{}{
				"field": aggReq.Field,
			}
			if interval, ok := aggReq.Opts["interval"]; ok {
				histAgg["fixed_interval"] = interval
			} else {
				histAgg["fixed_interval"] = "1h"
			}
			for k, v := range aggReq.Opts {
				if k != "interval" {
					histAgg[k] = v
				}
			}
			aggs[name] = map[string]interface{}{
				"date_histogram": histAgg,
			}
		case "avg", "sum", "min", "max", "cardinality":
			aggs[name] = map[string]interface{}{
				aggReq.Type: map[string]interface{}{
					"field": aggReq.Field,
				},
			}
		case "stats":
			aggs[name] = map[string]interface{}{
				"stats": map[string]interface{}{
					"field": aggReq.Field,
				},
			}
		}
	}

	if len(aggs) > 0 {
		query["aggs"] = aggs
	}
}
