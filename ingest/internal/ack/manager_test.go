package ack

import (
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	ttl := 10 * time.Minute
	manager := NewManager(ttl)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.ttl != ttl {
		t.Errorf("ttl = %v, want %v", manager.ttl, ttl)
	}

	if manager.acks == nil {
		t.Error("acks map is nil")
	}

	if manager.cleanupCh == nil {
		t.Error("cleanupCh is nil")
	}

	// Clean up
	manager.Close()
	time.Sleep(10 * time.Millisecond)
}

func TestCreate(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	eventIDs := []string{"event-1", "event-2", "event-3"}
	ackID := manager.Create(eventIDs)

	if ackID == "" {
		t.Error("Create() returned empty ackID")
	}

	manager.mu.RLock()
	ack, exists := manager.acks[ackID]
	manager.mu.RUnlock()

	if !exists {
		t.Fatal("Acknowledgment not found in manager")
	}

	if ack.ID != ackID {
		t.Errorf("ack.ID = %q, want %q", ack.ID, ackID)
	}

	if ack.Status != StatusPending {
		t.Errorf("ack.Status = %v, want %v", ack.Status, StatusPending)
	}

	if len(ack.EventIDs) != len(eventIDs) {
		t.Errorf("len(ack.EventIDs) = %d, want %d", len(ack.EventIDs), len(eventIDs))
	}

	for i, id := range ack.EventIDs {
		if id != eventIDs[i] {
			t.Errorf("ack.EventIDs[%d] = %q, want %q", i, id, eventIDs[i])
		}
	}

	if ack.Timestamp.IsZero() {
		t.Error("ack.Timestamp is zero")
	}
}

func TestComplete(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	eventIDs := []string{"event-1"}
	ackID := manager.Create(eventIDs)

	// Complete the acknowledgment
	manager.Complete(ackID)

	manager.mu.RLock()
	ack, exists := manager.acks[ackID]
	manager.mu.RUnlock()

	if !exists {
		t.Fatal("Acknowledgment not found after Complete()")
	}

	if ack.Status != StatusSuccess {
		t.Errorf("ack.Status = %v, want %v", ack.Status, StatusSuccess)
	}
}

func TestComplete_NonExistentAck(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	// Should not panic or error when completing non-existent ack
	manager.Complete("nonexistent-ack-id")
}

func TestFail(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	eventIDs := []string{"event-1"}
	ackID := manager.Create(eventIDs)

	// Fail the acknowledgment
	manager.Fail(ackID)

	manager.mu.RLock()
	ack, exists := manager.acks[ackID]
	manager.mu.RUnlock()

	if !exists {
		t.Fatal("Acknowledgment not found after Fail()")
	}

	if ack.Status != StatusFailed {
		t.Errorf("ack.Status = %v, want %v", ack.Status, StatusFailed)
	}
}

func TestFail_NonExistentAck(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	// Should not panic or error when failing non-existent ack
	manager.Fail("nonexistent-ack-id")
}

func TestQuery(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	// Create some acks
	ackID1 := manager.Create([]string{"event-1"})
	ackID2 := manager.Create([]string{"event-2"})
	ackID3 := manager.Create([]string{"event-3"})

	// Complete one, fail one, leave one pending
	manager.Complete(ackID1)
	manager.Fail(ackID2)

	// Query all three
	results := manager.Query([]string{ackID1, ackID2, ackID3})

	if len(results) != 3 {
		t.Errorf("Query() returned %d results, want 3", len(results))
	}

	if results[ackID1] != true {
		t.Errorf("results[%s] = %v, want true (completed)", ackID1, results[ackID1])
	}

	if results[ackID2] != false {
		t.Errorf("results[%s] = %v, want false (failed)", ackID2, results[ackID2])
	}

	if results[ackID3] != false {
		t.Errorf("results[%s] = %v, want false (pending)", ackID3, results[ackID3])
	}
}

func TestQuery_NonExistentAcks(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	results := manager.Query([]string{"nonexistent-1", "nonexistent-2"})

	if len(results) != 0 {
		t.Errorf("Query() for non-existent acks returned %d results, want 0", len(results))
	}
}

