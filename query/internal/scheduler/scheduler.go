package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/telhawk-systems/telhawk-stack/query/internal/models"
	"github.com/telhawk-systems/telhawk-stack/query/internal/notification"
)

// AlertExecutor defines the interface for executing alert queries.
type AlertExecutor interface {
	ExecuteSearch(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error)
}

// AlertStore defines the interface for retrieving and updating alerts.
type AlertStore interface {
	ListAlerts(ctx context.Context) (*models.AlertListResponse, error)
	UpdateLastTriggered(ctx context.Context, alertID string, timestamp time.Time) error
}

// Scheduler manages periodic execution of alerts and notification delivery.
type Scheduler struct {
	mu            sync.RWMutex
	executor      AlertExecutor
	store         AlertStore
	channel       notification.Channel
	running       bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	checkInterval time.Duration

	alertTimers map[string]*alertTimer
	metrics     *Metrics
}

type alertTimer struct {
	alertID  string
	ticker   *time.Ticker
	stopChan chan struct{}
}

// Metrics tracks scheduler performance and alert execution stats.
type Metrics struct {
	mu                 sync.RWMutex
	AlertsTriggered    int64
	AlertExecutions    int64
	AlertErrors        int64
	NotificationsSent  int64
	NotificationErrors int64
	LastCheckTime      time.Time
}

// Config configures the alert scheduler.
type Config struct {
	CheckInterval time.Duration
}

// NewScheduler creates a new alert scheduler.
func NewScheduler(executor AlertExecutor, store AlertStore, channel notification.Channel, cfg Config) *Scheduler {
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 30 * time.Second
	}

	return &Scheduler{
		executor:      executor,
		store:         store,
		channel:       channel,
		checkInterval: cfg.CheckInterval,
		alertTimers:   make(map[string]*alertTimer),
		metrics:       &Metrics{},
	}
}

// Start begins the alert scheduling loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	log.Printf("alert scheduler starting (check interval: %s)", s.checkInterval)

	s.wg.Add(1)
	go s.run(ctx)

	return nil
}

// Stop gracefully stops the alert scheduler.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	s.stopAllAlertTimers()

	s.wg.Wait()
	log.Printf("alert scheduler stopped")
	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	s.syncAlerts(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.syncAlerts(ctx)
		}
	}
}

func (s *Scheduler) syncAlerts(ctx context.Context) {
	s.metrics.mu.Lock()
	s.metrics.LastCheckTime = time.Now()
	s.metrics.mu.Unlock()

	alerts, err := s.store.ListAlerts(ctx)
	if err != nil {
		log.Printf("failed to list alerts: %v", err)
		return
	}

	activeAlertIDs := make(map[string]bool)

	for _, alert := range alerts.Alerts {
		if alert.Status != "active" {
			continue
		}

		activeAlertIDs[alert.ID] = true

		s.mu.RLock()
		_, exists := s.alertTimers[alert.ID]
		s.mu.RUnlock()

		if !exists {
			s.scheduleAlert(ctx, &alert)
		}
	}

	s.mu.Lock()
	for alertID, timer := range s.alertTimers {
		if !activeAlertIDs[alertID] {
			timer.stop()
			delete(s.alertTimers, alertID)
		}
	}
	s.mu.Unlock()
}

func (s *Scheduler) scheduleAlert(ctx context.Context, alert *models.Alert) {
	interval := time.Duration(alert.Schedule.IntervalMinutes) * time.Minute
	if interval < 1*time.Minute {
		if alert.Schedule.IntervalMinutes == 0 && s.checkInterval < 1*time.Minute {
			interval = 5 * time.Second
		} else {
			log.Printf("alert %s has invalid interval %d minutes, using 5 minutes", alert.ID, alert.Schedule.IntervalMinutes)
			interval = 5 * time.Minute
		}
	}

	timer := &alertTimer{
		alertID:  alert.ID,
		ticker:   time.NewTicker(interval),
		stopChan: make(chan struct{}),
	}

	s.mu.Lock()
	s.alertTimers[alert.ID] = timer
	s.mu.Unlock()

	log.Printf("scheduled alert %s (%s) with interval %s", alert.ID, alert.Name, interval)

	s.wg.Add(1)
	go s.runAlert(ctx, timer)
}

