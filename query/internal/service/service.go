package service

import (
	"bytes"
	context "context"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/client"
	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

var (
	ErrAlertNotFound     = errors.New("alert not found")
	ErrDashboardNotFound = errors.New("dashboard not found")
)

// QueryService provides implementations for the query API surface.
type QueryService struct {
	mu         sync.RWMutex
	startedAt  time.Time
	version    string
	alerts     map[string]models.Alert
	dashboards map[string]models.Dashboard
	osClient   *client.OpenSearchClient
}

// NewQueryService seeds in-memory data used by the HTTP handlers.
func NewQueryService(version string, osClient *client.OpenSearchClient) *QueryService {
	now := time.Now().UTC()
	return &QueryService{
		startedAt: now,
		version:   version,
		osClient:  osClient,
		alerts: map[string]models.Alert{
			"a1b6e360-3c35-4d63-87fd-03b27ef77d1f": {
				ID:          "a1b6e360-3c35-4d63-87fd-03b27ef77d1f",
				Name:        "Suspicious admin logins",
				Description: "Detects admin logins from unusual geographies",
				Query:       "index=ocsf authentication where user_role=\"admin\" and geoip.confidence < 30",
				Severity:    "high",
				Schedule: models.AlertSchedule{
					IntervalMinutes: 5,
					LookbackMinutes: 15,
				},
				Status: "active",
				Owner:  "soc@telhawk.local",
				LastTriggeredAt: func() *time.Time {
					val := now.Add(-2 * time.Hour)
					return &val
				}(),
			},
		},
		dashboards: map[string]models.Dashboard{
			"threat-overview": {
				ID:          "threat-overview",
				Name:        "Threat Overview",
				Description: "Executive summary of threat detections over the last 24 hours",
				Widgets: []map[string]interface{}{
					{
						"id":      "detections-by-severity",
						"type":    "bar",
						"title":   "Detections by severity",
						"query":   "index=ocsf stats count by severity",
						"display": map[string]interface{}{"palette": "risk"},
					},
					{
						"id":      "top-alerts",
						"type":    "table",
						"title":   "Top firing alerts",
						"query":   "index=alerts sort -trigger_count | head 10",
						"columns": []string{"alert", "trigger_count", "severity"},
					},
				},
			},
		},
	}
}

// ExecuteSearch executes a search query against OpenSearch.
func (s *QueryService) ExecuteSearch(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
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
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(searchResult.Hits.Hits))
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
	}

	latency := time.Since(startTime).Milliseconds()
	
	return &models.SearchResponse{
		RequestID:   generateID(),
		LatencyMS:   int(latency),
		ResultCount: len(results),
		Results:     results,
	}, nil
}

func (s *QueryService) buildOpenSearchQuery(req *models.SearchRequest) map[string]interface{} {
	query := make(map[string]interface{})
	
	boolQuery := make(map[string]interface{})
	must := []interface{}{}
	
	if req.Query != "" && req.Query != "*" {
		must = append(must, map[string]interface{}{
			"query_string": map[string]interface{}{
				"query":            req.Query,
				"default_operator": "AND",
			},
		})
	}
	
	if req.TimeRange != nil {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"time": map[string]interface{}{
					"gte": req.TimeRange.From.Unix(),
					"lte": req.TimeRange.To.Unix(),
				},
			},
		})
	}
	
	if len(must) > 0 {
		boolQuery["must"] = must
	} else {
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
		if req.Sort != nil {
			query["sort"] = []interface{}{
				map[string]interface{}{
					req.Sort.Field: map[string]interface{}{
						"order": req.Sort.Order,
					},
				},
			}
		}
		return query
	}
	
	query["query"] = map[string]interface{}{
		"bool": boolQuery,
	}
	
	if req.Sort != nil {
		query["sort"] = []interface{}{
			map[string]interface{}{
				req.Sort.Field: map[string]interface{}{
					"order": req.Sort.Order,
				},
			},
		}
	}
	
	return query
}

