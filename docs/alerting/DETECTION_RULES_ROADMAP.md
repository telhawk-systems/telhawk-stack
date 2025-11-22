# Detection Rules Roadmap

This document outlines the TelHawk detection rule expansion plan and future enhancements.

## Current Status (Phase 1-5 Complete)

### Summary Statistics

- **Total Rules**: 36 detection rules
- **Immediately Testable**: 35 rules (97%)
- **OCSF Event Classes Covered**: 8 classes
  - 3002 - Authentication
  - 3001 - Account Change
  - 1007 - Process Activity
  - 4001 - Network Activity
  - 4003 - DNS Activity
  - 4002 - HTTP Activity
  - 4006 - File Activity
  - (Plus existing detection finding support)
- **MITRE ATT&CK Tactics**: 13 tactics covered
- **MITRE ATT&CK Techniques**: 60+ techniques

### Completed Phases

#### Phase 1: Enhanced Existing Event Classes (8 rules)
✅ **Authentication (3002)**
- successful_login_after_failures.json
- off_hours_authentication.json
- impossible_travel.json

✅ **Process Activity (1007)**
- lolbins_execution.json
- wmi_execution_abuse.json
- obfuscated_powershell.json

✅ **Network Activity (4001)**
- smb_share_enumeration.json
- rdp_connection_spike.json

#### Phase 2: DNS Detection (4 rules)
✅ **DNS Activity (4003)** - Seeder enhanced
- dns_tunneling.json
- dga_detection.json
- dns_exfiltration.json
- dns_query_spike.json

#### Phase 3: HTTP Detection (4 rules)
✅ **HTTP Activity (4002)** - Seeder enhanced
- web_shell_detection.json
- http_c2_beaconing.json
- sql_injection_attempts.json
- suspicious_user_agents.json

#### Phase 4: File Activity Detection (4 rules)
✅ **File Activity (4006)** - Seeder enhanced
- mass_file_modifications.json
- sensitive_file_access.json
- temp_executable_creation.json
- mass_file_deletion.json

#### Phase 5: Defense Evasion (3 rules)
✅ **Using Process Activity (1007)**
- security_tool_termination.json
- command_history_clearing.json
- suspicious_temp_script_execution.json

---

## Phase 6: Advanced Correlation Rules (Future)

These rules require **temporal_ordered** or **temporal** correlation support in the seeder. Implementation requires Phase 2 seeder enhancements.

### Persistence Detection (OCSF 1006 - Scheduled Job Activity)

**1. Scheduled Task Creation After Compromise**
```json
{
  "name": "scheduled_task_persistence",
  "description": "Detects scheduled task creation following failed authentication - potential persistence mechanism",
  "correlation_type": "temporal_ordered",
  "events": [
    {
      "class_uid": 3002,
      "status_id": 2,
      "time_window": "1h"
    },
    {
      "class_uid": 1006,
      "activity_id": 1,
      "time_window": "30m"
    }
  ],
  "mitre_attack": {
    "tactics": ["Persistence"],
    "techniques": [
      "T1053.005 - Scheduled Task/Job: Scheduled Task",
      "T1053.003 - Scheduled Task/Job: Cron"
    ]
  }
}
```

**2. Service Creation for Persistence**
```json
{
  "name": "service_persistence",
  "description": "Detects Windows service creation for persistence after initial access",
  "correlation_type": "temporal_ordered",
  "events": [
    {
      "class_uid": 3002,
      "status_id": 1,
      "actor.user.name": "suspicious"
    },
    {
      "class_uid": 1007,
      "process.name": "sc.exe",
      "process.cmd_line": "create"
    }
  ],
  "mitre_attack": {
    "tactics": ["Persistence", "Privilege Escalation"],
    "techniques": [
      "T1543.003 - Create or Modify System Process: Windows Service"
    ]
  }
}
```

### Multi-Stage Attack Chains

**3. Credential Theft to Lateral Movement**
```json
{
  "name": "credential_theft_to_lateral_movement",
  "description": "Detects credential file access followed by lateral movement attempts",
  "correlation_type": "temporal_ordered",
  "events": [
    {
      "class_uid": 4006,
      "file.path": ["SAM", "NTDS", "shadow"],
      "activity_id": 2
    },
    {
      "class_uid": 4001,
      "dst_endpoint.port": [445, 3389, 22]
    }
  ],
  "mitre_attack": {
    "tactics": ["Credential Access", "Lateral Movement"],
    "techniques": [
      "T1003 - OS Credential Dumping",
      "T1021 - Remote Services"
    ]
  }
}
```

**4. Reconnaissance to Exploitation**
```json
{
  "name": "recon_to_exploitation",
  "description": "Detects port scanning followed by exploitation attempts",
  "correlation_type": "temporal",
  "events": [
    {
      "class_uid": 4001,
      "count_unique": ".dst_endpoint.port",
      "threshold": 10
    },
    {
      "class_uid": 4002,
      "http_response.code": [500, 403]
    }
  ],
  "mitre_attack": {
    "tactics": ["Discovery", "Initial Access"],
    "techniques": [
      "T1046 - Network Service Scanning",
      "T1190 - Exploit Public-Facing Application"
    ]
  }
}
```

---

## Phase 7: Registry Activity (Windows) - Future

**Requires**: New OCSF event class support or Windows-specific event parsing

### Registry Persistence Detection

**5. Registry Run Key Modification**
- Detects modifications to Windows registry run keys for persistence
- MITRE: T1547.001 - Boot or Logon Autostart Execution: Registry Run Keys

**6. UAC Bypass Detection**
- Detects registry modifications to bypass User Account Control
- MITRE: T1548.002 - Abuse Elevation Control Mechanism: Bypass User Account Control

