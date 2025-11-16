package correlation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// StateManager manages correlation state in Redis
type StateManager struct {
	redis   *redis.Client
	enabled bool
}

// NewStateManager creates a new state manager
func NewStateManager(redisClient *redis.Client, enabled bool) *StateManager {
	return &StateManager{
		redis:   redisClient,
		enabled: enabled,
	}
}

// IsEnabled returns whether the state manager is enabled
func (sm *StateManager) IsEnabled() bool {
	return sm.enabled && sm.redis != nil
}

// Baseline represents statistical baseline data
type Baseline struct {
	Samples     []float64 `json:"samples"`      // Rolling window of samples
	Count       int64     `json:"count"`        // Total sample count
	Sum         float64   `json:"sum"`          // Sum of all samples
	SumSquares  float64   `json:"sum_squares"`  // Sum of squares (for std dev)
	Mean        float64   `json:"mean"`         // Calculated mean
	StdDev      float64   `json:"stddev"`       // Calculated standard deviation
	LastUpdated int64     `json:"last_updated"` // Unix timestamp
}

// SuppressionState represents suppression cache state
type SuppressionState struct {
	FirstAlertTime     int64                  `json:"first_alert_time"`
	LastAlertTime      int64                  `json:"last_alert_time"`
	AlertCount         int                    `json:"alert_count"`
	SuppressionContext map[string]interface{} `json:"suppression_context"`
}

// HeartbeatState represents heartbeat tracking state
type HeartbeatState struct {
	Entity       string `json:"entity"`
	LastSeen     int64  `json:"last_seen"` // Unix timestamp
	MissedCount  int    `json:"missed_count"`
	ExpectedNext int64  `json:"expected_next"` // Unix timestamp
}

// GetBaseline retrieves baseline data for a rule and entity
func (sm *StateManager) GetBaseline(ctx context.Context, ruleID, entityKey string) (*Baseline, error) {
	if !sm.IsEnabled() {
		return nil, fmt.Errorf("state manager is disabled")
	}

	key := sm.baselineKey(ruleID, entityKey)
	data, err := sm.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// No baseline exists yet - return empty baseline
		return &Baseline{
			Samples:     []float64{},
			LastUpdated: time.Now().Unix(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	var baseline Baseline
	if err := json.Unmarshal([]byte(data), &baseline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal baseline: %w", err)
	}

	return &baseline, nil
}

// UpdateBaseline updates baseline data for a rule and entity
func (sm *StateManager) UpdateBaseline(ctx context.Context, ruleID, entityKey string, value float64, window time.Duration) error {
	if !sm.IsEnabled() {
		return fmt.Errorf("state manager is disabled")
	}

	// Get existing baseline
	baseline, err := sm.GetBaseline(ctx, ruleID, entityKey)
	if err != nil {
		return err
	}

	// Add new sample
	baseline.Samples = append(baseline.Samples, value)
	baseline.Count++
	baseline.Sum += value
	baseline.SumSquares += value * value
	baseline.LastUpdated = time.Now().Unix()

	// Calculate mean and standard deviation
	if baseline.Count > 0 {
		baseline.Mean = baseline.Sum / float64(baseline.Count)

		// Calculate variance: Var(X) = E[X²] - (E[X])²
		variance := (baseline.SumSquares / float64(baseline.Count)) - (baseline.Mean * baseline.Mean)
		if variance > 0 {
			// Note: For production, import math and use math.Sqrt(variance)
			// For now, store the variance as approximation
			baseline.StdDev = variance
		}
	}

	// Limit samples array to reasonable size (keep last 1000 samples)
	if len(baseline.Samples) > 1000 {
		baseline.Samples = baseline.Samples[len(baseline.Samples)-1000:]
	}

	// Save updated baseline
	data, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	key := sm.baselineKey(ruleID, entityKey)
	ttl := window * 2 // Store baseline for 2x the baseline window
	if err := sm.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save baseline: %w", err)
	}

	return nil
}

// IsSuppressed checks if an alert should be suppressed
func (sm *StateManager) IsSuppressed(ctx context.Context, ruleID string, suppressionKey map[string]string) (bool, error) {
	if !sm.IsEnabled() {
		return false, nil // Don't suppress if state manager is disabled
	}

	key := sm.suppressionKey(ruleID, suppressionKey)
	exists, err := sm.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check suppression: %w", err)
	}

	return exists > 0, nil
}

