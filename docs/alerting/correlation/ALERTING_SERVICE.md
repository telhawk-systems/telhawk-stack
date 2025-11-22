## Alerting Service Architecture

### Current State

The alerting service evaluates events individually:

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│  Rules Svc  │─────→│  Evaluator   │─────→│  Storage    │
│ (schemas)   │      │ (stateless)  │      │ (alerts)    │
└─────────────┘      └──────────────┘      └─────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  OpenSearch  │
                     │   (events)   │
                     └──────────────┘
```

### Correlation Architecture

```
┌─────────────┐      ┌──────────────────────────────┐      ┌─────────────┐
│  Rules Svc  │─────→│  Correlation Engine          │─────→│  Storage    │
│ (schemas)   │      │                              │      │ (alerts)    │
└─────────────┘      │  ┌─────────────────────┐    │      └─────────────┘
                     │  │ Evaluator           │    │
                     │  │ - Simple rules      │    │
                     │  │ - Correlation rules │    │
                     │  └─────────────────────┘    │
                     │           │                  │
                     │           ▼                  │
                     │  ┌─────────────────────┐    │
                     │  │ State Manager       │◄───┼───► Redis
                     │  │ - Baselines         │    │    (state)
                     │  │ - Suppression cache │    │
                     │  │ - Rate tracking     │    │
                     │  └─────────────────────┘    │
                     │           │                  │
                     │           ▼                  │
                     │  ┌─────────────────────┐    │
                     │  │ Query Orchestrator  │    │
                     │  │ - Single query      │    │
                     │  │ - Multi-query (join)│    │
                     │  │ - Aggregation       │    │
                     │  └─────────────────────┘    │
                     └──────────────┬───────────────┘
                                    │
                                    ▼
                             ┌──────────────┐
                             │  OpenSearch  │
                             │   (events)   │
                             └──────────────┘
```

### Component Design

#### 1. State Manager (Redis)

Manages stateful correlation data:

```go
type StateManager struct {
    redis *redis.Client
}

// Baseline management
func (sm *StateManager) GetBaseline(ruleID, entityKey string) (*Baseline, error)
func (sm *StateManager) UpdateBaseline(ruleID, entityKey string, value float64) error

// Suppression management
func (sm *StateManager) IsSupressed(ruleID string, suppressionKey map[string]string) (bool, error)
func (sm *StateManager) RecordAlert(ruleID string, suppressionKey map[string]string, window time.Duration) error

// Heartbeat tracking
func (sm *StateManager) RecordHeartbeat(ruleID, entityID string) error
func (sm *StateManager) GetMissingSince(ruleID, entityID string, expected time.Duration) (time.Time, error)
```

**Redis Key Patterns**:
```
# Baselines
baseline:<rule_id>:<entity_hash> → JSON{samples, mean, stddev, ...}

# Suppression
suppression:<rule_id>:<key_hash> → JSON{first_alert, count, context}

# Heartbeats
heartbeat:<rule_id>:<entity_id> → timestamp

# Rate tracking
rate:<rule_id>:<entity_hash>:<minute> → count
```

#### 2. Query Orchestrator

Executes correlation queries:

```go
type QueryOrchestrator struct {
    storageClient StorageClient
}

// Single query execution (event_count, value_count, baseline_deviation)
func (qo *QueryOrchestrator) ExecuteQuery(ctx context.Context, query string, window time.Duration) ([]*Event, error)

// Multi-query execution (temporal, temporal_ordered)
func (qo *QueryOrchestrator) ExecuteMultiQuery(ctx context.Context, queries []Query, window time.Duration) (map[string][]*Event, error)

// Join execution
func (qo *QueryOrchestrator) ExecuteJoin(ctx context.Context, left, right Query, conditions []JoinCondition, window time.Duration) ([]*JoinedEvent, error)
```

#### 3. Correlation Evaluator

Main evaluation logic with correlation support:

```go
type CorrelationEvaluator struct {
    rulesClient   RulesClient
    orchestrator  *QueryOrchestrator
    stateManager  *StateManager
    evaluators    map[string]CorrelationTypeEvaluator
}

// Main evaluation loop
func (ce *CorrelationEvaluator) Run(ctx context.Context, interval time.Duration) {
    // Fetch schemas
    schemas := ce.rulesClient.ListSchemas(ctx)

    for _, schema := range schemas {
        correlationType := schema.Model["correlation_type"]

        // Route to appropriate evaluator
        evaluator := ce.evaluators[correlationType]
        alerts := evaluator.Evaluate(ctx, schema, ce.orchestrator, ce.stateManager)

        // Check suppression
        for _, alert := range alerts {
            if !ce.isSupressed(schema, alert) {
                ce.storeAlert(ctx, alert)
            }
        }
    }
}

// Type-specific evaluators
type EventCountEvaluator struct{}
func (e *EventCountEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema, orch *QueryOrchestrator, state *StateManager) []*Alert

type JoinEvaluator struct{}
func (e *JoinEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema, orch *QueryOrchestrator, state *StateManager) []*Alert

type BaselineDeviationEvaluator struct{}
func (e *BaselineDeviationEvaluator) Evaluate(ctx context.Context, schema *DetectionSchema, orch *QueryOrchestrator, state *StateManager) []*Alert
```

### Evaluation Flow

```
1. Load correlation rules from Rules service
   ↓
2. Group by correlation_type
   ↓
3. For each rule:
   ├─→ Load active parameter set (if any)
   ├─→ Route to type-specific evaluator
   │   ├─→ EventCountEvaluator
   │   ├─→ JoinEvaluator
   │   ├─→ BaselineDeviationEvaluator
   │   └─→ etc.
   ├─→ Execute correlation query via Orchestrator
   ├─→ Check/update state via StateManager
   ├─→ Compare against threshold
   └─→ Generate alert(s)
   ↓
4. For each alert:
   ├─→ Check suppression in StateManager
   ├─→ If not suppressed:
   │   ├─→ Store alert in OpenSearch
   │   └─→ Record in suppression cache
   └─→ If suppressed: increment suppression counter
```

### Performance Considerations

**Query Optimization**:
- Use OpenSearch aggregations for counting
- Batch multi-query execution when possible
- Limit results with `max_events_per_window` parameter

**State Management**:
- Use Redis pipelining for batch baseline updates
- Set appropriate TTLs to prevent memory growth
- Use Redis Cluster for horizontal scaling

**Concurrency**:
- Evaluate rules concurrently (Go goroutines)
- Rate limit per-rule evaluation (prevent thundering herd)
- Timeout protection on slow queries

---

