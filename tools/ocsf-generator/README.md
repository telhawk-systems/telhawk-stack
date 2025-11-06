# OCSF Code Generator

Generates Go structs for all OCSF event classes from the official OCSF schema.

## Features

- ✅ Generates type-safe Go structs for all 59+ OCSF event classes
- ✅ Creates activity enums for each class
- ✅ Includes constructor helpers
- ✅ Preserves OCSF documentation in Go comments
- ✅ Maps OCSF types to appropriate Go types
- ✅ Handles required vs optional fields

## Usage

### Prerequisites

You need the OCSF schema repository. If you don't have it:

```bash
cd /path/to/telhawk-stack
git clone https://github.com/ocsf/ocsf-schema.git
```

### Generate All Event Classes

```bash
cd tools/ocsf-generator

# Generate with default paths (assumes ocsf-schema is at ../../ocsf-schema)
go run main.go

# Or specify custom paths
go run main.go \
  -schema /path/to/ocsf-schema \
  -output ../../core/pkg/ocsf/classes \
  -v  # verbose output
```

### Output

Generated files will be created in `core/pkg/ocsf/classes/`:

```
core/pkg/ocsf/classes/
├── authentication.go
├── network_activity.go
├── process_activity.go
├── file_activity.go
├── detection_finding.go
├── ... (59 total classes)
```

## Generated Code Structure

Each generated file contains:

1. **Event Struct**: Embeds `ocsf.Event` and adds class-specific fields
2. **Activity Constants**: Enum values for activity_id
3. **Constructor**: Helper function to create events with proper defaults

### Example: Authentication

```go
// Authentication - Authentication events report authentication session activities...
// OCSF Class UID: 3002
type Authentication struct {
    // Embed base OCSF event
    ocsf.Event

    // Class-specific attributes
    AuthProtocol string `json:"auth_protocol,omitempty"`
    Certificate  *Certificate `json:"certificate,omitempty"`
    DstEndpoint  *Endpoint `json:"dst_endpoint,omitempty"`
    // ... more fields
}

// Activity IDs for Authentication
const (
    AuthenticationActivityLogon = 1 // A new logon session was requested.
    AuthenticationActivityLogoff = 2 // A logon session was terminated...
    // ... more activities
)

// NewAuthentication creates a new Authentication event
func NewAuthentication(activityID int) *Authentication {
    return &Authentication{
        Event: ocsf.Event{
            CategoryUID: 3,
            ClassUID:    3002,
            ActivityID:  activityID,
            // ... pre-filled defaults
        },
    }
}
```

## Usage Example

Once generated, use the classes like this:

```go
package main

import (
    "github.com/telhawk-systems/telhawk-stack/core/pkg/ocsf/classes"
)

func main() {
    // Create an authentication logon event
    event := classes.NewAuthentication(classes.AuthenticationActivityLogon)
    event.SeverityID = ocsf.SeverityInformational
    event.Severity = "Informational"
    event.StatusID = ocsf.StatusSuccess
    event.Status = "Success"
    
    // Add class-specific fields
    event.AuthProtocol = "LDAP"
    event.DstEndpoint = &ocsf.Endpoint{
        Hostname: "auth.example.com",
    }
    
    // Serialize to JSON
    data, _ := json.Marshal(event)
}
```

## Regenerating After Schema Updates

When the OCSF schema is updated:

```bash
cd /path/to/ocsf-schema
git pull

cd /path/to/telhawk-stack/tools/ocsf-generator
go run main.go -v
```

All classes will be regenerated with the latest schema definitions.

## Flags

- `-schema` - Path to OCSF schema directory (default: `../../ocsf-schema`)
- `-output` - Output directory for generated code (default: `../../core/pkg/ocsf/classes`)
- `-v` - Verbose output showing each generated class

## Architecture

The generator:

1. Reads `categories.json` for category metadata
2. Reads `dictionary.json` for attribute type information
3. Scans `events/*/` directories for event class definitions
4. For each event class:
   - Generates a Go struct with all class-specific fields
   - Creates activity constants from the `activity_id` enum
   - Generates a constructor with proper OCSF field defaults
5. Writes generated code to output directory

## Type Mapping

OCSF types are mapped to Go types:

| OCSF Type | Go Type |
|-----------|---------|
| `string_t` | `string` |
| `integer_t` | `int` |
| `long_t` | `int64` |
| `float_t` | `float64` |
| `boolean_t` | `bool` |
| `timestamp_t` | `time.Time` |
| `json_t` | `interface{}` |
| Object types | `*ObjectName` (pointer) |
| Arrays | `[]Type` |

## Limitations

- Object types (like `User`, `Endpoint`, etc.) need to be defined separately in `core/pkg/ocsf/`
- Complex nested structures may need manual adjustment
- Extensions and profiles are not yet fully supported
- Deprecated fields are included but marked

## Future Enhancements

- [ ] Generate object types from `objects/` directory
- [ ] Generate validators for each class
- [ ] Support OCSF extensions
- [ ] Generate OpenAPI/JSON Schema definitions
- [ ] Add profile support
- [ ] Generate test fixtures

## License

Same as TelHawk Stack (TSSAL v1.0 - TelHawk Systems Source Available License)
