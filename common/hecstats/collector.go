package hecstats

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"
)

// Collector accumulates HEC token usage stats and flushes to Redis periodically.
// Safe for concurrent use from multiple goroutines.
type Collector struct {
	client        *Client
	flushInterval time.Duration
	logger        *slog.Logger

	mu      sync.Mutex
	batches map[string]*BatchUpdate // tokenID -> batch

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCollector creates a new stats collector that flushes to Redis periodically.
func NewCollector(client *Client, flushInterval time.Duration, logger *slog.Logger) *Collector {
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Collector{
		client:        client,
		flushInterval: flushInterval,
		logger:        logger,
		batches:       make(map[string]*BatchUpdate),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start background flush goroutine
	c.wg.Add(1)
	go c.flushLoop()

	return c
}

// Record accumulates a usage event for later batch flushing.
func (c *Collector) Record(tokenID string, eventCount int64, clientIP net.IP) {
	c.mu.Lock()
	defer c.mu.Unlock()

	batch, ok := c.batches[tokenID]
	if !ok {
		batch = NewBatchUpdate(tokenID)
		c.batches[tokenID] = batch
	}

	batch.Add(eventCount, clientIP)
}

// flushLoop runs in the background and flushes accumulated stats periodically.
func (c *Collector) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Final flush on shutdown
			c.flush()
			return
		case <-ticker.C:
			c.flush()
		}
	}
}

// flush writes all accumulated batches to Redis.
func (c *Collector) flush() {
	c.mu.Lock()
	// Swap out the batches map so we can release the lock quickly
	batches := c.batches
	c.batches = make(map[string]*BatchUpdate)
	c.mu.Unlock()

	if len(batches) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	flushed := 0
	totalEvents := int64(0)

	for _, batch := range batches {
		if err := c.client.FlushBatch(ctx, batch); err != nil {
			c.logger.Error("failed to flush HEC stats batch",
				"token_id", batch.TokenID,
				"event_count", batch.EventCount,
				"error", err,
			)
			// Re-add failed batch for retry (merge back)
			c.mu.Lock()
			if existing, ok := c.batches[batch.TokenID]; ok {
				existing.EventCount += batch.EventCount
				for ip := range batch.ClientIPs {
					existing.ClientIPs[ip] = struct{}{}
				}
				if batch.LastIP != nil {
					existing.LastIP = batch.LastIP
				}
			} else {
				c.batches[batch.TokenID] = batch
			}
			c.mu.Unlock()
		} else {
			flushed++
			totalEvents += batch.EventCount
		}
	}

	if flushed > 0 {
		c.logger.Debug("flushed HEC stats",
			"tokens", flushed,
			"total_events", totalEvents,
		)
	}
}

// FlushNow forces an immediate flush of all accumulated stats.
func (c *Collector) FlushNow() {
	c.flush()
}

// Stop stops the collector and flushes any remaining stats.
func (c *Collector) Stop() {
	c.cancel()
	c.wg.Wait()
}

// Stats returns the current accumulated stats (not yet flushed).
// Useful for debugging/monitoring.
func (c *Collector) Stats() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := make(map[string]int64, len(c.batches))
	for tokenID, batch := range c.batches {
		stats[tokenID] = batch.EventCount
	}
	return stats
}
