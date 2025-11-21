# OCSF Event Coverage Analysis

**Date:** 2025-11-02  
**OCSF Version:** 1.1.0 (as referenced in code)  
**Schema Version Checked:** Latest from schema.ocsf.io

## Executive Summary

TelHawk Stack currently has **minimal OCSF implementation** with only a generic event mapping in place. The codebase supports 0 out of 59 active OCSF event classes.

### Current Status
- ✅ **Infrastructure Ready**: Core normalization pipeline architecture is in place
- ✅ **Base Event Model**: Generic OCSF event structure defined in `common/ocsf/ocsf/event.go`
- ⚠️ **Event Mapping**: Only HEC normalizer exists, creates generic placeholder events
- ❌ **Class-Specific Implementation**: No specific OCSF class implementations
- ❌ **Validators**: No class-specific validators implemented

### Priority for Implementation

Based on the README.md stated support, TelHawk Stack claims to support:
1. ❌ Network Activity (0/11 implemented)
2. ❌ Authentication (0/1 implemented)
3. ❌ System Activity (0/13 implemented)
4. ❌ Application Activity (0/7 implemented)
5. ❌ Detection Finding (0/1 implemented)
6. ❌ Security Finding (deprecated - should use Detection Finding)

## OCSF Schema Overview

The OCSF schema defines **82 total event classes** across 8 categories:

| Category | Active Classes | Deprecated | Status in TelHawk |
|----------|---------------|------------|-------------------|
| **Identity & Access Management** | 6 | 0 | ❌ Not Implemented |
| **System Activity** | 13 | 0 | ❌ Not Implemented |
| **Network Activity** | 11 | 3 | ❌ Not Implemented |
| **Application Activity** | 7 | 1 | ❌ Not Implemented |
| **Discovery** | 8 | 18 | ❌ Not Implemented |
| **Findings** | 7 | 1 | ❌ Not Implemented |
| **Remediation** | 4 | 0 | ❌ Not Implemented |
| **Unmanned Systems** | 2 | 0 | ❌ Not Implemented |
| **Other** | 1 | 0 | ⚠️ Generic Only |
| **Total** | **59** | **23** | **0/59 (0%)** |

## Current Implementation

### Implemented Components

#### 1. Base Event Structure (`common/ocsf/ocsf/event.go`)
```go
type Event struct {
    Class        string                 // Currently hardcoded to "generic_event"
    Category     string                 // Currently hardcoded to "activity"
    Activity     string                 // Format: "ingest:{sourceType}"
    Schema       SchemaMetadata         // Namespace: "ocsf", Version: "1.1.0"
    Severity     string                 // Currently hardcoded to "unknown"
    ObservedTime time.Time
    Actor        Actor
    Target       Target
    Enrichments  map[string]interface{}
    Properties   map[string]interface{}
    Raw          RawDescriptor
}
```

**Issues:**
- Missing `class_uid` (required OCSF field for event class ID)
- Missing `category_uid` (required OCSF field for category ID)
- Missing `activity_id` (required OCSF field for activity type)
- Missing `type_uid` (required OCSF composite identifier)
- Missing `severity_id` (should use numeric values 0-6)
- Missing `status` and `status_id` fields
- Missing `time` (event occurrence time) vs `observed_time`
- Actor/Target structures are too generic for OCSF compliance

#### 2. HEC Normalizer
- Only normalizer implemented
- Creates generic placeholder events
- Does not map to specific OCSF classes
- Minimal field mapping from HEC payload

### Missing Critical OCSF Fields

The current Event struct is missing these **required** OCSF base fields:

```go
// Should be added to Event struct:
ClassUID     int                    `json:"class_uid"`      // e.g., 3002 for Authentication
CategoryUID  int                    `json:"category_uid"`   // e.g., 3 for IAM
ActivityID   int                    `json:"activity_id"`    // e.g., 1 for Logon
TypeUID      int                    `json:"type_uid"`       // Composite: (category_uid * 100) + class_uid
SeverityID   int                    `json:"severity_id"`    // 0=Unknown, 1=Info, 2=Low...6=Critical
Status       string                 `json:"status,omitempty"`
StatusID     int                    `json:"status_id,omitempty"`
Time         time.Time              `json:"time"`           // When event occurred
Metadata     Metadata               `json:"metadata"`       // Product, version, profiles, etc.
```

## Detailed Event Class Breakdown

### 1. Identity & Access Management (Priority: HIGH)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 3001 | Account Change | ❌ Not Implemented | User account lifecycle |
| 3002 | **Authentication** | ❌ Not Implemented | **CLAIMED IN README** - Critical for security |
| 3003 | Authorize Session | ❌ Not Implemented | Session privilege assignment |
| 3004 | Entity Management | ❌ Not Implemented | Cloud resource management |
| 3005 | User Access Management | ❌ Not Implemented | Permission changes |
| 3006 | Group Management | ❌ Not Implemented | Group membership |

**Priority**: HIGH - Authentication (3002) is explicitly mentioned in README as supported but not implemented.

