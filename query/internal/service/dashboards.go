package service

import (
	"context"
	"sort"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

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
