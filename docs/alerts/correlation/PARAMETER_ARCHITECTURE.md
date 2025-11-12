## Parameter Architecture

### Versioning Strategy

**Structural Parameters** (require new version):
- `correlation_type` - changing type changes evaluation logic
- `queries` / `sequence` / `join_conditions` - changing detection logic
- `group_by` - changing aggregation keys affects results

**Tuning Parameters** (adjustable without versioning):
- `threshold`, `operator` - numeric comparisons
- `time_window`, `max_gap` - time boundaries
- `sensitivity`, `deviation_threshold` - detection sensitivity
- `suppression.window`, `suppression.max_alerts` - alert throttling

### Parameter Sets

Rules can define multiple named parameter configurations:

```json
{
  "model": {
    "correlation_type": "event_count",
    "parameters": {
      "time_window": "5m",
      "group_by": ["user.name"]
    },
    "parameter_sets": [
      {
        "name": "dev",
        "description": "Relaxed thresholds for development",
        "threshold": 5,
        "time_window": "10m"
      },
      {
        "name": "prod",
        "description": "Production thresholds",
        "threshold": 10,
        "time_window": "5m"
      },
      {
        "name": "strict",
        "description": "High-security environment",
        "threshold": 3,
        "time_window": "3m"
      }
    ],
    "active_parameter_set": "prod"
  }
}
```

**Selection mechanism**:
1. Rule specifies `active_parameter_set` (default: none)
2. API allows setting active set: `PUT /api/v1/schemas/{id}/parameters`
3. Parameters from active set override base parameters
4. Base parameters used if no set active

### Parameter Validation

Each correlation type defines parameter schema:

```go
type EventCountParams struct {
    TimeWindow    string   `json:"time_window" validate:"required,duration"`
    Threshold     int      `json:"threshold" validate:"required,gt=0"`
    Operator      string   `json:"operator" validate:"required,oneof=gt gte lt lte eq ne"`
    GroupBy       []string `json:"group_by" validate:"omitempty,dive,required"`
}

type JoinParams struct {
    TimeWindow     string        `json:"time_window" validate:"required,duration"`
    LeftQuery      QueryConfig   `json:"left_query" validate:"required"`
    RightQuery     QueryConfig   `json:"right_query" validate:"required"`
    JoinConditions []JoinCondition `json:"join_conditions" validate:"required,min=1,dive"`
    JoinType       string        `json:"join_type" validate:"oneof=inner left any"`
}
```

Validation occurs:
1. **At rule creation** - fail fast if parameters invalid
2. **At parameter set activation** - prevent broken configs
3. **At evaluation time** - graceful fallback if Redis unavailable

---

