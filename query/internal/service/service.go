package service

import (
	context "context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/auth"
	"github.com/telhawk-systems/telhawk-stack/query/internal/client"
	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
	"github.com/telhawk-systems/telhawk-stack/query/internal/repository"
)

var (
	ErrAlertNotFound     = errors.New("alert not found")
	ErrDashboardNotFound = errors.New("dashboard not found")
	ErrSearchDisabled    = errors.New("search_disabled")
	ErrValidationFailed  = errors.New("validation failed")
)

// QueryService provides implementations for the query API surface.
type QueryService struct {
	mu         sync.RWMutex
	startedAt  time.Time
	version    string
	alerts     map[string]models.Alert
	dashboards map[string]models.Dashboard
	osClient   *client.OpenSearchClient

	// saved searches backing store + auth
	repo       *repository.PostgresRepository
	authClient *auth.Client
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

// WithDependencies wires the optional repo and auth client.
func (s *QueryService) WithDependencies(repo *repository.PostgresRepository, authClient *auth.Client) *QueryService {
	s.repo = repo
	s.authClient = authClient
	return s
}

// ValidateToken returns a durable user ID for a given bearer token.
func (s *QueryService) ValidateToken(ctx context.Context, token string) (string, error) {
	if s.authClient == nil {
		return "", fmt.Errorf("auth client not configured")
	}
	vr, err := s.authClient.Validate(ctx, token)
	if err != nil {
		return "", err
	}
	return vr.UserID, nil
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