// ListAlerts returns all stubbed alert definitions sorted by name.
func (s *QueryService) ListAlerts(ctx context.Context) (*models.AlertListResponse, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	alerts := make([]models.Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		alerts = append(alerts, alert)
	}
	sort.Slice(alerts, func(i, j int) bool { return alerts[i].Name < alerts[j].Name })
	return &models.AlertListResponse{Alerts: alerts}, nil
}

// UpsertAlert creates or updates an alert definition.
func (s *QueryService) UpsertAlert(ctx context.Context, req *models.AlertRequest) (*models.Alert, bool, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	created := false
	var id string
	if req.ID != nil && *req.ID != "" {
		id = *req.ID
	} else {
		id = generateID()
		created = true
	}
	alert := models.Alert{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Query:       req.Query,
		Severity:    req.Severity,
		Schedule:    req.Schedule,
		Status:      req.Status,
		Owner:       req.Owner,
	}
	if alert.Status == "" {
		alert.Status = "active"
	}
	existing, ok := s.alerts[id]
	if ok {
		alert.LastTriggeredAt = existing.LastTriggeredAt
	}
	s.alerts[id] = alert
	return &alert, created, nil
}

// GetAlert retrieves an alert definition by id.
func (s *QueryService) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	alert, ok := s.alerts[id]
	if !ok {
		return nil, ErrAlertNotFound
	}
	return &alert, nil
}

// PatchAlert applies partial updates to an alert.
func (s *QueryService) PatchAlert(ctx context.Context, id string, req *models.AlertPatchRequest) (*models.Alert, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	alert, ok := s.alerts[id]
	if !ok {
		return nil, ErrAlertNotFound
	}
	if req.Status != "" {
		alert.Status = req.Status
	}
	if req.Owner != "" {
		alert.Owner = req.Owner
	}
	s.alerts[id] = alert
	return &alert, nil
}

// ListDashboards returns predefined dashboards.
func (s *QueryService) ListDashboards(ctx context.Context) (*models.DashboardListResponse, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	dashboards := make([]models.Dashboard, 0, len(s.dashboards))
	for _, dashboard := range s.dashboards {
		dashboards = append(dashboards, dashboard)
	}
	sort.Slice(dashboards, func(i, j int) bool { return dashboards[i].Name < dashboards[j].Name })
	return &models.DashboardListResponse{Dashboards: dashboards}, nil
}

// GetDashboard retrieves a dashboard by id.
func (s *QueryService) GetDashboard(ctx context.Context, id string) (*models.Dashboard, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	dashboard, ok := s.dashboards[id]
	if !ok {
		return nil, ErrDashboardNotFound
	}
	return &dashboard, nil
}

// RequestExport creates a stub export job response.
func (s *QueryService) RequestExport(ctx context.Context, req *models.ExportRequest) (*models.ExportResponse, error) {
	_ = ctx
	expires := time.Now().UTC().Add(1 * time.Hour)
	return &models.ExportResponse{
		ExportID:  generateID(),
		Status:    "pending",
		ExpiresAt: expires,
	}, nil
}

// Health compiles health metadata for the service.
func (s *QueryService) Health(ctx context.Context) *models.HealthResponse {
	_ = ctx
	uptime := time.Since(s.startedAt).Seconds()
	return &models.HealthResponse{
		Status:        "healthy",
		Version:       s.version,
		UptimeSeconds: int64(uptime),
	}
}

func generateID() string {
	buf := make([]byte, 16)
	if _, err := crypto_rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return formatUUID(buf)
}

func formatUUID(b []byte) string {
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexBytes := make([]byte, 36)
	hex.Encode(hexBytes[0:8], b[0:4])
	hexBytes[8] = '-'
	hex.Encode(hexBytes[9:13], b[4:6])
	hexBytes[13] = '-'
	hex.Encode(hexBytes[14:18], b[6:8])
	hexBytes[18] = '-'
	hex.Encode(hexBytes[19:23], b[8:10])
	hexBytes[23] = '-'
	hex.Encode(hexBytes[24:], b[10:16])
	return string(hexBytes)
}
