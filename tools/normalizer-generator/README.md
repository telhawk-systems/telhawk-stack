# OCSF Normalizer Generator

Automatically generates Go normalizers for OCSF event classes from configuration.

## Overview

This tool generates type-safe, OCSF-compliant event normalizers that convert raw log data into standardized OCSF events. It follows the same code generation pattern as `tools/ocsf-generator`.

## Features

✅ Generates one normalizer per OCSF event class  
✅ Intelligent source type pattern matching  
✅ Common field name variant handling  
✅ Activity ID determination from keywords  
✅ Shared helper functions for consistency  
✅ Type-safe code with zero manual errors  

## Usage

### Generate All Normalizers

```bash
cd tools/normalizer-generator
go run main.go -v
```

### Generated Output

Files are created in `core/internal/normalizer/generated/`:

```
core/internal/normalizer/generated/
├── helpers.go                      # Shared extraction functions
├── authentication_normalizer.go    # IAM authentication events
├── network_activity_normalizer.go  # Network connection events
├── process_activity_normalizer.go  # Process lifecycle events
├── file_activity_normalizer.go     # File operations
├── dns_activity_normalizer.go      # DNS queries
├── http_activity_normalizer.go     # HTTP requests
└── detection_finding_normalizer.go # Security detections
```

## Configuration

### Field Mappings (`field_mappings.json`)

Maps common field name variants to OCSF fields:

```json
{
  "common_fields": {
    "user": {
      "ocsf_field": "User.Name",
      "variants": ["user", "username", "user_name", "account"],
      "type": "string"
    },
    "timestamp": {
      "ocsf_field": "Time",
      "variants": ["timestamp", "time", "@timestamp"],
      "type": "timestamp",
      "parser": "parseTimestamp"
    }
  }
}
```

### Source Type Patterns (`sourcetype_patterns.json`)

Defines how to classify events by source type:

```json
{
  "patterns": {
    "authentication": {
      "class_uid": 3002,
      "category": "iam",
      "sourcetype_patterns": ["^auth", "^login", "^sso"],
      "content_patterns": ["login", "authentication"],
      "priority": 100,
      "activity_keywords": {
        "logon": ["login", "logon"],
        "logoff": ["logout", "logoff"]
      }
    }
  }
}
```

## Generated Code Structure

Each normalizer includes:

### 1. Type Definition
```go
type AuthenticationNormalizer struct{}
```

### 2. Constructor
```go
func NewAuthenticationNormalizer() *AuthenticationNormalizer {
    return &AuthenticationNormalizer{}
}
```

### 3. Supports Method
Pattern matching for source type classification:
```go
func (n *AuthenticationNormalizer) Supports(format, sourceType string) bool {
    st := strings.ToLower(sourceType)
    return strings.Contains(st, "auth") || 
           strings.Contains(st, "login") || ...
}
```

### 4. Normalize Method
Extracts fields and creates OCSF event:
```go
func (n *AuthenticationNormalizer) Normalize(ctx context.Context, envelope *RawEventEnvelope) (*ocsf.Event, error) {
    var payload map[string]interface{}
    json.Unmarshal(envelope.Payload, &payload)
    
    activityID := n.determineActivityID(payload)
    event := iam.NewAuthentication(activityID)
    
    event.Time = ExtractTimestamp(payload, envelope.ReceivedAt)
    event.User = ExtractUser(payload)
    // ... more field extraction
    
    return &event.Event, nil
}
```

### 5. Activity Determination
Maps actions to OCSF activity IDs:
```go
func (n *AuthenticationNormalizer) determineActivityID(payload map[string]interface{}) int {
    action := strings.ToLower(ExtractString(payload, "action", "event_type"))
    if strings.Contains(action, "logout") {
        return 2 // Logoff
    }
    return 1 // Logon
}
```

## Shared Helpers (`helpers.go`)

All normalizers share common extraction functions:

- `ExtractString(payload, keys...)` - Try multiple field names
- `ExtractUser(payload)` - Extract user information
- `ExtractTimestamp(payload, fallback)` - Parse various time formats
- `ExtractStatus(payload)` - Map status strings to OCSF codes
- `ExtractSeverity(payload)` - Map severity strings to OCSF codes

## Usage in Code

```go
package main

import (
    "github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
    "github.com/telhawk-systems/telhawk-stack/core/internal/normalizer/generated"
)

func main() {
    // Create generated normalizers
    authNorm := generated.NewAuthenticationNormalizer()
    netNorm := generated.NewNetworkActivityNormalizer()
    procNorm := generated.NewProcessActivityNormalizer()
    // ...
    
    // Register with normalizer registry
    registry := normalizer.NewRegistry(
        authNorm,
        netNorm,
        procNorm,
        // Add remaining normalizers
    )
    
    // Use in pipeline
    pipeline := pipeline.New(registry, validators)
}
```

## Adding New Event Classes

To add a new OCSF event class:

1. Add pattern to `sourcetype_patterns.json`:
```json
{
  "patterns": {
    "your_event_class": {
      "class_uid": 1234,
      "category": "category_name",
      "sourcetype_patterns": ["^pattern"],
      "priority": 90
    }
  }
}
```

2. Regenerate:
```bash
go run main.go
```

3. New normalizer is automatically created!

## Flags

- `-schema` - Path to OCSF schema directory (not used yet)
- `-output` - Output directory (default: `../../core/internal/normalizer/generated`)
- `-v` - Verbose output

## Development

### Modifying Field Extraction

Update helper functions in `helpers.go` generation or add new extractors:

```go
func generateHelpersFile() error {
    // Add your new helper here
    buf.WriteString(`
// ExtractEndpoint extracts network endpoint information
func ExtractEndpoint(payload map[string]interface{}) *objects.NetworkEndpoint {
    // ... implementation
}
`)
}
```

### Adding Custom Parsers

Define in `field_mappings.json`:
```json
{
  "special_field": {
    "ocsf_field": "CustomField",
    "parser": "parseSpecialField"
  }
}
```

Then implement in helpers.

## Benefits

1. **Consistency** - All normalizers follow identical patterns
2. **Maintainability** - Update config → regenerate everything
3. **Completeness** - Easy to add all 59+ OCSF classes
4. **Type Safety** - Generated code is type-checked by Go compiler
5. **DRY Principle** - Don't write what can be generated
6. **Zero Errors** - No manual coding mistakes

## Future Enhancements

- [ ] Read OCSF schema directly (like ocsf-generator)
- [ ] Generate class-specific field extractors
- [ ] Add validator generation
- [ ] Support OCSF extensions and profiles
- [ ] Generate test cases
- [ ] Add metrics/observability hooks

## Related

- `tools/ocsf-generator/` - Generates OCSF event class structs
- `docs/NORMALIZER_GENERATION.md` - Overall strategy documentation
- [OCSF Schema](https://schema.ocsf.io/)