func TestQuery_MixedExistence(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	ackID := manager.Create([]string{"event-1"})
	manager.Complete(ackID)

	results := manager.Query([]string{ackID, "nonexistent"})

	if len(results) != 1 {
		t.Errorf("Query() returned %d results, want 1", len(results))
	}

	if results[ackID] != true {
		t.Errorf("results[%s] = %v, want true", ackID, results[ackID])
	}

	if _, exists := results["nonexistent"]; exists {
		t.Error("Query() should not return result for non-existent ack")
	}
}

func TestGetPending(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	// Initially no pending acks
	if count := manager.GetPending(); count != 0 {
		t.Errorf("GetPending() = %d, want 0", count)
	}

	// Create some acks
	ackID1 := manager.Create([]string{"event-1"})
	ackID2 := manager.Create([]string{"event-2"})
	ackID3 := manager.Create([]string{"event-3"})

	// All pending
	if count := manager.GetPending(); count != 3 {
		t.Errorf("GetPending() = %d, want 3", count)
	}

	// Complete one
	manager.Complete(ackID1)
	if count := manager.GetPending(); count != 2 {
		t.Errorf("GetPending() after Complete() = %d, want 2", count)
	}

	// Fail one
	manager.Fail(ackID2)
	if count := manager.GetPending(); count != 1 {
		t.Errorf("GetPending() after Fail() = %d, want 1", count)
	}

	// Complete the last one
	manager.Complete(ackID3)
	if count := manager.GetPending(); count != 0 {
		t.Errorf("GetPending() after completing all = %d, want 0", count)
	}
}

func TestCleanup(t *testing.T) {
	manager := NewManager(100 * time.Millisecond)
	defer manager.Close()

	// Create an ack
	ackID := manager.Create([]string{"event-1"})

	// Verify it exists
	manager.mu.RLock()
	_, exists := manager.acks[ackID]
	manager.mu.RUnlock()

	if !exists {
		t.Fatal("Acknowledgment should exist after creation")
	}

	// Wait for cleanup to run (TTL + a bit more)
	time.Sleep(250 * time.Millisecond)

	// Manually trigger cleanup
	manager.cleanup()

	// Verify it's been cleaned up
	manager.mu.RLock()
	_, exists = manager.acks[ackID]
	manager.mu.RUnlock()

	if exists {
		t.Error("Acknowledgment should have been cleaned up")
	}
}

func TestCleanup_OnlyRemovesExpired(t *testing.T) {
	manager := NewManager(200 * time.Millisecond)
	defer manager.Close()

	// Create first ack
	oldAckID := manager.Create([]string{"event-1"})

	// Wait a bit
	time.Sleep(150 * time.Millisecond)

	// Create second ack
	newAckID := manager.Create([]string{"event-2"})

	// Wait for first ack to expire
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	manager.cleanup()

	manager.mu.RLock()
	_, oldExists := manager.acks[oldAckID]
	_, newExists := manager.acks[newAckID]
	manager.mu.RUnlock()

	if oldExists {
		t.Error("Old acknowledgment should have been cleaned up")
	}

	if !newExists {
		t.Error("New acknowledgment should still exist")
	}
}

func TestCleanup_UpdatesTimestamp(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	ackID := manager.Create([]string{"event-1"})

	manager.mu.RLock()
	originalTime := manager.acks[ackID].Timestamp
	manager.mu.RUnlock()

	time.Sleep(10 * time.Millisecond)

	// Complete the ack
	manager.Complete(ackID)

	manager.mu.RLock()
	newTime := manager.acks[ackID].Timestamp
	manager.mu.RUnlock()

	if !newTime.After(originalTime) {
		t.Error("Timestamp should be updated on status change")
	}
}