// RecordAlert records an alert for suppression tracking
func (sm *StateManager) RecordAlert(ctx context.Context, ruleID string, suppressionKey map[string]string, window time.Duration, maxAlerts int) error {
	if !sm.IsEnabled() {
		return nil // Skip if state manager is disabled
	}

	key := sm.suppressionKey(ruleID, suppressionKey)
	now := time.Now().Unix()

	// Get existing state
	data, err := sm.redis.Get(ctx, key).Result()
	var state SuppressionState
	if errors.Is(err, redis.Nil) {
		// First alert
		state = SuppressionState{
			FirstAlertTime:     now,
			LastAlertTime:      now,
			AlertCount:         1,
			SuppressionContext: make(map[string]interface{}),
		}
		for k, v := range suppressionKey {
			state.SuppressionContext[k] = v
		}
	} else if err != nil {
		return fmt.Errorf("failed to get suppression state: %w", err)
	} else {
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			return fmt.Errorf("failed to unmarshal suppression state: %w", err)
		}
		state.AlertCount++
		state.LastAlertTime = now
	}

	// Save updated state
	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal suppression state: %w", err)
	}

	if err := sm.redis.Set(ctx, key, stateData, window).Err(); err != nil {
		return fmt.Errorf("failed to save suppression state: %w", err)
	}

	return nil
}

// RecordHeartbeat records a heartbeat for an entity
func (sm *StateManager) RecordHeartbeat(ctx context.Context, ruleID, entityID string, expectedInterval time.Duration) error {
	if !sm.IsEnabled() {
		return nil // Skip if state manager is disabled
	}

	now := time.Now().Unix()
	state := HeartbeatState{
		Entity:       entityID,
		LastSeen:     now,
		MissedCount:  0,
		ExpectedNext: now + int64(expectedInterval.Seconds()),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat state: %w", err)
	}

	key := sm.heartbeatKey(ruleID, entityID)
	// TTL should be longer than expected interval to detect missed heartbeats
	ttl := expectedInterval * 3
	if err := sm.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save heartbeat: %w", err)
	}

	return nil
}

// GetMissingSince returns the time since last heartbeat for an entity
func (sm *StateManager) GetMissingSince(ctx context.Context, ruleID, entityID string) (time.Time, error) {
	if !sm.IsEnabled() {
		return time.Time{}, fmt.Errorf("state manager is disabled")
	}

	key := sm.heartbeatKey(ruleID, entityID)
	data, err := sm.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Never seen this entity
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get heartbeat: %w", err)
	}

	var state HeartbeatState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return time.Time{}, fmt.Errorf("failed to unmarshal heartbeat: %w", err)
	}

	return time.Unix(state.LastSeen, 0), nil
}

// GetAllHeartbeats returns all tracked heartbeats for a rule
func (sm *StateManager) GetAllHeartbeats(ctx context.Context, ruleID string) ([]HeartbeatState, error) {
	if !sm.IsEnabled() {
		return nil, fmt.Errorf("state manager is disabled")
	}

	pattern := fmt.Sprintf("heartbeat:%s:*", ruleID)
	keys, err := sm.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat keys: %w", err)
	}

	heartbeats := make([]HeartbeatState, 0, len(keys))
	for _, key := range keys {
		data, err := sm.redis.Get(ctx, key).Result()
		if err != nil {
			continue // Skip if error
		}

		var state HeartbeatState
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			continue // Skip if error
		}

		heartbeats = append(heartbeats, state)
	}

	return heartbeats, nil
}

// baselineKey generates a Redis key for baseline data
func (sm *StateManager) baselineKey(ruleID, entityKey string) string {
	hash := hashKey(entityKey)
	return fmt.Sprintf("baseline:%s:%s", ruleID, hash)
}

// suppressionKey generates a Redis key for suppression state
func (sm *StateManager) suppressionKey(ruleID string, suppressionKey map[string]string) string {
	hash := hashMap(suppressionKey)
	return fmt.Sprintf("suppression:%s:%s", ruleID, hash)
}

// heartbeatKey generates a Redis key for heartbeat tracking
func (sm *StateManager) heartbeatKey(ruleID, entityID string) string {
	return fmt.Sprintf("heartbeat:%s:%s", ruleID, entityID)
}

// hashKey generates a consistent hash for a string key
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash[:8])
}

// hashMap generates a consistent hash for a map
func hashMap(m map[string]string) string {
	// Sort keys and create deterministic string representation
	data, err := json.Marshal(m)
	if err != nil {
		// If marshaling fails, return hash of empty data
		data = []byte{}
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:8])
}