func (s *Scheduler) runAlert(ctx context.Context, timer *alertTimer) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.stopChan:
			return
		case <-timer.ticker.C:
			s.executeAlert(ctx, timer.alertID)
		}
	}
}

func (s *Scheduler) executeAlert(ctx context.Context, alertID string) {
	s.metrics.mu.Lock()
	s.metrics.AlertExecutions++
	s.metrics.mu.Unlock()

	alerts, err := s.store.ListAlerts(ctx)
	if err != nil {
		log.Printf("failed to retrieve alert %s: %v", alertID, err)
		s.incrementErrors()
		return
	}

	var alert *models.Alert
	for _, a := range alerts.Alerts {
		if a.ID == alertID {
			alert = &a
			break
		}
	}

	if alert == nil {
		log.Printf("alert %s not found", alertID)
		s.incrementErrors()
		return
	}

	if alert.Status != "active" {
		return
	}

	now := time.Now().UTC()
	lookback := time.Duration(alert.Schedule.LookbackMinutes) * time.Minute
	if lookback == 0 {
		lookback = time.Duration(alert.Schedule.IntervalMinutes) * time.Minute
	}

	searchReq := &models.SearchRequest{
		Query: alert.Query,
		TimeRange: &models.TimeRange{
			From: now.Add(-lookback),
			To:   now,
		},
		Limit: 100,
	}

	resp, err := s.executor.ExecuteSearch(ctx, searchReq)
	if err != nil {
		log.Printf("failed to execute alert %s (%s): %v", alert.ID, alert.Name, err)
		s.incrementErrors()
		return
	}

	if resp.ResultCount == 0 {
		return
	}

	log.Printf("alert %s (%s) triggered with %d results", alert.ID, alert.Name, resp.ResultCount)

	s.metrics.mu.Lock()
	s.metrics.AlertsTriggered++
	s.metrics.mu.Unlock()

	if err := s.channel.Send(ctx, alert, resp.Results); err != nil {
		log.Printf("failed to send notification for alert %s: %v", alert.ID, err)
		s.metrics.mu.Lock()
		s.metrics.NotificationErrors++
		s.metrics.mu.Unlock()
		return
	}

	s.metrics.mu.Lock()
	s.metrics.NotificationsSent++
	s.metrics.mu.Unlock()

	if err := s.store.UpdateLastTriggered(ctx, alert.ID, now); err != nil {
		log.Printf("failed to update last triggered time for alert %s: %v", alert.ID, err)
	}
}

func (s *Scheduler) incrementErrors() {
	s.metrics.mu.Lock()
	s.metrics.AlertErrors++
	s.metrics.mu.Unlock()
}

func (s *Scheduler) stopAllAlertTimers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, timer := range s.alertTimers {
		timer.stop()
	}
	s.alertTimers = make(map[string]*alertTimer)
}

func (at *alertTimer) stop() {
	at.ticker.Stop()
	close(at.stopChan)
}

// GetMetrics returns a snapshot of scheduler metrics.
func (s *Scheduler) GetMetrics() map[string]interface{} {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return map[string]interface{}{
		"alerts_triggered":    s.metrics.AlertsTriggered,
		"alert_executions":    s.metrics.AlertExecutions,
		"alert_errors":        s.metrics.AlertErrors,
		"notifications_sent":  s.metrics.NotificationsSent,
		"notification_errors": s.metrics.NotificationErrors,
		"last_check_time":     s.metrics.LastCheckTime.Format(time.RFC3339),
		"active_alert_count":  len(s.alertTimers),
	}
}
