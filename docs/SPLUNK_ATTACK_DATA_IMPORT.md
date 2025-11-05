# Splunk Attack Data Import Plan

## Overview

This document outlines the strategy for importing security datasets from [Splunk's attack_data repository](https://github.com/splunk/attack_data) into TelHawk Stack. This approach eliminates the need to manually build extensive Azure infrastructure for data collection by leveraging pre-built, MITRE ATT&CK-mapped security datasets.

## What is Splunk attack_data?

Splunk's attack_data is a curated repository of security datasets generated from real attack simulations, containing:

- **600+ MITRE ATT&CK technique datasets** organized by technique ID (T1003.001, etc.)
- **Multiple data sources per technique** (Sysmon, Security Event Logs, CrowdStrike, etc.)
- **Rich metadata** in YAML format describing each dataset
- **Real attack data** from tools like Atomic Red Team, malware samples, and APT simulations
- **Multiple formats** including honeypot data, suspicious behavior, and malware datasets

### Repository Structure
```
splunk/attack_data/
├── datasets/
│   ├── attack_techniques/     # MITRE ATT&CK mapped datasets
│   │   ├── T1003.001/        # OS Credential Dumping: LSASS Memory
│   │   ├── T1003.002/        # OS Credential Dumping: Security Account Manager
│   │   ├── T1003.003/        # OS Credential Dumping: NTDS
│   │   └── ...               # 600+ technique directories
│   ├── malware/              # Malware-specific datasets
│   ├── apt_simulations/      # APT campaign simulations
│   ├── honeypots/            # Honeypot captured data
│   └── suspicious_behaviour/ # General suspicious activity
└── bin/
    └── replay.py             # Splunk HEC replay tool
```

### Dataset Metadata Format
Each technique directory contains:
- **YAML metadata file** describing the dataset
- **Log files** (windows-sysmon.log, windows-security.log, crowdstrike_falcon.log, etc.)

Example YAML structure:
```yaml
author: Patrick Bareiss
id: cc9b25e1-efc9-11eb-926b-550bf0943fbb
date: '2020-10-08'
description: 'Atomic Test Results for T1003.003 NTDS dumping techniques'
environment: attack_range
directory: atomic_red_team
mitre_technique:
  - T1003.003
datasets:
  - name: windows-sysmon
    path: /datasets/attack_techniques/T1003.003/atomic_red_team/windows-sysmon.log
    sourcetype: XmlWinEventLog
    source: XmlWinEventLog:Microsoft-Windows-Sysmon/Operational
  - name: windows-security
    path: /datasets/attack_techniques/T1003.003/atomic_red_team/4688_windows-security.log
    sourcetype: XmlWinEventLog
    source: XmlWinEventLog:Security
```

## Benefits for TelHawk Stack

### 1. **Immediate Security Data Availability**
- No need to build Azure VMs, domain controllers, or attack infrastructure
- Access to 600+ pre-built attack scenarios
- Real security events from actual attack simulations

### 2. **MITRE ATT&CK Coverage**
- Comprehensive coverage across MITRE ATT&CK framework
- Maps directly to detection rules and analytics
- Enables threat hunting use case development

### 3. **Detection Development**
- Test OCSF normalization against real attack data
- Develop and validate security detections
- Build dashboards and alerts with realistic data

### 4. **Cost Savings**
- Eliminates Azure infrastructure costs for data generation
- No need for attack simulation licenses
- Reduces time to productive SIEM deployment

## Implementation Strategy

### Phase 1: Repository Setup & Dataset Selection

#### 1.1 Clone the attack_data Repository
```bash
cd /opt
git lfs install --skip-smudge
git clone https://github.com/splunk/attack_data
cd attack_data
```

#### 1.2 Select Initial Datasets
Start with high-value attack techniques:
- **T1003.001** - LSASS Memory Dumping
- **T1003.002** - SAM Credential Theft
- **T1003.003** - NTDS.dit Dumping
- **T1059.001** - PowerShell Execution
- **T1105** - Ingress Tool Transfer
- **T1136** - Create Account
- **T1543.003** - Windows Service Creation
- **T1562.001** - Disable or Modify Tools

Download specific datasets:
```bash
# Pull specific technique datasets
git lfs pull --include=datasets/attack_techniques/T1003.001/
git lfs pull --include=datasets/attack_techniques/T1003.002/
git lfs pull --include=datasets/attack_techniques/T1003.003/
git lfs pull --include=datasets/attack_techniques/T1059.001/

# Or pull all attack technique data (~9GB)
git lfs pull --include=datasets/attack_techniques/
```

### Phase 2: Build Import Pipeline

#### 2.1 Create Import Tool Structure
```bash
mkdir -p /home/ehorton/telhawk-stack/tools/attack-data-importer
cd /home/ehorton/telhawk-stack/tools/attack-data-importer
```

#### 2.2 Import Tool Components

Create a Go-based import tool with these capabilities:

**Core Features:**
1. **Dataset Discovery**
   - Scan attack_data directories for YAML files
   - Parse metadata to understand dataset structure
   - Validate dataset integrity

2. **Format Parsing**
   - Parse Windows XML Event Logs
   - Handle JSON-formatted logs
   - Support raw text logs
   - Parse Sysmon, Security, PowerShell logs

3. **HEC Ingestion**
   - Send events to TelHawk ingest service via HEC API
   - Batch events for efficiency
   - Handle backpressure and retries
   - Track ingestion progress

4. **Metadata Enrichment**
   - Add MITRE ATT&CK technique tags
   - Include dataset provenance information
   - Tag with attack_data source metadata

**Tool Architecture:**
```
attack-data-importer/
├── cmd/
│   └── importer/
│       └── main.go              # CLI entry point
├── internal/
│   ├── scanner/
│   │   └── scanner.go           # Directory/YAML scanner
│   ├── parser/
│   │   ├── evtx.go              # Windows Event XML parser
│   │   ├── json.go              # JSON log parser
│   │   └── syslog.go            # Syslog/text parser
│   ├── enricher/
│   │   └── enricher.go          # MITRE/metadata enrichment
│   └── ingest/
│       └── hec.go               # HEC client for ingestion
├── config.yaml                   # Configuration file
└── README.md
```

#### 2.3 Configuration Example

**config.yaml:**
```yaml
attack_data:
  repository_path: /opt/attack_data
  datasets_path: /opt/attack_data/datasets/attack_techniques

telhawk:
  ingest_url: http://localhost:8088
  hec_token: ${TELHAWK_HEC_TOKEN}
  index: attack_data
  batch_size: 100
  workers: 4

parsing:
  # Handle Windows XML Event Logs
  evtx_enabled: true
  # Parse standard JSON logs
  json_enabled: true
  # Parse raw text/syslog
  text_enabled: true

enrichment:
  add_mitre_tags: true
  add_dataset_metadata: true
  add_provenance: true
```

### Phase 3: Data Parsing & Transformation

#### 3.1 Windows Event Log Parsing
Windows Event Logs (EVTX) need special handling:

```go
// Example parsing logic
type EventLogParser struct {
    // Parse Windows XML Event Log format
}

func (p *EventLogParser) Parse(data []byte) ([]Event, error) {
    // Parse <Event xmlns="..."> structure
    // Extract:
    // - EventID
    // - TimeCreated
    // - Computer
    // - EventData fields
    // - System fields
}
```

#### 3.2 Event Transformation for HEC

Transform parsed events to HEC format:
```json
{
  "time": 1617187200.0,
  "host": "DC01.contoso.local",
  "source": "attack_data",
  "sourcetype": "XmlWinEventLog:Security",
  "index": "attack_data",
  "event": {
    "EventID": 4688,
    "ProcessName": "C:\\Windows\\System32\\mimikatz.exe",
    "CommandLine": "mimikatz.exe privilege::debug sekurlsa::logonpasswords",
    // ... other fields
  },
  "fields": {
    "mitre_technique": "T1003.001",
    "mitre_tactic": "Credential Access",
    "attack_data_source": "atomic_red_team",
    "dataset_id": "cc9b25e1-efc9-11eb-926b-550bf0943fbb",
    "attack_data_author": "Patrick Bareiss"
  }
}
```

#### 3.3 MITRE ATT&CK Enrichment

Add ATT&CK metadata to every event:
- **Technique ID** (T1003.001)
- **Technique Name** (LSASS Memory)
- **Tactic** (Credential Access)
- **Sub-technique** (if applicable)

### Phase 4: Import Execution

#### 4.1 Command-Line Interface

```bash
# Import specific technique
./attack-data-importer import \
  --technique T1003.001 \
  --hec-token abc123... \
  --index attack_data

# Import multiple techniques
./attack-data-importer import \
  --techniques T1003.001,T1003.002,T1059.001 \
  --hec-token abc123...

# Import all credential access techniques
./attack-data-importer import \
  --tactic "Credential Access" \
  --hec-token abc123...

# Dry-run to validate without ingesting
./attack-data-importer import \
  --technique T1003.001 \
  --dry-run

# Import with progress tracking
./attack-data-importer import \
  --techniques T1003.001 \
  --progress \
  --output import-report.json
```

#### 4.2 Import Process Flow

```
1. Scan Repository
   └─> Find YAML metadata files
   └─> Parse dataset definitions

2. For Each Dataset:
   └─> Read YAML metadata
   └─> Locate log files
   └─> Parse log format (EVTX/JSON/text)
   └─> Enrich with MITRE tags
   └─> Batch events (100/batch)
   └─> Send to HEC endpoint
   └─> Track progress/errors

3. Generate Report
   └─> Events ingested
   └─> Techniques covered
   └─> Any errors/warnings
```

#### 4.3 Progress Tracking

```json
{
  "import_session": "2024-11-05T10:45:00Z",
  "techniques_imported": [
    {
      "technique_id": "T1003.001",
      "events_ingested": 1234,
      "datasets": ["windows-sysmon", "windows-security"],
      "errors": 0
    }
  ],
  "total_events": 5678,
  "total_bytes": 12345678,
  "duration_seconds": 45,
  "errors": []
}
```

### Phase 5: OpenSearch Index Configuration

#### 5.1 Create Dedicated Index

Create a separate index for attack_data:
```json
PUT /attack_data
{
  "settings": {
    "number_of_shards": 2,
    "number_of_replicas": 1,
    "index.mapping.total_fields.limit": 5000
  },
  "mappings": {
    "properties": {
      "@timestamp": {"type": "date"},
      "host": {"type": "keyword"},
      "source": {"type": "keyword"},
      "sourcetype": {"type": "keyword"},
      "mitre_technique": {"type": "keyword"},
      "mitre_tactic": {"type": "keyword"},
      "dataset_id": {"type": "keyword"},
      "EventID": {"type": "integer"},
      "ProcessName": {"type": "text", "fields": {"keyword": {"type": "keyword"}}},
      "CommandLine": {"type": "text"},
      "event": {"type": "object", "enabled": true}
    }
  }
}
```

#### 5.2 Index Templates

Create index template for future attack_data imports:
```json
PUT _index_template/attack_data_template
{
  "index_patterns": ["attack_data*"],
  "template": {
    "settings": {
      "number_of_shards": 2,
      "number_of_replicas": 1
    },
    "mappings": {
      "properties": {
        "mitre_technique": {"type": "keyword"},
        "mitre_tactic": {"type": "keyword"}
      }
    }
  }
}
```

### Phase 6: Validation & Testing

#### 6.1 Validate Import
```bash
# Check event count
curl -u admin:password http://localhost:9200/attack_data/_count

# Query by technique
curl -X POST http://localhost:9200/attack_data/_search -H 'Content-Type: application/json' -d '
{
  "query": {
    "term": {"mitre_technique": "T1003.001"}
  }
}'

# Check MITRE coverage
curl -X POST http://localhost:9200/attack_data/_search -H 'Content-Type: application/json' -d '
{
  "size": 0,
  "aggs": {
    "techniques": {
      "terms": {"field": "mitre_technique", "size": 100}
    }
  }
}'
```

#### 6.2 Test OCSF Normalization

Verify that imported events are normalized:
```bash
# Query normalized events
curl -X POST http://localhost:9200/attack_data/_search -d '
{
  "query": {"exists": {"field": "ocsf.class_name"}}
}'
```

#### 6.3 Build Test Dashboards

Create dashboards to visualize imported data:
- **MITRE ATT&CK Coverage** - Heatmap of techniques
- **Attack Timeline** - Chronological event visualization
- **Top Techniques** - Bar chart of most common techniques
- **Event Sources** - Pie chart of data sources (Sysmon, Security, etc.)

### Phase 7: Automation & Scheduling

#### 7.1 Scheduled Imports

Create systemd timer or cron job for periodic imports:

```bash
# /etc/cron.d/attack-data-import
# Daily import of new datasets
0 2 * * * telhawk /opt/telhawk/attack-data-importer import --new-datasets --hec-token-file /etc/telhawk/hec.token
```

#### 7.2 Continuous Integration

```yaml
# .github/workflows/import-attack-data.yml
name: Import New Attack Data
on:
  schedule:
    - cron: '0 2 * * 0'  # Weekly on Sunday
  workflow_dispatch:

jobs:
  import:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Pull latest attack_data
        run: |
          cd /opt/attack_data
          git pull
          git lfs pull --include=datasets/attack_techniques/
      - name: Run import
        run: |
          ./attack-data-importer import \
            --new-datasets \
            --hec-token ${{ secrets.HEC_TOKEN }}
```

## Implementation Timeline

### Week 1: Repository Setup & Tool Development
- Clone attack_data repository
- Design import tool architecture
- Implement basic YAML parsing
- Build HEC client

### Week 2: Parser Development
- Implement Windows Event Log parser
- Implement JSON parser
- Implement text/syslog parser
- Add MITRE enrichment logic

### Week 3: Integration & Testing
- Integrate with TelHawk ingest service
- Test with sample datasets
- Validate OCSF normalization
- Build progress tracking

### Week 4: Production Deployment
- Import initial dataset collection
- Configure OpenSearch indices
- Build validation dashboards
- Document operational procedures

## Dataset Prioritization

### Phase 1: Initial Import (High-Value Techniques)
Focus on common attack techniques:
1. **Credential Access** (T1003.x) - Credential dumping
2. **Execution** (T1059.x) - Command/script execution
3. **Persistence** (T1543.x, T1547.x) - Service/registry persistence
4. **Defense Evasion** (T1562.x) - Disabling security tools
5. **Lateral Movement** (T1021.x) - Remote services

**Estimated Size:** ~2-3 GB, ~100,000 events

### Phase 2: Extended Coverage (Common Tactics)
6. **Privilege Escalation** (T1055.x, T1134.x)
7. **Discovery** (T1082, T1083, T1087.x)
8. **Collection** (T1005, T1114.x)
9. **Exfiltration** (T1041, T1567.x)
10. **Impact** (T1485, T1486)

**Estimated Size:** ~4-5 GB, ~200,000 events

### Phase 3: Full Coverage
- All 600+ technique datasets
- APT simulation datasets
- Malware behavior datasets
- Honeypot data

**Estimated Size:** ~9 GB+, ~500,000+ events

## Operational Considerations

### Storage Requirements
- **Per technique:** ~10-50 MB (varies)
- **100 techniques:** ~2-3 GB
- **Full repository:** ~9 GB
- **OpenSearch overhead:** 1.5-2x raw data

### Performance
- **Import speed:** ~1,000-5,000 events/sec
- **100k events:** ~30-60 seconds
- **Full import:** ~1-2 hours

### Maintenance
- **Weekly sync:** Check for new datasets in attack_data repo
- **Monthly full import:** Re-import updated datasets
- **Quarterly review:** Validate MITRE coverage and detection gaps

## Integration with Detection Development

### Use Cases

#### 1. Detection Rule Testing
```bash
# Import technique dataset
./attack-data-importer import --technique T1003.001

# Run detection rules against dataset
thawk query search 'index=attack_data mitre_technique=T1003.001 | detect_mimikatz'
```

#### 2. Threat Hunting
```bash
# Hunt for credential access patterns
thawk query search '
  index=attack_data 
  mitre_tactic="Credential Access"
  | stats count by mitre_technique, ProcessName
'
```

#### 3. Analytics Development
- Use real attack data to tune detection thresholds
- Identify false positive patterns
- Build behavioral baselines

### Detection Content Mapping

Map imported datasets to detection rules:
```yaml
detection:
  name: Mimikatz Execution Detected
  mitre_attack:
    - T1003.001
  data_sources:
    - attack_data:T1003.001:windows-sysmon
    - attack_data:T1003.001:windows-security
  search: |
    index=attack_data sourcetype=XmlWinEventLog:Sysmon
    EventID=1 Image=*mimikatz.exe
```

## Alternative: Using Splunk's Replay Tool

Splunk provides a Python tool for replaying datasets directly:

```bash
# Install dependencies
cd /opt/attack_data
pip install -r bin/requirements.txt

# Replay to TelHawk HEC endpoint
python bin/replay.py \
  --host http://localhost:8088 \
  --token abc123... \
  --sourcetype XmlWinEventLog:Sysmon \
  datasets/attack_techniques/T1003.001/atomic_red_team/windows-sysmon.log
```

**Pros:**
- Quick and simple
- Official Splunk tool
- Well-tested

**Cons:**
- Less control over enrichment
- Limited customization
- Python dependency

**Recommendation:** Use replay.py for quick testing, build custom importer for production.

## Success Metrics

Track these metrics to measure success:

### Coverage Metrics
- **Techniques Imported:** Target 100+ techniques
- **Events Ingested:** Target 100k+ events
- **Data Sources:** Sysmon, Security, PowerShell, CrowdStrike, etc.

### Operational Metrics
- **Import Success Rate:** >99%
- **Detection Coverage:** Map datasets to detection rules
- **Query Performance:** <1s for technique-specific queries

### Quality Metrics
- **OCSF Normalization:** 100% of events normalized
- **MITRE Tagging:** 100% of events tagged
- **Data Integrity:** Zero corruption errors

## Next Steps

1. **Immediate:** Clone attack_data repository and explore datasets
2. **Week 1:** Build basic import tool with YAML parsing
3. **Week 2:** Implement Windows Event Log parsing
4. **Week 3:** Test full import pipeline with sample techniques
5. **Week 4:** Production deployment and validation

## Conclusion

Importing Splunk's attack_data provides TelHawk Stack with immediate access to high-quality security datasets without the overhead of building and maintaining attack simulation infrastructure. This approach accelerates detection development, enables threat hunting, and provides realistic data for OCSF normalization testing.

## References

- [Splunk attack_data Repository](https://github.com/splunk/attack_data)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [Splunk HEC API Documentation](https://docs.splunk.com/Documentation/Splunk/latest/Data/UsetheHTTPEventCollector)
- [TelHawk Ingest Service Documentation](../ingest/README.md)
- [OCSF Schema Documentation](../docs/OCSF_COVERAGE.md)
