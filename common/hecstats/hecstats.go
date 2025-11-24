// Package hecstats provides Redis-backed HEC token usage statistics.
//
// Designed for multiple ingest instances writing concurrently.
// Stats are updated in real-time and can be read by any service (web, API, etc).
//
// Redis Key Structure:
//
//	hec:stats:{token_id}              - Hash with current stats
//	hec:hourly:{token_id}:{YYYYMMDDHH} - Event count for specific hour (expires 48h)
//	hec:daily:{token_id}:{YYYYMMDD}   - Event count for specific day (expires 7d)
//	hec:ips:{token_id}:{YYYYMMDD}     - Set of unique IPs for day (expires 7d)
//	hec:instances:{token_id}          - Hash of ingest instance -> last seen timestamp
package hecstats

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Stats represents current usage statistics for a HEC token.
type Stats struct {
	TokenID          string            `json:"token_id"`
	LastUsedAt       *time.Time        `json:"last_used_at,omitempty"`
	LastUsedIP       string            `json:"last_used_ip,omitempty"`
	TotalEvents      int64             `json:"total_events"`
	EventsLastHour   int64             `json:"events_last_hour"`
	EventsLast24h    int64             `json:"events_last_24h"`
	UniqueIPsToday   int64             `json:"unique_ips_today"`
	IngestInstances  map[string]string `json:"ingest_instances,omitempty"` // instance_id -> last_seen
	StatsRetrievedAt time.Time         `json:"stats_retrieved_at"`
}

// Client provides methods to record and retrieve HEC token statistics.
type Client struct {
	redis      *redis.Client
	instanceID string // Unique identifier for this ingest instance
}

// NewClient creates a new HEC stats client.
// instanceID should be unique per ingest instance (e.g., hostname, pod name, UUID).
func NewClient(redisURL string, instanceID string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Client{
		redis:      client,
		instanceID: instanceID,
	}, nil
}

// NewClientFromRedis creates a client from an existing Redis connection.
func NewClientFromRedis(client *redis.Client, instanceID string) *Client {
	return &Client{
		redis:      client,
		instanceID: instanceID,
	}
}

