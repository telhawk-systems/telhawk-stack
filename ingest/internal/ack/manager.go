package ack

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/metrics"
)

type Status int

const (
	StatusPending Status = iota
	StatusSuccess
	StatusFailed
)

type Ack struct {
	ID        string
	Status    Status
	Timestamp time.Time
	EventIDs  []string
}

type Manager struct {
	acks      map[string]*Ack
	mu        sync.RWMutex
	ttl       time.Duration
	cleanupCh chan struct{}
}

func NewManager(ttl time.Duration) *Manager {
	m := &Manager{
		acks:      make(map[string]*Ack),
		ttl:       ttl,
		cleanupCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go m.cleanupLoop()

	return m
}

// Create creates a new acknowledgement for a batch of events
func (m *Manager) Create(eventIDs []string) string {
	ackID := uuid.New().String()

	ack := &Ack{
		ID:        ackID,
		Status:    StatusPending,
		Timestamp: time.Now(),
		EventIDs:  eventIDs,
	}

	m.mu.Lock()
	m.acks[ackID] = ack
	m.mu.Unlock()

	metrics.AcksPending.Inc()

	return ackID
}

// Complete marks an acknowledgement as successful
func (m *Manager) Complete(ackID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ack, exists := m.acks[ackID]; exists {
		ack.Status = StatusSuccess
		ack.Timestamp = time.Now()
		metrics.AcksPending.Dec()
		metrics.AcksCompleted.Inc()
	}
}

// Fail marks an acknowledgement as failed
func (m *Manager) Fail(ackID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ack, exists := m.acks[ackID]; exists {
		ack.Status = StatusFailed
		ack.Timestamp = time.Now()
		metrics.AcksPending.Dec()
	}
}

// Query queries the status of one or more acknowledgements
func (m *Manager) Query(ackIDs []string) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]bool)
	for _, ackID := range ackIDs {
		if ack, exists := m.acks[ackID]; exists {
			result[ackID] = ack.Status == StatusSuccess
		}
	}

	return result
}

// GetPending returns the number of pending acknowledgements
func (m *Manager) GetPending() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, ack := range m.acks {
		if ack.Status == StatusPending {
			count++
		}
	}
	return count
}

// cleanupLoop periodically removes expired acknowledgements
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.cleanupCh:
			return
		}
	}
}

func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.ttl)
	for id, ack := range m.acks {
		if ack.Timestamp.Before(cutoff) {
			if ack.Status == StatusPending {
				metrics.AcksPending.Dec()
			}
			delete(m.acks, id)
		}
	}
}

// Close stops the cleanup goroutine
func (m *Manager) Close() {
	close(m.cleanupCh)
}