---

## Phase 8: Kerberos-Specific Attacks - Future

**Requires**: Enhanced authentication event parsing with Kerberos protocol details

### Advanced Authentication Attacks

**7. Kerberoasting Detection**
- Detects excessive TGS requests for service accounts
- MITRE: T1558.003 - Steal or Forge Kerberos Tickets: Kerberoasting

**8. Golden/Silver Ticket Detection**
- Detects anomalous Kerberos ticket usage patterns
- MITRE: T1558.001 - Steal or Forge Kerberos Tickets: Golden Ticket

---

## Phase 9: Cloud API Activity (OCSF 6003) - Future

**Requires**: Integration with cloud provider APIs (AWS, Azure, GCP)

### Cloud-Specific Detection

**9. Unusual Cloud API Calls**
- Detects suspicious AWS/Azure/GCP API operations
- MITRE: T1078.004 - Valid Accounts: Cloud Accounts

**10. Cloud Resource Enumeration**
- Detects reconnaissance via cloud management APIs
- MITRE: T1580 - Cloud Infrastructure Discovery

---

## Phase 10: Email Activity (OCSF 4009-4011) - Future

**Requires**: Email log integration (Office 365, Exchange, Gmail)

### Email-Based Threats

**11. Phishing Detection**
- Detects emails with malicious attachments or URLs
- MITRE: T1566 - Phishing

**12. Email Exfiltration Rules**
- Detects mass email forwarding or unusual external emails
- MITRE: T1114 - Email Collection

---

## Seeder Enhancement Roadmap

### Phase 2 Enhancements (Required for Phase 6)

**Temporal Correlation Support**
- Implement `temporal_ordered` event generation
- Implement `temporal` (any order) event generation
- Implement `join` (field correlation) event generation

**Expected Development Time**: 5-7 days

### Phase 3 Enhancements (For Registry/Kerberos)

**New Event Class Support**
- Windows Registry activity events
- Kerberos-specific authentication fields
- Service creation/modification events

**Expected Development Time**: 3-4 days

---

## Coverage Gaps Summary

### High-Priority Gaps (Not Yet Covered)

1. **Registry Activity** (Windows persistence, UAC bypass)
2. **Kerberos Attacks** (Kerberoasting, Golden/Silver tickets)
3. **Event Log Manipulation** (Log clearing, tampering)
4. **Memory Injection** (Process injection, DLL injection)
5. **Kernel Module Loading** (Rootkits, driver installation)
6. **Cloud-Specific Attacks** (API abuse, resource manipulation)
7. **Email-Based Threats** (Phishing, exfiltration)
8. **Lateral Movement Chains** (Multi-hop movement detection)

### Medium-Priority Gaps

9. **DHCP Activity** (Network reconnaissance)
10. **NTP Activity** (Time manipulation)
11. **Tunnel Activity** (Protocol tunneling detection)
12. **VPN Activity** (Unauthorized VPN usage)
13. **Remote Desktop Activity** (Beyond connection spikes)
14. **FTP Activity** (Data exfiltration, unauthorized transfers)

---

## Testing and Validation

### Completed Testing

✅ All 35 immediately testable rules validated with seeder
- Event generation matches rule filters
- Threshold values trigger correctly
- Group-by fields generate appropriate correlation
- OCSF compliance verified

### Future Testing Requirements

**For Phase 6 Rules**:
- Temporal correlation logic validation
- Event sequence ordering verification
- Time window boundary testing

**For Cloud/Email Rules**:
- Integration testing with external data sources
- API rate limiting considerations
- Field mapping validation

---

## Metrics and Success Criteria

### Current Coverage

- **OCSF Event Classes**: 8 of ~77 (10.4%)
- **MITRE ATT&CK Techniques**: 60+ techniques
- **MITRE ATT&CK Tactics**: 13 of 14 (93%)
  - Missing: Resource Development

### Target Coverage (All Phases Complete)

- **OCSF Event Classes**: 15+ critical classes (20%)
- **MITRE ATT&CK Techniques**: 100+ techniques
- **MITRE ATT&CK Tactics**: 14 of 14 (100%)
- **Detection Rule Count**: 50+ rules

---

## Implementation Priority

### Immediate (Next Sprint)

1. Deploy Phase 1-5 rules to production
2. Monitor alert volume and tune thresholds
3. Collect feedback from SOC analysts

### Short-term (1-2 months)

1. Implement temporal correlation in seeder (Phase 2 enhancement)
2. Create Phase 6 advanced correlation rules
3. Add Windows registry event parsing

### Medium-term (3-6 months)

1. Integrate cloud API logging (AWS CloudTrail, Azure Activity Log)
2. Add email log ingestion (Office 365, Exchange)
3. Implement Phase 9-10 rules

### Long-term (6-12 months)

1. Machine learning-based baseline deviation detection
2. Behavioral analytics for insider threat detection
3. Automated threat hunting workflows

---

## Maintenance and Updates

### Quarterly Reviews

- Review rule effectiveness (true positive vs false positive rates)
- Update thresholds based on operational feedback
- Add new rules for emerging threats (CVEs, TTPs)
- Update MITRE ATT&CK technique mappings

### Annual Reviews

- Comprehensive gap analysis against MITRE ATT&CK
- Seeder enhancement planning
- OCSF schema version updates
- Integration with new data sources

---

## Related Documentation

- `/alerting/rules/*.json` - Current detection rule definitions
- `cli/internal/seeder/` - Event seeder implementation
- `docs/SERVICES.md` - Detection service architecture
- `CLAUDE.md` - Development guidelines and seeder usage
