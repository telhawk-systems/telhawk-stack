# Nonrepudiation Strategy

TelHawk Stack implements comprehensive nonrepudiation controls to ensure all actions are attributable, tamper-proof, and auditable.

## Authentication Service

### Audit Logging
Every authentication action is logged with:
- **Actor identification**: User ID, username, actor type (user/service/system)
- **Action details**: What was done (login, register, token operations)
- **Context**: IP address, user agent, timestamp (UTC)
- **Result**: Success or failure with reason
- **HMAC signature**: Tamper-proof cryptographic signature

#### Logged Actions
- User registration
- Login attempts (success and failure)
- Token refresh
- Token validation
- Token revocation
- HEC token creation
- HEC token validation
- User updates
- Password changes

### Signature Verification
Each audit log entry is signed with HMAC-SHA256:
```
signature = HMAC-SHA256(id + timestamp + actorID + action + resource + result, secret_key)
```

This prevents:
- Log tampering
- Backdating entries
- Unauthorized modifications

## Event Ingestion

### Event Signatures
Every ingested event receives:
- **Unique ID**: UUID for tracking
- **Ingestion timestamp**: When received (immutable)
- **Source IP**: Origin of the event
- **HMAC signature**: Cryptographic proof of authenticity

### Ingestion Logs
Each ingestion request creates a tamper-proof log:
- HEC token used
- Source IP
- Number of events
- Bytes received
- Success/failure status
- HMAC signature

## Query Service

### Query Audit Trail
All searches are logged:
- Who ran the query (user ID)
- Query text (full SPL)
- Time range queried
- Number of results
- Timestamp
- Source IP

## Storage

### Immutable Writes
- Events written to OpenSearch with append-only semantics
- Original raw event preserved alongside normalized OCSF
- Indexing timestamp recorded separately from event timestamp

### Index Lifecycle
- Indices are read-only after rollover
- Snapshots taken before any retention deletion
- Deletion operations logged with audit trail

## Compliance Benefits

### GDPR/CCPA
- Right to erasure: Audit trail shows what was deleted and when
- Right to access: Complete history of data processing

### SOX/HIPAA
- Complete audit trail of who accessed what data
- Tamper-proof logging prevents evidence destruction
- Time-stamped actions establish sequence of events

### PCI DSS
- 10.1: Audit trails for all security events
- 10.2: Automated logging of authentication
- 10.3: Audit log entries include user, event type, date/time, source
- 10.5: Audit trails protected from unauthorized modifications

### SOC 2
- Trust Service Criteria CC7.2: System monitoring
- Trust Service Criteria CC7.3: Evaluation of security events

## Implementation Details

### Auth Service Audit Log Format
```json
{
  "id": "uuid",
  "timestamp": "2024-11-01T12:34:56.789Z",
  "actor_type": "user",
  "actor_id": "user-uuid",
  "actor_username": "analyst1",
  "action": "login",
  "resource": "session",
  "resource_id": "session-uuid",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "result": "success",
  "reason": "",
  "metadata": {
    "session_id": "session-uuid",
    "roles": ["analyst"]
  },
  "signature": "abc123..."
}
```

### Event Signature Format
```json
{
  "id": "event-uuid",
  "timestamp": "2024-11-01T12:34:56.789Z",
  "source": "firewall-01",
  "source_ip": "10.0.0.1",
  "event_data": {...},
  "signature": "def456...",
  "ingested": "2024-11-01T12:34:57.000Z"
}
```

## Verification

### Audit Log Verification
```bash
# Verify audit log integrity
thawk audit verify --from 2024-01-01 --to 2024-12-31

# Export audit logs for compliance
thawk audit export --format json --output audit-2024.json
```

### Event Verification
```bash
# Verify event signatures
thawk events verify --index security-2024.11.01

# Check for tampered events
thawk events integrity-check --all
```

## Chain of Custody

1. **Ingestion**: Event received → signed → stored with metadata
2. **Processing**: OCSF normalization → preserves original + signature
3. **Storage**: Indexed in OpenSearch → signature stored in metadata
4. **Query**: Results include original signature for verification
5. **Export**: Signatures included in exported data for forensics

## Forensic Analysis

All signatures enable:
- **Provenance**: Trace event back to origin
- **Timeline**: Establish exact sequence of events
- **Attribution**: Identify who did what
- **Integrity**: Prove data hasn't been tampered with

## Key Management

**Production**: Use proper secrets management (HashiCorp Vault, AWS Secrets Manager)

**Development**: Secrets hardcoded (replace before production)

Required secrets:
- `AUTH_AUDIT_SECRET`: For authentication audit logs
- `EVENT_SIGNATURE_SECRET`: For event signatures
- `INGESTION_LOG_SECRET`: For ingestion logs

## Future Enhancements

- [ ] PostgreSQL audit log storage with write-once tables
- [ ] Audit log export to WORM storage
- [ ] Blockchain anchoring for long-term integrity
- [ ] Digital signatures with PKI for user actions
- [ ] Audit log forwarding to external SIEM
