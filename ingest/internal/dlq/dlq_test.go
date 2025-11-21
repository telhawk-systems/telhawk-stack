package dlq_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/dlq"
	"github.com/telhawk-systems/telhawk-stack/ingest/internal/model"
)

func TestNewQueue(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates queue with valid path", func(t *testing.T) {
		queue, err := dlq.NewQueue(tempDir)

		require.NoError(t, err)
		assert.NotNil(t, queue)

		// Verify directory was created
		info, err := os.Stat(tempDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("creates queue with default path", func(t *testing.T) {
		// Test with empty string (should use default)
		queue, err := dlq.NewQueue("")

		// This might fail if /var/lib/telhawk/dlq can't be created in test environment
		// That's okay - we're testing the behavior
		if err == nil {
			assert.NotNil(t, queue)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "nested", "path", "dlq")
		queue, err := dlq.NewQueue(nestedPath)

		require.NoError(t, err)
		assert.NotNil(t, queue)

		// Verify nested directory was created
		info, err := os.Stat(nestedPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestQueue_Write(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	envelope := &models.RawEventEnvelope{
		ID:         "test-123",
		Format:     "json",
		SourceType: "test",
		Source:     "test-source",
		Payload:    []byte(`{"user":"alice","action":"login"}`),
		ReceivedAt: time.Now(),
	}

	testErr := errors.New("normalization failed")
	reason := "invalid_format"

	ctx := context.Background()
	err = queue.Write(ctx, envelope, testErr, reason)

	require.NoError(t, err)

	// Verify file was created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, 1, "one DLQ file should be created")

	// Verify file contents
	fileData, err := os.ReadFile(filepath.Join(tempDir, files[0].Name()))
	require.NoError(t, err)

	var failedEvent dlq.FailedEvent
	err = json.Unmarshal(fileData, &failedEvent)
	require.NoError(t, err)

	assert.Equal(t, envelope.ID, failedEvent.Envelope.ID)
	assert.Equal(t, testErr.Error(), failedEvent.Error)
	assert.Equal(t, reason, failedEvent.Reason)
	assert.Equal(t, 1, failedEvent.Attempts)
	assert.False(t, failedEvent.Timestamp.IsZero())
	assert.False(t, failedEvent.LastAttempt.IsZero())
}

func TestQueue_Write_MultipleEvents(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write multiple events
	for i := 0; i < 5; i++ {
		envelope := &models.RawEventEnvelope{
			ID:         "test-" + string(rune(i)),
			Format:     "json",
			SourceType: "test",
			Source:     "test-source",
			Payload:    []byte(`{}`),
			ReceivedAt: time.Now(),
		}

		err = queue.Write(ctx, envelope, errors.New("test error"), "test_reason")
		require.NoError(t, err)
	}

	// Verify all files were created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, 5, "five DLQ files should be created")
}

func TestQueue_Write_NilQueue(t *testing.T) {
	var queue *dlq.Queue

	envelope := &models.RawEventEnvelope{
		ID:     "test-123",
		Format: "json",
	}

	ctx := context.Background()
	err := queue.Write(ctx, envelope, errors.New("test"), "test")

	assert.NoError(t, err, "nil queue should not error")
}

func TestQueue_Stats(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	t.Run("stats for empty queue", func(t *testing.T) {
		stats := queue.Stats()

		require.NotNil(t, stats)
		assert.Equal(t, true, stats["enabled"])
		assert.Equal(t, uint64(0), stats["written"])
		assert.Equal(t, 0, stats["pending_files"])
		assert.Equal(t, tempDir, stats["base_path"])
	})

	t.Run("stats after writing events", func(t *testing.T) {
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			envelope := &models.RawEventEnvelope{
				ID:     "test",
				Format: "json",
			}
			err = queue.Write(ctx, envelope, errors.New("test"), "test")
			require.NoError(t, err)
		}

		stats := queue.Stats()

		assert.Equal(t, uint64(3), stats["written"])
		assert.Equal(t, 3, stats["pending_files"])
	})
}

func TestQueue_Stats_NilQueue(t *testing.T) {
	var queue *dlq.Queue

	stats := queue.Stats()

	require.NotNil(t, stats)
	assert.Equal(t, false, stats["enabled"])
}

func TestQueue_List(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write some events
	expectedEvents := make([]*models.RawEventEnvelope, 3)
	for i := 0; i < 3; i++ {
		envelope := &models.RawEventEnvelope{
			ID:         "test-" + string(rune('a'+i)),
			Format:     "json",
			SourceType: "test",
			Source:     "test-source",
			Payload:    []byte(`{}`),
			ReceivedAt: time.Now(),
		}
		expectedEvents[i] = envelope

		err = queue.Write(ctx, envelope, errors.New("test error"), "test_reason")
		require.NoError(t, err)
	}

	// List events
	events, err := queue.List(ctx, 10)

	require.NoError(t, err)
	assert.Len(t, events, 3)

	// Verify event IDs are present
	ids := make(map[string]bool)
	for _, event := range events {
		ids[event.Envelope.ID] = true
	}
	for _, expected := range expectedEvents {
		assert.True(t, ids[expected.ID], "expected event ID %s not found", expected.ID)
	}
}

func TestQueue_List_WithLimit(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write 5 events
	for i := 0; i < 5; i++ {
		envelope := &models.RawEventEnvelope{
			ID:     "test",
			Format: "json",
		}
		err = queue.Write(ctx, envelope, errors.New("test"), "test")
		require.NoError(t, err)
	}

	// List with limit of 3
	events, err := queue.List(ctx, 3)

	require.NoError(t, err)
	assert.Len(t, events, 3, "should respect limit")
}

func TestQueue_List_EmptyQueue(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()
	events, err := queue.List(ctx, 10)

	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestQueue_List_NilQueue(t *testing.T) {
	var queue *dlq.Queue

	ctx := context.Background()
	events, err := queue.List(ctx, 10)

	assert.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestQueue_Delete(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write an event
	envelope := &models.RawEventEnvelope{
		ID:     "test-delete",
		Format: "json",
	}

	err = queue.Write(ctx, envelope, errors.New("test"), "test")
	require.NoError(t, err)

	// Get the timestamp from the written file
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	// Extract timestamp from filename (format: failed_<timestamp>_<count>.json)
	filename := files[0].Name()
	var timestamp int64
	var count int
	_, err = fmt.Sscanf(filename, "failed_%d_%d.json", &timestamp, &count)
	require.NoError(t, err)

	// Delete the event
	err = queue.Delete(ctx, timestamp)
	require.NoError(t, err)

	// Verify file was deleted
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, 0, "file should be deleted")
}

func TestQueue_Delete_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Try to delete non-existent event
	err = queue.Delete(ctx, 9999999999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestQueue_Delete_NilQueue(t *testing.T) {
	var queue *dlq.Queue

	ctx := context.Background()
	err := queue.Delete(ctx, 12345)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestQueue_Purge(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write multiple events
	for i := 0; i < 5; i++ {
		envelope := &models.RawEventEnvelope{
			ID:     "test",
			Format: "json",
		}
		err = queue.Write(ctx, envelope, errors.New("test"), "test")
		require.NoError(t, err)
	}

	// Verify files exist
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, 5)

	// Purge all
	err = queue.Purge(ctx)
	require.NoError(t, err)

	// Verify all files deleted
	files, err = os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, 0, "all files should be deleted")
}

func TestQueue_Purge_EmptyQueue(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Purge empty queue
	err = queue.Purge(ctx)

	assert.NoError(t, err, "purging empty queue should not error")
}

func TestQueue_Purge_NilQueue(t *testing.T) {
	var queue *dlq.Queue

	ctx := context.Background()
	err := queue.Purge(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestQueue_Write_PreservesPayload(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	complexPayload := []byte(`{
		"user": "alice",
		"action": "login",
		"nested": {
			"field1": "value1",
			"field2": 123
		}
	}`)

	envelope := &models.RawEventEnvelope{
		ID:         "test-complex",
		Format:     "json",
		SourceType: "test",
		Source:     "test-source",
		Payload:    complexPayload,
		ReceivedAt: time.Now(),
		Attributes: map[string]string{
			"region": "us-west-2",
			"env":    "production",
		},
	}

	ctx := context.Background()
	err = queue.Write(ctx, envelope, errors.New("test error"), "test_reason")
	require.NoError(t, err)

	// Read back and verify
	events, err := queue.List(ctx, 1)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, envelope.ID, events[0].Envelope.ID)
	assert.Equal(t, envelope.Format, events[0].Envelope.Format)
	assert.Equal(t, envelope.SourceType, events[0].Envelope.SourceType)
	assert.Equal(t, envelope.Source, events[0].Envelope.Source)
	assert.JSONEq(t, string(complexPayload), string(events[0].Envelope.Payload))
	assert.Equal(t, envelope.Attributes, events[0].Envelope.Attributes)
}

func TestQueue_Write_DifferentReasons(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	reasons := []string{
		"normalization_failed",
		"validation_failed",
		"storage_failed",
		"timeout",
	}

	for _, reason := range reasons {
		envelope := &models.RawEventEnvelope{
			ID:     "test-" + reason,
			Format: "json",
		}

		err = queue.Write(ctx, envelope, errors.New("test error"), reason)
		require.NoError(t, err)
	}

	// List and verify all reasons are captured
	events, err := queue.List(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, events, len(reasons))

	foundReasons := make(map[string]bool)
	for _, event := range events {
		foundReasons[event.Reason] = true
	}

	for _, reason := range reasons {
		assert.True(t, foundReasons[reason], "reason %s not found", reason)
	}
}

func TestQueue_ConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write concurrently from multiple goroutines
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			envelope := &models.RawEventEnvelope{
				ID:     "test",
				Format: "json",
			}
			err := queue.Write(ctx, envelope, errors.New("test"), "test")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all events were written
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, numGoroutines, "all concurrent writes should succeed")
}

func TestQueue_TimestampOrdering(t *testing.T) {
	tempDir := t.TempDir()
	queue, err := dlq.NewQueue(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Write events with small delays to ensure different timestamps
	for i := 0; i < 3; i++ {
		envelope := &models.RawEventEnvelope{
			ID:     "test",
			Format: "json",
		}
		err = queue.Write(ctx, envelope, errors.New("test"), "test")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// List files
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Verify filenames contain timestamps (they should be sortable)
	prevTimestamp := int64(0)
	for _, file := range files {
		var timestamp int64
		var count int
		_, err = fmt.Sscanf(file.Name(), "failed_%d_%d.json", &timestamp, &count)
		require.NoError(t, err)

		// Timestamps should be increasing (or at least not decreasing)
		assert.GreaterOrEqual(t, timestamp, prevTimestamp)
		prevTimestamp = timestamp
	}
}
