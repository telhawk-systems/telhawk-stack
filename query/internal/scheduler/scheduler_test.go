package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
)

type mockExecutor struct {
	results []map[string]interface{}
	err     error
}

func (m *mockExecutor) ExecuteSearch(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &models.SearchResponse{
		RequestID:   "test-req",
		LatencyMS:   10,
		ResultCount: len(m.results),
		Results:     m.results,
	}, nil
}

type mockStore struct {
	mu                sync.RWMutex
	alerts            []models.Alert
	lastTriggeredTime map[string]time.Time
}

func (m *mockStore) ListAlerts(ctx context.Context) (*models.AlertListResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	alertsCopy := make([]models.Alert, len(m.alerts))
	copy(alertsCopy, m.alerts)
	return &models.AlertListResponse{Alerts: alertsCopy}, nil
}

func (m *mockStore) UpdateLastTriggered(ctx context.Context, alertID string, timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastTriggeredTime == nil {
		m.lastTriggeredTime = make(map[string]time.Time)
	}
	m.lastTriggeredTime[alertID] = timestamp
	return nil
}

type mockChannel struct {
	mu    sync.Mutex
	calls int
	last  *models.Alert
}

func (m *mockChannel) Send(ctx context.Context, alert *models.Alert, results []map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.last = alert
	return nil
}

func (m *mockChannel) Type() string {
	return "mock"
}

func (m *mockChannel) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestSchedulerStartStop(t *testing.T) {
	executor := &mockExecutor{}
	store := &mockStore{}
	channel := &mockChannel{}

	s := NewScheduler(executor, store, channel, Config{
		CheckInterval: 100 * time.Millisecond,
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if err := s.Stop(); err != nil {
		t.Fatalf("failed to stop scheduler: %v", err)
	}

	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to restart scheduler: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("failed to stop scheduler: %v", err)
	}
}

func TestSchedulerExecutesActiveAlerts(t *testing.T) {
	executor := &mockExecutor{
		results: []map[string]interface{}{
			{"message": "test event"},
		},
	}

	now := time.Now().UTC()
	store := &mockStore{
		alerts: []models.Alert{
			{
				ID:       "alert-1",
				Name:     "Test Alert",
				Query:    "test query",
				Severity: "high",
				Status:   "active",
				Schedule: models.AlertSchedule{
					IntervalMinutes: 0,
					LookbackMinutes: 5,
				},
			},
		},
		lastTriggeredTime: make(map[string]time.Time),
	}

	channel := &mockChannel{}

	s := NewScheduler(executor, store, channel, Config{
		CheckInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	time.Sleep(6 * time.Second)

	if channel.CallCount() == 0 {
		t.Error("expected alert to be triggered at least once")
	}

	store.mu.RLock()
	triggered, ok := store.lastTriggeredTime["alert-1"]
	store.mu.RUnlock()

	if !ok {
		t.Error("expected last triggered time to be updated")
	} else if triggered.Before(now) {
		t.Error("last triggered time should be after test start")
	}
}

func TestSchedulerIgnoresInactiveAlerts(t *testing.T) {
	executor := &mockExecutor{
		results: []map[string]interface{}{
			{"message": "test event"},
		},
	}

	store := &mockStore{
		alerts: []models.Alert{
			{
				ID:       "alert-1",
				Name:     "Inactive Alert",
				Query:    "test query",
				Severity: "high",
				Status:   "paused",
				Schedule: models.AlertSchedule{
					IntervalMinutes: 1,
					LookbackMinutes: 5,
				},
			},
		},
	}

	channel := &mockChannel{}

	s := NewScheduler(executor, store, channel, Config{
		CheckInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	time.Sleep(1200 * time.Millisecond)

	if channel.CallCount() > 0 {
		t.Errorf("expected no notifications for inactive alert, got %d", channel.CallCount())
	}
}

func TestSchedulerDoesNotTriggerOnZeroResults(t *testing.T) {
	executor := &mockExecutor{
		results: []map[string]interface{}{},
	}

	store := &mockStore{
		alerts: []models.Alert{
			{
				ID:       "alert-1",
				Name:     "Test Alert",
				Query:    "test query",
				Severity: "high",
				Status:   "active",
				Schedule: models.AlertSchedule{
					IntervalMinutes: 1,
					LookbackMinutes: 5,
				},
			},
		},
	}

	channel := &mockChannel{}

	s := NewScheduler(executor, store, channel, Config{
		CheckInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	time.Sleep(1200 * time.Millisecond)

	if channel.CallCount() > 0 {
		t.Errorf("expected no notifications when query returns zero results, got %d", channel.CallCount())
	}
}

func TestSchedulerMetrics(t *testing.T) {
	executor := &mockExecutor{
		results: []map[string]interface{}{
			{"message": "test event"},
		},
	}

	store := &mockStore{
		alerts: []models.Alert{
			{
				ID:       "alert-1",
				Name:     "Test Alert",
				Query:    "test query",
				Severity: "high",
				Status:   "active",
				Schedule: models.AlertSchedule{
					IntervalMinutes: 0,
					LookbackMinutes: 5,
				},
			},
		},
		lastTriggeredTime: make(map[string]time.Time),
	}

	channel := &mockChannel{}

	s := NewScheduler(executor, store, channel, Config{
		CheckInterval: 50 * time.Millisecond,
	})

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}
	defer s.Stop()

	time.Sleep(6 * time.Second)

	metrics := s.GetMetrics()

	if metrics["active_alert_count"].(int) != 1 {
		t.Errorf("expected 1 active alert, got %d", metrics["active_alert_count"])
	}

	if metrics["alert_executions"].(int64) == 0 {
		t.Error("expected alert executions > 0")
	}

	if metrics["alerts_triggered"].(int64) == 0 {
		t.Error("expected alerts triggered > 0")
	}

	if metrics["notifications_sent"].(int64) == 0 {
		t.Error("expected notifications sent > 0")
	}
}
