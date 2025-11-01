package service

import (
	context "context"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

var (
	ErrAlertNotFound     = errors.New("alert not found")
	ErrDashboardNotFound = errors.New("dashboard not found")
)

// QueryService provides stubbed implementations for the query API surface.
type QueryService struct {
	mu         sync.RWMutex
	startedAt  time.Time
	version    string
	alerts     map[string]models.Alert
	dashboards map[string]models.Dashboard
}

// NewQueryService seeds in-memory data used by the HTTP handlers.
func NewQueryService(version string) *QueryService {
	now := time.Now().UTC()
	return &QueryService{
		startedAt: now,
		version:   version,
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

// ExecuteSearch returns canned results representing a query invocation.
func (s *QueryService) ExecuteSearch(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	_ = ctx
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	results := []map[string]interface{}{
		{
			"event_time": time.Now().UTC().Add(-45 * time.Minute).Format(time.RFC3339),
			"log_source": "edr",
			"class":      "malware",
			"severity":   "high",
			"host":       "workstation-17",
		},
		{
			"event_time": time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339),
			"log_source": "edr",
			"class":      "malware",
			"severity":   "medium",
			"host":       "workstation-22",
		},
	}
	if limit < len(results) {
		results = results[:limit]
	}
	resp := &models.SearchResponse{
		RequestID:   generateID(),
		LatencyMS:   42,
		ResultCount: len(results),
		Results:     results,
	}
	return resp, nil
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
