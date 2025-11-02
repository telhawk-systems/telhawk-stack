# Core Normalization Pipeline

The core service converts raw ingestion payloads into OCSF-compliant events before
shipping them downstream to storage and query services.

## Package Layout

- `core/pkg/ocsf` – canonical Go structs that mirror the OCSF schema for use across
  services.
  - `event.go` - Core OCSF Event structure with all required fields
  - `constants.go` - OCSF category, class, activity, severity, and status enumerations
  - `examples_test.go` - Usage examples for creating OCSF events
- `core/internal/normalizer` – plug-in style converters that transform protocol-
  specific payloads (e.g., Splunk HEC) into the OCSF event model.
- `core/internal/validator` – validation units that enforce structural and class-
  specific requirements on already-normalized events.
- `core/internal/pipeline` – orchestrates the normalization flow by selecting a
  normalizer, running validators, and returning JSON output to callers.
- `core/internal/service` – wraps the pipeline and tracks simple processing
  metrics for health endpoints.
- `core/internal/handlers` – HTTP boundary that accepts base64 encoded payloads
  and returns normalized events.

## Request Flow

1. Ingest forwards raw events to `/api/v1/normalize` with metadata describing the
   source, payload format, and reception time.
2. The router selects a normalizer using `normalizer.Registry.Find`. If none are
   registered for the given format/source type the request is rejected.
3. The resulting `ocsf.Event` instance is validated by a `validator.Chain`, which
   runs class-aware checks that can be extended per schema.
4. On success, the pipeline serializes the event as JSON and returns it to the
   caller. Ingest can then forward the validated payload to storage.

## Extending the Pipeline

- **New normalizers**: Implement `normalizer.Normalizer` and register it inside
  `core/cmd/core/main.go`. Normalizers can use shared lookup tables under
  `core/internal/normalizer/maps` (create as needed) to translate source-specific
  values. Use the constants from `ocsf.Constants` package for category, class,
  and activity IDs.
- **Additional validation**: Create new types that satisfy `validator.Validator`
  and append them to the chain. Validators can be scoped to specific OCSF
  classes by returning `false` from `Supports` for unrelated events.
- **Enrichment hooks**: Compose post-normalization processors by wrapping the
  pipeline with additional steps before returning the result to callers.

## OCSF Event Structure

All events must include these required OCSF fields:

```go
type Event struct {
    // Required OCSF base fields
    CategoryUID int       `json:"category_uid"`    // e.g., 3 for IAM
    ClassUID    int       `json:"class_uid"`       // e.g., 3002 for Authentication
    ActivityID  int       `json:"activity_id"`     // e.g., 1 for Logon
    TypeUID     int       `json:"type_uid"`        // Composite identifier
    Time        time.Time `json:"time"`            // When event occurred
    SeverityID  int       `json:"severity_id"`     // 0-6 (Unknown to Fatal)
    Metadata    Metadata  `json:"metadata"`        // Product info, version, profiles

    // Human-readable equivalents
    Category string `json:"category"`
    Class    string `json:"class"`
    Activity string `json:"activity"`
    Severity string `json:"severity"`

    // Status fields
    Status   string `json:"status,omitempty"`
    StatusID int    `json:"status_id,omitempty"`

    // Additional fields...
}
```

Use the helper function `ocsf.ComputeTypeUID(categoryUID, classUID, activityID)` to
calculate the proper `type_uid` value. See `core/pkg/ocsf/examples_test.go` for
complete examples.