### 2. System Activity (Priority: HIGH)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 1001 | **File System Activity** | ❌ Not Implemented | File operations (create, delete, modify) |
| 1002 | Kernel Extension Activity | ❌ Not Implemented | Driver/kernel module operations |
| 1003 | Kernel Activity | ❌ Not Implemented | Kernel resource management |
| 1004 | Memory Activity | ❌ Not Implemented | Memory allocation, buffer overflows |
| 1005 | Module Activity | ❌ Not Implemented | DLL/library loading |
| 1006 | Scheduled Job Activity | ❌ Not Implemented | Cron, Task Scheduler events |
| 1007 | **Process Activity** | ❌ Not Implemented | Process lifecycle (launch, terminate) |
| 1008 | Event Log Activity | ❌ Not Implemented | Log tampering detection |
| 1009 | Script Activity | ❌ Not Implemented | PowerShell, bash script execution |
| 201001 | Registry Key Activity | ❌ Not Implemented | Windows registry operations |
| 201002 | Registry Value Activity | ❌ Not Implemented | Windows registry value changes |
| 201003 | Windows Resource Activity | ❌ Not Implemented | Windows object access |
| 201004 | Windows Service Activity | ❌ Not Implemented | Service control manager events |

**Priority**: HIGH - Core EDR/SIEM functionality depends on these events.

### 3. Network Activity (Priority: MEDIUM-HIGH)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 4001 | **Network Activity** | ❌ Not Implemented | **CLAIMED IN README** - Generic network connections |
| 4002 | HTTP Activity | ❌ Not Implemented | Web traffic, proxy logs |
| 4003 | DNS Activity | ❌ Not Implemented | DNS queries and responses |
| 4004 | DHCP Activity | ❌ Not Implemented | IP address assignments |
| 4005 | RDP Activity | ❌ Not Implemented | Remote desktop connections |
| 4006 | SMB Activity | ❌ Not Implemented | File share access |
| 4007 | SSH Activity | ❌ Not Implemented | SSH connections |
| 4008 | FTP Activity | ❌ Not Implemented | File transfer protocol |
| 4009 | Email Activity | ❌ Not Implemented | SMTP, email logs |
| 4013 | NTP Activity | ❌ Not Implemented | Time synchronization |
| 4014 | Tunnel Activity | ❌ Not Implemented | VPN, tunneling protocols |

**Priority**: MEDIUM-HIGH - Network monitoring is core SIEM functionality.

### 4. Application Activity (Priority: MEDIUM)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 6001 | Web Resources Activity | ❌ Not Implemented | Web app operations |
| 6002 | Application Lifecycle | ❌ Not Implemented | App install/start/stop |
| 6003 | API Activity | ❌ Not Implemented | **CLAIMED IN README** - API calls (AWS CloudTrail) |
| 6005 | Datastore Activity | ❌ Not Implemented | Database operations |
| 6006 | File Hosting Activity | ❌ Not Implemented | SharePoint, OneDrive, Box |
| 6007 | Scan Activity | ❌ Not Implemented | Antivirus/vulnerability scans |
| 6008 | Application Error | ❌ Not Implemented | Application exceptions |

**Priority**: MEDIUM - Important for cloud and application monitoring.

### 5. Findings (Priority: HIGH)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 2002 | Vulnerability Finding | ❌ Not Implemented | CVE discoveries |
| 2003 | Compliance Finding | ❌ Not Implemented | Compliance violations |
| 2004 | **Detection Finding** | ❌ Not Implemented | **CLAIMED IN README** - Security detections |
| 2005 | Incident Finding | ❌ Not Implemented | Security incident tracking |
| 2006 | Data Security Finding | ❌ Not Implemented | DLP alerts |
| 2007 | Application Security Posture Finding | ❌ Not Implemented | ASPM findings |
| 2008 | IAM Analysis Finding | ❌ Not Implemented | IAM risk analysis |

**Priority**: HIGH - Detection Finding (2004) is explicitly mentioned in README.

### 6. Discovery (Priority: LOW)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 5001 | Device Inventory Info | ❌ Not Implemented | Device discovery |
| 5003 | User Inventory Info | ❌ Not Implemented | User enumeration |
| 5004 | Operating System Patch State | ❌ Not Implemented | Patch compliance |
| 5019 | Device Config State Change | ❌ Not Implemented | Configuration changes |
| 5020 | Software Inventory Info | ❌ Not Implemented | Installed software |
| 5021 | OSINT Inventory Info | ❌ Not Implemented | Threat intelligence |
| 5023 | Cloud Resources Inventory Info | ❌ Not Implemented | Cloud asset discovery |
| 5040 | Live Evidence Info | ❌ Not Implemented | Forensic data collection |

**Priority**: LOW - Asset management, less critical for initial SIEM functionality.

### 7. Remediation (Priority: LOW)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 7001 | Remediation Activity | ❌ Not Implemented | Generic remediation |
| 7002 | File Remediation Activity | ❌ Not Implemented | File-based countermeasures |
| 7003 | Process Remediation Activity | ❌ Not Implemented | Process termination |
| 7004 | Network Remediation Activity | ❌ Not Implemented | Network isolation |