// RecordUsage records that a HEC token was used to ingest events.
// This is designed to be called frequently (every request) - Redis handles the load.
// For batch updates, use RecordBatchUsage instead.
func (c *Client) RecordUsage(ctx context.Context, tokenID string, eventCount int64, clientIP net.IP) error {
	now := time.Now()
	hourKey := now.Format("2006010215") // YYYYMMDDHH
	dayKey := now.Format("20060102")    // YYYYMMDD
	nowUnix := strconv.FormatInt(now.Unix(), 10)

	pipe := c.redis.Pipeline()

	// Main stats hash
	statsKey := fmt.Sprintf("hec:stats:%s", tokenID)
	pipe.HSet(ctx, statsKey, map[string]interface{}{
		"last_used_at": nowUnix,
		"last_used_ip": clientIP.String(),
	})
	pipe.HIncrBy(ctx, statsKey, "total_events", eventCount)

	// Hourly counter (48h expiry for rolling window calculations)
	hourlyKey := fmt.Sprintf("hec:hourly:%s:%s", tokenID, hourKey)
	pipe.IncrBy(ctx, hourlyKey, eventCount)
	pipe.Expire(ctx, hourlyKey, 48*time.Hour)

	// Daily counter (7d expiry)
	dailyKey := fmt.Sprintf("hec:daily:%s:%s", tokenID, dayKey)
	pipe.IncrBy(ctx, dailyKey, eventCount)
	pipe.Expire(ctx, dailyKey, 7*24*time.Hour)

	// Unique IPs per day
	ipsKey := fmt.Sprintf("hec:ips:%s:%s", tokenID, dayKey)
	pipe.SAdd(ctx, ipsKey, clientIP.String())
	pipe.Expire(ctx, ipsKey, 7*24*time.Hour)

	// Track which ingest instance is handling this token
	instancesKey := fmt.Sprintf("hec:instances:%s", tokenID)
	pipe.HSet(ctx, instancesKey, c.instanceID, nowUnix)
	pipe.Expire(ctx, instancesKey, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	return nil
}

// BatchUpdate holds accumulated stats for batch writing.
type BatchUpdate struct {
	TokenID    string
	EventCount int64
	ClientIPs  map[string]struct{} // unique IPs seen
	LastIP     net.IP
}

// NewBatchUpdate creates a new batch update accumulator for a token.
func NewBatchUpdate(tokenID string) *BatchUpdate {
	return &BatchUpdate{
		TokenID:   tokenID,
		ClientIPs: make(map[string]struct{}),
	}
}

// Add accumulates an event into the batch.
func (b *BatchUpdate) Add(eventCount int64, clientIP net.IP) {
	b.EventCount += eventCount
	if clientIP != nil {
		ipStr := clientIP.String()
		b.ClientIPs[ipStr] = struct{}{}
		b.LastIP = clientIP
	}
}

// FlushBatch writes accumulated batch stats to Redis.
// Call this periodically (e.g., every 30 seconds) instead of on every request.
func (c *Client) FlushBatch(ctx context.Context, batch *BatchUpdate) error {
	if batch.EventCount == 0 {
		return nil
	}

	now := time.Now()
	hourKey := now.Format("2006010215")
	dayKey := now.Format("20060102")
	nowUnix := strconv.FormatInt(now.Unix(), 10)

	pipe := c.redis.Pipeline()

	// Main stats hash
	statsKey := fmt.Sprintf("hec:stats:%s", batch.TokenID)
	pipe.HSet(ctx, statsKey, map[string]interface{}{
		"last_used_at": nowUnix,
		"last_used_ip": batch.LastIP.String(),
	})
	pipe.HIncrBy(ctx, statsKey, "total_events", batch.EventCount)

	// Hourly counter
	hourlyKey := fmt.Sprintf("hec:hourly:%s:%s", batch.TokenID, hourKey)
	pipe.IncrBy(ctx, hourlyKey, batch.EventCount)
	pipe.Expire(ctx, hourlyKey, 48*time.Hour)

	// Daily counter
	dailyKey := fmt.Sprintf("hec:daily:%s:%s", batch.TokenID, dayKey)
	pipe.IncrBy(ctx, dailyKey, batch.EventCount)
	pipe.Expire(ctx, dailyKey, 7*24*time.Hour)

	// Unique IPs - add all seen in this batch
	ipsKey := fmt.Sprintf("hec:ips:%s:%s", batch.TokenID, dayKey)
	if len(batch.ClientIPs) > 0 {
		ips := make([]interface{}, 0, len(batch.ClientIPs))
		for ip := range batch.ClientIPs {
			ips = append(ips, ip)
		}
		pipe.SAdd(ctx, ipsKey, ips...)
		pipe.Expire(ctx, ipsKey, 7*24*time.Hour)
	}

	// Track ingest instance
	instancesKey := fmt.Sprintf("hec:instances:%s", batch.TokenID)
	pipe.HSet(ctx, instancesKey, c.instanceID, nowUnix)
	pipe.Expire(ctx, instancesKey, 24*time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to flush batch: %w", err)
	}

	return nil
}

// GetStats retrieves current statistics for a HEC token.
func (c *Client) GetStats(ctx context.Context, tokenID string) (*Stats, error) {
	now := time.Now()
	hourKey := now.Format("2006010215")
	dayKey := now.Format("20060102")

	// Build list of last 24 hourly keys for rolling window
	hourlyKeys := make([]string, 24)
	for i := 0; i < 24; i++ {
		t := now.Add(-time.Duration(i) * time.Hour)
		hourlyKeys[i] = fmt.Sprintf("hec:hourly:%s:%s", tokenID, t.Format("2006010215"))
	}

	pipe := c.redis.Pipeline()

	// Main stats
	statsKey := fmt.Sprintf("hec:stats:%s", tokenID)
	statsCmd := pipe.HGetAll(ctx, statsKey)

	// Current hour
	currentHourCmd := pipe.Get(ctx, fmt.Sprintf("hec:hourly:%s:%s", tokenID, hourKey))

	// Last 24 hours (sum all hourly keys)
	hourlyCountCmds := make([]*redis.StringCmd, len(hourlyKeys))
	for i, key := range hourlyKeys {
		hourlyCountCmds[i] = pipe.Get(ctx, key)
	}

	// Unique IPs today
	ipsKey := fmt.Sprintf("hec:ips:%s:%s", tokenID, dayKey)
	uniqueIPsCmd := pipe.SCard(ctx, ipsKey)

	// Ingest instances
	instancesKey := fmt.Sprintf("hec:instances:%s", tokenID)
	instancesCmd := pipe.HGetAll(ctx, instancesKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := &Stats{
		TokenID:          tokenID,
		StatsRetrievedAt: now,
		IngestInstances:  make(map[string]string),
	}

	// Parse main stats
	if statsMap, err := statsCmd.Result(); err == nil {
		if lastUsedStr, ok := statsMap["last_used_at"]; ok {
			if unix, err := strconv.ParseInt(lastUsedStr, 10, 64); err == nil {
				t := time.Unix(unix, 0)
				stats.LastUsedAt = &t
			}
		}
		if ip, ok := statsMap["last_used_ip"]; ok {
			stats.LastUsedIP = ip
		}
		if totalStr, ok := statsMap["total_events"]; ok {
			stats.TotalEvents, _ = strconv.ParseInt(totalStr, 10, 64)
		}
	}

	// Current hour
	if val, err := currentHourCmd.Int64(); err == nil {
		stats.EventsLastHour = val
	}

	// Sum last 24 hours
	for _, cmd := range hourlyCountCmds {
		if val, err := cmd.Int64(); err == nil {
			stats.EventsLast24h += val
		}
	}

	// Unique IPs
	if val, err := uniqueIPsCmd.Result(); err == nil {
		stats.UniqueIPsToday = val
	}

	// Ingest instances
	if instances, err := instancesCmd.Result(); err == nil {
		for instance, lastSeen := range instances {
			if unix, err := strconv.ParseInt(lastSeen, 10, 64); err == nil {
				stats.IngestInstances[instance] = time.Unix(unix, 0).Format(time.RFC3339)
			}
		}
	}

	return stats, nil
}

// GetMultiStats retrieves statistics for multiple tokens at once.
func (c *Client) GetMultiStats(ctx context.Context, tokenIDs []string) (map[string]*Stats, error) {
	results := make(map[string]*Stats, len(tokenIDs))

	// TODO: Optimize with pipelining if needed
	for _, tokenID := range tokenIDs {
		stats, err := c.GetStats(ctx, tokenID)
		if err != nil {
			return nil, err
		}
		results[tokenID] = stats
	}

	return results, nil
}

// ListActiveTokens returns token IDs that have been used in the last duration.
func (c *Client) ListActiveTokens(ctx context.Context, since time.Duration) ([]string, error) {
	// Scan for all hec:stats:* keys
	var tokenIDs []string
	cutoff := time.Now().Add(-since).Unix()

	iter := c.redis.Scan(ctx, 0, "hec:stats:*", 1000).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// Extract token ID from key
		if len(key) > 10 { // "hec:stats:" = 10 chars
			tokenID := key[10:]

			// Check last_used_at
			lastUsed, err := c.redis.HGet(ctx, key, "last_used_at").Int64()
			if err == nil && lastUsed >= cutoff {
				tokenIDs = append(tokenIDs, tokenID)
			}
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan tokens: %w", err)
	}

	return tokenIDs, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.redis.Close()
}
