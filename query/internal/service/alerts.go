package service

import (
	"context"
	"sort"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

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

// UpdateLastTriggered updates the last triggered timestamp for an alert.
func (s *QueryService) UpdateLastTriggered(ctx context.Context, alertID string, timestamp time.Time) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	alert, ok := s.alerts[alertID]
	if !ok {
		return ErrAlertNotFound
	}
	alert.LastTriggeredAt = &timestamp
	s.alerts[alertID] = alert
	return nil
}
