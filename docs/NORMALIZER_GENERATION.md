# OCSF Normalizer Generation Strategy

## Overview

TelHawk Stack uses **code generation** to create OCSF-compliant event normalizers. This follows the same pattern as the existing `tools/ocsf-generator` which generates OCSF event classes.

## Why Code Generation?

### Current Approach (Manual) ❌
- Write ~300 lines per normalizer by hand
- Inconsistent field mapping across classes
- Prone to human error
- Difficult to maintain when OCSF schema updates
- Only covers a few event classes

### Generated Approach (Automated) ✅
- One generator creates normalizers for all 59+ OCSF classes
- Consistent field mapping rules across all classes
- Zero manual coding errors
- Easy to update: regenerate when schema changes
- Complete OCSF coverage automatically

## Architecture

```
tools/
├── ocsf-generator/           # Existing: generates event classes
│   └── main.go               # Reads OCSF schema → generates Go structs
│
└── normalizer-generator/     # NEW: generates normalizers
    ├── main.go               # Reads OCSF schema → generates normalizers
    ├── field_mappings.json   # Common field name variants
    └── sourcetype_patterns.json  # Source type classification rules
```

## What Gets Generated

### Input
From OCSF schema repository:
- `events/{category}/{class}.json` - Event class definitions
- `objects/{object}.json` - Object type definitions  
- `dictionary.json` - Field type information
- `categories.json` - Category metadata

### Output
Generated files in `core/internal/normalizer/generated/`:

```go
// authentication_normalizer.go
type AuthenticationNormalizer struct {
    registry *SourceTypeRegistry
}

func (n *AuthenticationNormalizer) Supports(format, sourceType string) bool {
    // Generated pattern matching
    return strings.Contains(sourceType, "auth") || 
           strings.Contains(sourceType, "login")
}

func (n *AuthenticationNormalizer) Normalize(ctx context.Context, envelope *RawEventEnvelope) (*ocsf.Event, error) {
    // Generated field extraction and mapping
    var payload map[string]interface{}
    json.Unmarshal(envelope.Payload, &payload)
    
    event := iam.NewAuthentication(determineActivityID(payload))
    event.User = extractUser(payload) // Generated helper
    event.Time = extractTimestamp(payload)
    // ... more generated field mappings
    return &event.Event, nil
}

// Generated helper methods
func extractUser(payload map[string]interface{}) *objects.User { /* ... */ }
func extractTimestamp(payload map[string]interface{}) time.Time { /* ... */ }
```

## Field Mapping Strategy

The generator uses a mapping table for common field name variants:

```json
{
  "user": {
    "ocsf_field": "User.Name",
    "variants": ["user", "username", "user_name", "account", "principal", "identity"]
  },
  "user_id": {
    "ocsf_field": "User.Uid",
    "variants": ["user_id", "uid", "user_uid", "account_id"]
  },
  "timestamp": {
    "ocsf_field": "Time",
    "variants": ["timestamp", "time", "@timestamp", "event_time", "datetime"],
    "parser": "parseTime"
  },
  "src_ip": {
    "ocsf_field": "SrcEndpoint.Uid",
    "variants": ["src_ip", "source_ip", "src_addr", "source_address"]
  }
}
```

## Source Type Classification

Patterns for determining which normalizer to use:

```json
{
  "authentication": {
    "sourcetype_patterns": ["^auth", "^login", "^session", "^sso", "^oauth", "^ldap"],
    "content_patterns": ["login", "logout", "authentication", "credential"],
    "priority": 100
  },
  "network_activity": {
    "sourcetype_patterns": ["^network", "^firewall", "^proxy", "^connection"],
    "content_patterns": ["src_ip", "dst_ip", "src_port", "dst_port"],
    "priority": 90
  }
}
```

## Generated Code Structure

Each generated normalizer includes:

1. **Type Definition**
   ```go
   type {Class}Normalizer struct {
       registry *SourceTypeRegistry
   }
   ```

2. **Constructor**
   ```go
   func New{Class}Normalizer(registry *SourceTypeRegistry) *{Class}Normalizer
   ```

3. **Supports Method** - Pattern matching
   ```go
   func (n *{Class}Normalizer) Supports(format, sourceType string) bool
   ```

4. **Normalize Method** - Field extraction and mapping
   ```go
   func (n *{Class}Normalizer) Normalize(ctx, envelope) (*ocsf.Event, error)
   ```

5. **Helper Methods** - Object extractors (User, Endpoint, Process, etc.)
   ```go
   func extractUser(payload) *objects.User
   func extractEndpoint(payload, prefix) *objects.NetworkEndpoint
   func extractProcess(payload) *objects.Process
   ```

## Usage After Generation

```go
// Initialize all generated normalizers
registry := normalizer.NewSourceTypeRegistry()
authNorm := generated.NewAuthenticationNormalizer(registry)
netNorm := generated.NewNetworkActivityNormalizer(registry)
procNorm := generated.NewProcessActivityNormalizer(registry)
// ... all 59+ normalizers

// Register them
normalizerRegistry := normalizer.NewRegistry(
    authNorm,
    netNorm,
    procNorm,
    // ... rest
)

// Use in pipeline
pipeline := pipeline.New(normalizerRegistry, validators)
```

## Regeneration Workflow

When OCSF schema updates:

```bash
# 1. Update OCSF schema
cd /path/to/ocsf-schema
git pull

# 2. Regenerate event classes
cd tools/ocsf-generator
go run main.go

# 3. Regenerate normalizers
cd tools/normalizer-generator
go run main.go

# 4. Rebuild and test
cd ../..
go test ./core/...
```

## Implementation Steps

### Phase 1: Create Generator Tool
- [ ] Create `tools/normalizer-generator/main.go`
- [ ] Define field mapping rules in JSON
- [ ] Define source type classification patterns
- [ ] Generate one normalizer as proof-of-concept

### Phase 2: Generate Core Normalizers
- [ ] Authentication (IAM category)
- [ ] Network Activity
- [ ] Process Activity
- [ ] File Activity
- [ ] Detection Finding

### Phase 3: Complete Coverage
- [ ] Generate all 59+ OCSF class normalizers
- [ ] Test with real-world log samples
- [ ] Tune field mapping rules

### Phase 4: Integration
- [ ] Update pipeline to use generated normalizers
- [ ] Update tests
- [ ] Document in main README

## Benefits

1. **Consistency** - All normalizers follow identical patterns
2. **Completeness** - All OCSF classes covered, not just a few
3. **Maintainability** - Update mappings → regenerate everything
4. **Quality** - No manual coding errors
5. **Speed** - Generator creates in seconds what takes hours manually
6. **Schema Compliance** - Always matches latest OCSF version

## Related Documentation

- [OCSF Schema](https://schema.ocsf.io/)
- [OCSF GitHub](https://github.com/ocsf/ocsf-schema)
- `tools/ocsf-generator/README.md` - Event class generator
- `docs/core-pipeline.md` - Normalization pipeline overview

## Next Steps

1. Create `tools/normalizer-generator/` directory
2. Port generator patterns from `tools/ocsf-generator/`
3. Add field mapping configuration
4. Generate first normalizer (Authentication)
5. Iterate and expand coverage