**Priority**: LOW - Advanced feature, D3FEND framework support.

### 8. Unmanned Systems (Priority: VERY LOW)

| Class UID | Event Class | Status | Notes |
|-----------|-------------|--------|-------|
| 8001 | Drone Flights Activity | ❌ Not Implemented | UAS/drone tracking |
| 8002 | Airborne Broadcast Activity | ❌ Not Implemented | ADS-B aircraft tracking |

**Priority**: VERY LOW - Niche use case, not typical SIEM requirement.

## Implementation Recommendations

### Phase 1: Core OCSF Compliance (CRITICAL)

**Goal**: Fix base Event structure to be OCSF-compliant

1. **Update `common/ocsf/ocsf/event.go`**:
   - Add missing required fields: `class_uid`, `category_uid`, `activity_id`, `type_uid`, `severity_id`
   - Add `metadata` object with `product`, `version`, `profiles[]`
   - Separate `time` (event occurrence) from `observed_time` (event collection)
   - Make Actor/Target structures optional and class-specific
   - Add `status` and `status_id` fields

2. **Create OCSF Constants Package** (`common/ocsf/ocsf/constants.go`):
   - Define enums for categories, classes, activities, severities, statuses
   - Provide type-safe constants instead of magic numbers
   - Add helper functions for `type_uid` calculation

3. **Update HEC Normalizer**:
   - Map to appropriate OCSF classes based on HEC metadata
   - Set correct `class_uid`, `category_uid`, `activity_id`
   - Populate metadata properly

### Phase 2: High-Priority Event Classes

Implement in this order based on SIEM criticality and README claims:

1. **Authentication (3002)** - Explicitly claimed in README
   - Login attempts, logouts, failed authentication
   - Most critical for security monitoring

2. **Network Activity (4001)** - Explicitly claimed in README
   - Generic network connections
   - Foundation for network security monitoring

3. **Detection Finding (2004)** - Explicitly claimed in README
   - Security detections and alerts
   - Core SIEM output format

4. **Process Activity (1007)** - Essential for EDR
   - Process creation, termination
   - Command line logging

5. **File System Activity (1001)** - Essential for EDR
   - File operations
   - Ransomware detection

### Phase 3: Extended Security Events

6. HTTP Activity (4002)
7. DNS Activity (4003)
8. API Activity (6003)
9. Compliance Finding (2003)
10. Account Change (3001)

### Phase 4: Advanced Features

- Remaining Network Activity classes (SMB, RDP, SSH, etc.)
- Windows-specific events (Registry, Services)
- Application events
- Discovery events
- Remediation events

## Implementation Approach

### 1. Create Class-Specific Normalizers

```
core/internal/normalizer/
├── authentication.go       # Authentication events
├── network_activity.go     # Network events
├── process_activity.go     # Process events
├── file_activity.go        # File system events
├── detection_finding.go    # Security findings
└── ...
```

### 2. Create OCSF Event Builders

```go
// common/ocsf/ocsf/builders/authentication.go
func NewAuthenticationEvent(activity AuthenticationActivity) *ocsf.Event {
    return &ocsf.Event{
        ClassUID:    3002,
        CategoryUID: 3,
        Class:       "authentication",
        Category:    "iam",
        ActivityID:  int(activity),
        TypeUID:     300200 + int(activity),
        // ... rest of fields
    }
}
```

### 3. Create Class-Specific Validators

```
core/internal/validator/
├── authentication_validator.go
├── network_validator.go
├── process_validator.go
└── ...
```

### 4. Update Documentation

- Document supported event classes
- Provide mapping examples
- Create integration guides

## Testing Requirements

For each implemented event class:

1. Unit tests for event builders
2. Unit tests for normalizers
3. Integration tests with sample data
4. OCSF schema validation tests
5. End-to-end pipeline tests

## Resources

- **OCSF Schema**: https://schema.ocsf.io/
- **OCSF GitHub**: https://github.com/ocsf/ocsf-schema
- **OCSF Documentation**: https://schema.ocsf.io/1.1.0/
- **Reference Implementations**: https://github.com/ocsf/ocsf-server

## Conclusion

**Current State**: TelHawk Stack has a solid architecture for OCSF normalization but **zero actual OCSF event class implementations**. The README claims support for 6 event categories, but none are implemented.

**Immediate Actions Required**:
1. Fix base Event structure to be OCSF-compliant (add missing required fields)
2. Implement Authentication (3002) - claimed in README
3. Implement Network Activity (4001) - claimed in README  
4. Implement Detection Finding (2004) - claimed in README
5. Update README to reflect actual implementation status

**Estimated Effort**:
- Phase 1 (Base Compliance): 1-2 weeks
- Phase 2 (5 priority classes): 3-4 weeks
- Phase 3 (10 extended classes): 4-6 weeks
- Phase 4 (Remaining classes): 8-12 weeks

**Total**: 4-6 months for comprehensive OCSF coverage of all 59 active event classes.