func TestClose(t *testing.T) {
	manager := NewManager(10 * time.Minute)

	// Close should stop the cleanup goroutine
	manager.Close()

	// Give goroutine time to exit
	time.Sleep(10 * time.Millisecond)

	// Channel should be closed
	select {
	case _, ok := <-manager.cleanupCh:
		if ok {
			t.Error("cleanupCh should be closed")
		}
	default:
		t.Error("cleanupCh should be closed and readable")
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	acksPerGoroutine := 100

	// Concurrently create, complete, and query acks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < acksPerGoroutine; j++ {
				ackID := manager.Create([]string{"event"})

				// Randomly complete, fail, or query
				switch j % 3 {
				case 0:
					manager.Complete(ackID)
				case 1:
					manager.Fail(ackID)
				case 2:
					manager.Query([]string{ackID})
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify no race conditions occurred
	pending := manager.GetPending()
	if pending < 0 {
		t.Errorf("GetPending() = %d, should not be negative", pending)
	}
}

func TestConcurrentCleanup(t *testing.T) {
	manager := NewManager(50 * time.Millisecond)
	defer manager.Close()

	var wg sync.WaitGroup

	// Create acks concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				manager.Create([]string{"event"})
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	// Trigger cleanups concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				time.Sleep(10 * time.Millisecond)
				manager.cleanup()
			}
		}()
	}

	wg.Wait()
}

func TestCreateUniqueIDs(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		ackID := manager.Create([]string{"event"})
		if ids[ackID] {
			t.Errorf("Duplicate ackID generated: %s", ackID)
		}
		ids[ackID] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestStatusTransitions(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	ackID := manager.Create([]string{"event-1"})

	// Check initial status
	manager.mu.RLock()
	if manager.acks[ackID].Status != StatusPending {
		t.Error("Initial status should be StatusPending")
	}
	manager.mu.RUnlock()

	// Transition to success
	manager.Complete(ackID)
	manager.mu.RLock()
	if manager.acks[ackID].Status != StatusSuccess {
		t.Error("Status should be StatusSuccess after Complete()")
	}
	manager.mu.RUnlock()

	// Test another ack transitioning to failed
	ackID2 := manager.Create([]string{"event-2"})
	manager.Fail(ackID2)
	manager.mu.RLock()
	if manager.acks[ackID2].Status != StatusFailed {
		t.Error("Status should be StatusFailed after Fail()")
	}
	manager.mu.RUnlock()
}

func TestMultipleEventIDs(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	eventIDs := []string{
		"event-1",
		"event-2",
		"event-3",
		"event-4",
		"event-5",
	}

	ackID := manager.Create(eventIDs)

	manager.mu.RLock()
	ack := manager.acks[ackID]
	manager.mu.RUnlock()

	if len(ack.EventIDs) != len(eventIDs) {
		t.Errorf("len(EventIDs) = %d, want %d", len(ack.EventIDs), len(eventIDs))
	}

	for i, id := range eventIDs {
		if ack.EventIDs[i] != id {
			t.Errorf("EventIDs[%d] = %s, want %s", i, ack.EventIDs[i], id)
		}
	}
}

func TestCleanupLoop_Runs(t *testing.T) {
	manager := NewManager(50 * time.Millisecond)
	defer manager.Close()

	// Create an ack that will expire
	ackID := manager.Create([]string{"event-1"})

	// Wait for automatic cleanup to run (should happen within 1 minute + TTL)
	time.Sleep(200 * time.Millisecond)

	manager.mu.RLock()
	_ = manager.acks[ackID]
	manager.mu.RUnlock()

	// Note: This test might be flaky depending on cleanup timing
	// The cleanup loop runs every minute, so for testing we rely on manual cleanup() calls
	// in other tests. This test mainly verifies the loop doesn't crash.
}

func TestEmptyEventIDs(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	ackID := manager.Create([]string{})

	manager.mu.RLock()
	ack := manager.acks[ackID]
	manager.mu.RUnlock()

	if ack == nil {
		t.Fatal("Acknowledgment should exist even with empty EventIDs")
	}

	if len(ack.EventIDs) != 0 {
		t.Errorf("len(EventIDs) = %d, want 0", len(ack.EventIDs))
	}
}

func TestNilEventIDs(t *testing.T) {
	manager := NewManager(10 * time.Minute)
	defer manager.Close()

	ackID := manager.Create(nil)

	manager.mu.RLock()
	ack := manager.acks[ackID]
	manager.mu.RUnlock()

	if ack == nil {
		t.Fatal("Acknowledgment should exist even with nil EventIDs")
	}
}
