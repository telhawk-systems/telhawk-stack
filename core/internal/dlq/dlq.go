package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/model"
)

// FailedEvent captures normalization failure details for replay.
type FailedEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Envelope    *model.RawEventEnvelope `json:"envelope"`
	Error       string                 `json:"error"`
	Reason      string                 `json:"reason"`
	Attempts    int                    `json:"attempts"`
	LastAttempt time.Time              `json:"last_attempt"`
}

// Queue writes failed normalization events to disk for later analysis/replay.
type Queue struct {
	basePath string
	mu       sync.Mutex
	written  uint64
}

// NewQueue creates a DLQ that writes to the specified directory.
func NewQueue(basePath string) (*Queue, error) {
	if basePath == "" {
		basePath = "/var/lib/telhawk/dlq"
	}
	
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create dlq directory: %w", err)
	}
	
	return &Queue{
		basePath: basePath,
	}, nil
}

// Write records a failed event to the dead-letter queue.
func (q *Queue) Write(ctx context.Context, envelope *model.RawEventEnvelope, err error, reason string) error {
	if q == nil {
		return nil
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	failed := FailedEvent{
		Timestamp:   time.Now().UTC(),
		Envelope:    envelope,
		Error:       err.Error(),
		Reason:      reason,
		Attempts:    1,
		LastAttempt: time.Now().UTC(),
	}
	
	// Create timestamped filename
	filename := fmt.Sprintf("failed_%d_%d.json",
		time.Now().Unix(),
		q.written,
	)
	filePath := filepath.Join(q.basePath, filename)
	
	data, marshalErr := json.MarshalIndent(failed, "", "  ")
	if marshalErr != nil {
		log.Printf("ERROR: failed to marshal DLQ entry: %v", marshalErr)
		return marshalErr
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("ERROR: failed to write DLQ entry: %v", err)
		return err
	}
	
	q.written++
	log.Printf("DLQ: wrote failed event to %s (reason: %s)", filename, reason)
	
	return nil
}

// Stats returns DLQ metrics.
func (q *Queue) Stats() map[string]interface{} {
	if q == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	// Count files in directory
	files, err := os.ReadDir(q.basePath)
	if err != nil {
		log.Printf("ERROR: failed to read DLQ directory: %v", err)
		return map[string]interface{}{
			"enabled":       true,
			"written":       q.written,
			"pending_files": 0,
			"error":         err.Error(),
		}
	}
	
	return map[string]interface{}{
		"enabled":       true,
		"written":       q.written,
		"pending_files": len(files),
		"base_path":     q.basePath,
	}
}

// List returns all failed events in the queue.
func (q *Queue) List(ctx context.Context, limit int) ([]FailedEvent, error) {
	if q == nil {
		return nil, fmt.Errorf("dlq not enabled")
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	files, err := os.ReadDir(q.basePath)
	if err != nil {
		return nil, fmt.Errorf("read dlq directory: %w", err)
	}
	
	var events []FailedEvent
	count := 0
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		if limit > 0 && count >= limit {
			break
		}
		
		filePath := filepath.Join(q.basePath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("ERROR: failed to read DLQ file %s: %v", file.Name(), err)
			continue
		}
		
		var failed FailedEvent
		if err := json.Unmarshal(data, &failed); err != nil {
			log.Printf("ERROR: failed to parse DLQ file %s: %v", file.Name(), err)
			continue
		}
		
		events = append(events, failed)
		count++
	}
	
	return events, nil
}

// Delete removes a failed event from the queue.
func (q *Queue) Delete(ctx context.Context, timestamp int64) error {
	if q == nil {
		return fmt.Errorf("dlq not enabled")
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	// Find file with matching timestamp
	pattern := filepath.Join(q.basePath, fmt.Sprintf("failed_%d_*.json", timestamp))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("search dlq files: %w", err)
	}
	
	if len(matches) == 0 {
		return fmt.Errorf("event not found")
	}
	
	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return fmt.Errorf("delete dlq file: %w", err)
		}
		log.Printf("DLQ: deleted %s", filepath.Base(match))
	}
	
	return nil
}

// Purge removes all events from the queue.
func (q *Queue) Purge(ctx context.Context) error {
	if q == nil {
		return fmt.Errorf("dlq not enabled")
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	files, err := os.ReadDir(q.basePath)
	if err != nil {
		return fmt.Errorf("read dlq directory: %w", err)
	}
	
	deleted := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		filePath := filepath.Join(q.basePath, file.Name())
		if err := os.Remove(filePath); err != nil {
			log.Printf("ERROR: failed to delete DLQ file %s: %v", file.Name(), err)
			continue
		}
		deleted++
	}
	
	log.Printf("DLQ: purged %d events", deleted)
	return nil
}
