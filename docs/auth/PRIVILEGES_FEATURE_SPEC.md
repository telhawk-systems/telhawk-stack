# TelHawk Privileges and Permissions Feature Document

**Version:** 1.0  
**Status:** Design Specification  
**Target:** Release Candidate (RC) versions  
**Last Updated:** 2025-11-06

---

## Executive Summary

This document defines a comprehensive, enterprise-grade privilege and permission system for TelHawk Stack. The system provides fine-grained access control while remaining intuitive for administrators and auditable for compliance requirements.

---

## Design Principles

1. **Least Privilege by Default** - Users receive minimum permissions needed for their role
2. **Defense in Depth** - Multiple layers of authorization checks
3. **Audit Everything** - All permission checks and changes are logged
4. **Explicit Deny** - Absence of permission means denial
5. **Performance First** - Permission checks are fast and cacheable
6. **Future-Proof** - Design allows for additional permission types without breaking changes

---

## Permission Model Overview

TelHawk uses a three-tier permission model:

```
Tier 1: Roles (Coarse-grained)
    ↓
Tier 2: Capabilities (Fine-grained actions)
    ↓
Tier 3: Resource ACLs (Object-level access)
```

---

## Tier 1: Role-Based Access Control (RBAC)

### Default System Roles

#### **admin**
- **Purpose**: Full system administration
- **Use Case**: System administrators, security leads
- **Capabilities**: ALL
- **Scope**: Global, all resources
- **Typical Count**: 2-5 per organization

#### **security_analyst**
- **Purpose**: Security operations and investigation
- **Use Case**: SOC analysts, threat hunters
- **Capabilities**:
  - Search and analyze all security data
  - Create and manage alerts
  - Create and share dashboards
  - Export search results
  - View audit logs
- **Scope**: All indexes, own objects + shared
- **Typical Count**: 10-50 per organization

#### **analyst**
- **Purpose**: General data analysis without security admin
- **Use Case**: Business analysts, data scientists
- **Capabilities**:
  - Search assigned indexes
  - Create personal dashboards and searches
  - Export limited data
- **Scope**: Assigned indexes only, own objects
- **Typical Count**: 50-200 per organization

#### **viewer**
- **Purpose**: Read-only access to dashboards and reports
- **Use Case**: Management, stakeholders
- **Capabilities**:
  - View shared dashboards
  - Run predefined searches
  - View alerts
- **Scope**: Shared objects only, no data export
- **Typical Count**: 100-1000 per organization

#### **ingester**
- **Purpose**: Data ingestion only, no search
- **Use Case**: Application services, log forwarders
- **Capabilities**:
  - Submit events via HEC
  - Create HEC tokens
- **Scope**: Write-only to assigned indexes
- **Typical Count**: 50-500 per organization

#### **compliance_auditor**
- **Purpose**: Audit and compliance review
- **Use Case**: Compliance officers, external auditors
- **Capabilities**:
  - View audit logs
  - Run compliance reports
  - View user activity
  - Read-only system configuration
- **Scope**: Audit data only, no operational data
- **Typical Count**: 2-10 per organization

### Custom Roles

Organizations can create custom roles by combining capabilities:

```json
{
  "name": "ir_responder",
  "description": "Incident Response Team Member",
  "capabilities": [
    "search:execute",
    "search:export",
    "alerts:create",
    "alerts:edit_own",
    "cases:create",
    "cases:edit_all",
    "data:delete"
  ],
  "index_access": ["security-*", "firewall-*", "endpoint-*"],
  "inherited_from": ["security_analyst"]
}
```

---

## Tier 2: Capability-Based Permissions

### Capability Naming Convention

Format: `resource:action[:scope]`

Examples:
- `search:execute` - Basic search capability
- `users:update` - Update user accounts
- `dashboards:edit_own` - Edit own dashboards
- `alerts:edit_all` - Edit any alert

### Capability Catalog

#### **Authentication & User Management**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `auth:login` | Log into the system | ALL |
| `auth:impersonate` | Log in as another user | admin |
| `users:list` | View list of users | admin, security_analyst |
| `users:create` | Create new users | admin |
| `users:update` | Modify user accounts | admin |
| `users:delete` | Delete user accounts | admin |
| `users:reset_password` | Reset user passwords | admin |
| `users:view_activity` | View user activity logs | admin, compliance_auditor |

#### **Search & Query**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `search:execute` | Execute search queries | admin, security_analyst, analyst |
| `search:export` | Export search results | admin, security_analyst, analyst |
| `search:export_unlimited` | Export without row limits | admin |
| `search:realtime` | Execute real-time searches | admin, security_analyst |
| `search:delete_by_keyword` | Delete events matching criteria | admin |
| `search:access_raw` | Access raw event data | admin, security_analyst |
| `search:statistical_only` | Only statistical results, no raw data | analyst |

#### **Saved Searches**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `saved_searches:create` | Create saved searches | admin, security_analyst, analyst |
| `saved_searches:edit_own` | Edit own saved searches | admin, security_analyst, analyst |
| `saved_searches:edit_all` | Edit any saved search | admin |
| `saved_searches:delete_own` | Delete own saved searches | admin, security_analyst, analyst |
| `saved_searches:delete_all` | Delete any saved search | admin |
| `saved_searches:share` | Share searches with others | admin, security_analyst |
| `saved_searches:schedule` | Schedule automated searches | admin, security_analyst |

#### **Dashboards**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `dashboards:view_shared` | View shared dashboards | ALL (except ingester) |
| `dashboards:create` | Create dashboards | admin, security_analyst, analyst |
| `dashboards:edit_own` | Edit own dashboards | admin, security_analyst, analyst |
| `dashboards:edit_all` | Edit any dashboard | admin |
| `dashboards:delete_own` | Delete own dashboards | admin, security_analyst, analyst |
| `dashboards:delete_all` | Delete any dashboard | admin |
| `dashboards:share` | Share dashboards | admin, security_analyst |
| `dashboards:publish` | Publish to home page | admin |

#### **Alerts & Correlation**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `alerts:view` | View alerts | admin, security_analyst, analyst, viewer |
| `alerts:create` | Create alert rules | admin, security_analyst |
| `alerts:edit_own` | Edit own alert rules | admin, security_analyst |
| `alerts:edit_all` | Edit any alert rule | admin |
| `alerts:delete_own` | Delete own alerts | admin, security_analyst |
| `alerts:delete_all` | Delete any alert | admin |
| `alerts:trigger` | Manually trigger alerts | admin, security_analyst |
| `alerts:suppress` | Suppress alert notifications | admin, security_analyst |
| `alerts:acknowledge` | Acknowledge alerts | admin, security_analyst |

#### **Data Ingestion**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `data:ingest_hec` | Send data via HEC endpoint | admin, ingester |
| `data:ingest_api` | Send data via API | admin, ingester |
| `data:delete` | Delete indexed data | admin |
| `data:modify` | Modify event data | admin |

#### **Indexes**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `indexes:list` | List available indexes | admin, security_analyst, analyst |
| `indexes:create` | Create new indexes | admin |
| `indexes:configure` | Modify index settings | admin |
| `indexes:delete` | Delete indexes | admin |
| `indexes:manage_retention` | Configure retention policies | admin |

#### **System Configuration**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `system:configure` | Modify system settings | admin |
| `system:restart` | Restart services | admin |
| `system:backup` | Trigger backups | admin |
| `system:restore` | Restore from backup | admin |
| `system:view_health` | View system health | admin, security_analyst |
| `system:view_metrics` | View performance metrics | admin |

#### **Role & Permission Management**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `roles:list` | List roles | admin |
| `roles:create` | Create custom roles | admin |
| `roles:update` | Modify role capabilities | admin |
| `roles:delete` | Delete custom roles | admin |
| `roles:assign` | Assign roles to users | admin |

#### **HEC Token Management**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `hec:list_own` | List own HEC tokens | admin, ingester |
| `hec:list_all` | List all HEC tokens | admin |
| `hec:create` | Create HEC tokens | admin, ingester |
| `hec:revoke_own` | Revoke own tokens | admin, ingester |
| `hec:revoke_all` | Revoke any token | admin |

#### **Audit & Compliance**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `audit:view` | View audit logs | admin, compliance_auditor |
| `audit:export` | Export audit logs | admin, compliance_auditor |
| `audit:configure` | Configure audit settings | admin |
| `compliance:run_reports` | Generate compliance reports | admin, compliance_auditor |

#### **Investigation & Cases**

| Capability | Description | Default Roles |
|------------|-------------|---------------|
| `cases:create` | Create investigation cases | admin, security_analyst |
| `cases:edit_own` | Edit own cases | admin, security_analyst |
| `cases:edit_all` | Edit any case | admin |
| `cases:close` | Close cases | admin, security_analyst |
| `cases:view_all` | View all cases | admin, security_analyst |

---

## Tier 3: Resource-Level Access Control

### Resource ACL Model

Every resource (dashboard, saved search, alert, case) has an ACL:

```json
{
  "resource_id": "dash-a1b2c3",
  "resource_type": "dashboard",
  "owner_id": "user-xyz",
  "acl": {
    "readers": ["role:security_analyst", "user:alice@company.com"],
    "writers": ["user:alice@company.com"],
    "public": false,
    "organization_wide": true,
    "inherits_from": null
  },
  "created_at": "2025-11-06T12:00:00Z",
  "updated_at": "2025-11-06T12:00:00Z"
}
```

### ACL Permission Levels

1. **Owner** - Full control, can delete and change permissions
2. **Writer** - Can modify the resource
3. **Reader** - Can view the resource
4. **None** - No access (explicit or implicit)

### Sharing Models

#### Private (Default)
- Only owner can access
- Not visible to others

#### Organization-Wide
- All users with appropriate capabilities can view
- Only owner (or those with `edit_all` capability) can edit

#### Role-Based Sharing
- Shared with specific roles
- Example: Share dashboard with `security_analyst` role

#### User-Based Sharing
- Shared with specific users
- Example: Share alert with specific team members

#### Public
- Visible to all authenticated users
- Read-only unless user has edit capabilities

---

## Index-Level Data Access Control

### Index Access Rules

Users can only search indexes they have explicit access to:

```json
{
  "user_id": "user-123",
  "index_access": {
    "allowed_patterns": [
      "app-logs-*",
      "web-logs-prod",
      "security-events-*"
    ],
    "denied_patterns": [
      "*-sensitive",
      "finance-*"
    ]
  },
  "default_index": "app-logs-prod",
  "enforced_filters": {
    "tenant_id": "tenant-abc"
  }
}
```

### Pattern Matching

- Wildcard support: `logs-*`, `*-prod`, `app-*-logs`
- Explicit deny overrides allow
- Empty allow list = deny all

### Enforced Search Filters

Automatically injected into queries based on role/user:

```
Original Query: error status:500
Enforced Filter: tenant_id="customer-123" AND region="us-west"
Final Query: (error status:500) AND (tenant_id="customer-123" AND region="us-west")
```

**Use Cases:**
- Multi-tenant isolation
- Geographic restrictions
- Department/team isolation
- Compliance boundaries

---

## Permission Inheritance and Composition

### Role Inheritance

Roles can inherit capabilities from parent roles:

```
admin
  ↓ inherits
security_analyst
  ↓ inherits  
analyst
```

### Additive Model

User's final permissions = Union of all assigned roles

```
User assigned: [analyst, compliance_auditor]
Final capabilities: analyst.capabilities ∪ compliance_auditor.capabilities
```

### Explicit Deny

Deny rules cannot be overridden by allow rules:

```json
{
  "user_id": "user-123",
  "roles": ["security_analyst"],
  "explicit_denies": ["data:delete"],
  "reason": "Temporary restriction during investigation"
}
```

---

## Permission Evaluation Algorithm

```
1. Is user authenticated? → NO: DENY
2. Is user enabled? → NO: DENY
3. Does user have required capability? → NO: DENY
4. Check explicit denies → FOUND: DENY
5. Is resource access required?
   YES:
     5a. Does user own resource? → YES: ALLOW
     5b. Is resource public? → YES: ALLOW (if capability permits)
     5c. Is user in resource ACL? → YES: ALLOW
     5d. Does user role match ACL? → YES: ALLOW
     5e. DENY
   NO: ALLOW
6. Check index access (for search operations)
7. Apply enforced filters
8. ALLOW
```

### Caching Strategy

- User capabilities: Cache 5 minutes
- Resource ACLs: Cache 2 minutes
- Index access: Cache 5 minutes
- Invalidate on role/permission changes

---

## API Protection

### Middleware Stack

```
Request
  ↓
Authentication Middleware (validate JWT)
  ↓
Authorization Middleware (check capability)
  ↓
Resource ACL Middleware (check resource access)
  ↓
Rate Limiting Middleware (per user/role)
  ↓
Audit Logging Middleware
  ↓
Handler
```

### Example API Endpoints

```
POST   /api/v1/users          → requires: users:create
GET    /api/v1/users          → requires: users:list
PUT    /api/v1/users/:id      → requires: users:update
DELETE /api/v1/users/:id      → requires: users:delete

POST   /api/v1/search         → requires: search:execute
POST   /api/v1/search/export  → requires: search:export

GET    /api/v1/dashboards/:id → requires: dashboards:view_shared + ACL check
PUT    /api/v1/dashboards/:id → requires: dashboards:edit_own|edit_all + ACL check
DELETE /api/v1/dashboards/:id → requires: dashboards:delete_own|delete_all + ACL check
```

---

## UI Permission Integration

### Feature Visibility

UI elements are conditionally rendered based on capabilities:

```jsx
{hasCapability('users:create') && (
  <Button onClick={createUser}>Create User</Button>
)}

{hasCapability('dashboards:edit_own') && isOwner(dashboard) && (
  <Button onClick={editDashboard}>Edit</Button>
)}

{hasCapability('alerts:delete_all') || (hasCapability('alerts:delete_own') && isOwner(alert)) && (
  <Button onClick={deleteAlert}>Delete</Button>
)}
```

### Navigation Guards

```jsx
<Route path="/admin/users" 
  element={
    <RequireCapability capability="users:list">
      <UsersPage />
    </RequireCapability>
  } 
/>
```

### Disabled States

Features without permissions shown but disabled with tooltip:

```
[Save Dashboard] (disabled)
Tooltip: "You need 'dashboards:create' permission"
```

---

## Audit Logging

### Permission Check Events

Every permission check is logged:

```json
{
  "event_type": "permission_check",
  "timestamp": "2025-11-06T12:34:56Z",
  "user_id": "user-123",
  "user_name": "alice@company.com",
  "capability_checked": "users:delete",
  "result": "allowed",
  "resource_type": "user",
  "resource_id": "user-456",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "request_id": "req-abc123"
}
```

### Permission Changes

```json
{
  "event_type": "permission_change",
  "timestamp": "2025-11-06T12:34:56Z",
  "actor_id": "admin-001",
  "actor_name": "admin@company.com",
  "change_type": "role_assignment",
  "target_user_id": "user-123",
  "changes": {
    "roles_added": ["security_analyst"],
    "roles_removed": ["analyst"]
  },
  "reason": "Promoted to SOC team"
}
```

---

## Database Schema

### Roles Table

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    capabilities JSONB NOT NULL,
    is_system_role BOOLEAN DEFAULT false,
    inherits_from UUID[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### User Roles Table

```sql
CREATE TABLE user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
```

### Resource ACLs Table

```sql
CREATE TABLE resource_acls (
    id UUID PRIMARY KEY,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    owner_id UUID REFERENCES users(id) ON DELETE CASCADE,
    acl JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(resource_type, resource_id)
);

CREATE INDEX idx_resource_acls_owner ON resource_acls(owner_id);
CREATE INDEX idx_resource_acls_type ON resource_acls(resource_type);
```

### User Index Access Table

```sql
CREATE TABLE user_index_access (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    allowed_patterns TEXT[] NOT NULL,
    denied_patterns TEXT[] DEFAULT '{}',
    enforced_filters JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id)
);
```

---

## Migration Path

### RC1 (Current State → Capability Model)

**Goals:**
- Add capabilities to existing roles
- Implement capability checking middleware
- Add capability-based UI guards

**Changes:**
- Database: Add `capabilities` column to roles, populate defaults
- Backend: Add capability checking in handlers
- Frontend: Add capability context and guards
- No breaking changes to existing functionality

**Timeline:** 2-3 weeks

### RC2 (Resource ACLs)

**Goals:**
- Add ownership to saved objects
- Implement sharing functionality
- ACL-based access checks

**Changes:**
- Database: Add resource_acls table
- Backend: ACL checking middleware
- Frontend: Sharing UI, ownership indicators
- Migration: Set current user as owner of existing objects

**Timeline:** 3-4 weeks

### RC3 (Index Access Control)

**Goals:**
- Index-level data access
- Search filter enforcement

**Changes:**
- Database: Add user_index_access table
- Backend: Query rewriting for enforced filters
- Frontend: Index selector respects access
- Admin UI for managing index access

**Timeline:** 2-3 weeks

### RC4 (Advanced Features)

**Goals:**
- Role inheritance
- Temporary permissions
- IP restrictions
- Session management

**Timeline:** 4-6 weeks

---

## Security Considerations

### Defense Against Common Attacks

#### Privilege Escalation
- **Prevention**: All permission changes require `roles:assign` + audit logging
- **Detection**: Alert on unusual role assignments

#### Horizontal Privilege Escalation  
- **Prevention**: Resource ownership checks before modification
- **Detection**: Alert on access to unowned resources

#### Token Theft
- **Prevention**: Short-lived tokens, refresh token rotation
- **Mitigation**: Revoke sessions, require re-authentication

#### Brute Force Permission Discovery
- **Prevention**: Rate limiting on API endpoints
- **Response**: Generic "403 Forbidden" (don't leak capability names)

---

## Performance Targets

- Permission check latency: < 5ms (cached)
- Permission check latency: < 50ms (uncached)
- Cache hit rate: > 95%
- Database queries per check: ≤ 2 (with proper indexing)
- UI render blocking: 0 (async permission loading)

---

## Testing Strategy

### Unit Tests
- Capability evaluation logic
- ACL permission checking
- Role inheritance resolution
- Search filter injection

### Integration Tests
- End-to-end permission flows
- API endpoint protection
- UI feature gating
- Audit log generation

### Security Tests
- Privilege escalation attempts
- ACL bypass attempts
- Token tampering
- Permission cache invalidation

---

## Compliance Mapping

### SOC 2
- Access control policies → Role definitions
- Audit logging → Permission check events
- Least privilege → Capability-based permissions

### PCI DSS
- Requirement 7 (Access Control) → User roles and capabilities
- Requirement 10 (Logging) → Audit trail
- Requirement 8 (Identity Management) → User management

### HIPAA
- Access Controls → Role-based and resource-level ACLs
- Audit Controls → Complete audit logging
- Integrity Controls → Immutable audit logs

### GDPR
- Access Limitation → Enforced filters and index access
- Accountability → Audit trail with user attribution

---

## API Examples

### Check User Capabilities

```http
GET /api/v1/auth/capabilities
Authorization: Bearer <token>

Response:
{
  "user_id": "user-123",
  "capabilities": [
    "search:execute",
    "dashboards:create",
    "alerts:view"
  ],
  "roles": ["security_analyst"]
}
```

### Create Role with Capabilities

```http
POST /api/v1/roles
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "threat_hunter",
  "description": "Advanced threat hunting team",
  "capabilities": [
    "search:execute",
    "search:export",
    "search:realtime",
    "alerts:create",
    "cases:create",
    "data:delete"
  ],
  "inherits_from": ["security_analyst"]
}
```

### Share Dashboard

```http
POST /api/v1/dashboards/:id/share
Authorization: Bearer <token>
Content-Type: application/json

{
  "share_with": {
    "roles": ["security_analyst"],
    "users": ["alice@company.com", "bob@company.com"]
  },
  "permission": "reader"
}
```

### Set Index Access

```http
PUT /api/v1/users/:id/index-access
Authorization: Bearer <token>
Content-Type: application/json

{
  "allowed_patterns": ["security-*", "firewall-*"],
  "denied_patterns": ["*-archive"],
  "enforced_filters": {
    "tenant_id": "tenant-abc"
  }
}
```

---

## CLI Support

```bash
# List user capabilities
thawk auth whoami --show-capabilities

# Create role
thawk role create threat_hunter \
  --capabilities search:execute,search:export,alerts:create \
  --inherits-from security_analyst

# Assign role
thawk user update user-123 --add-role threat_hunter

# Share dashboard
thawk dashboard share dash-abc \
  --with-role security_analyst \
  --permission reader

# Set index access
thawk user set-index-access user-123 \
  --allow "security-*,firewall-*" \
  --deny "*-sensitive"
```

---

## Conclusion

This privilege system provides enterprise-grade access control with:

✅ **Granular control** - 50+ capabilities covering all operations  
✅ **Flexible sharing** - Multiple sharing models for collaboration  
✅ **Data isolation** - Index-level access and enforced filters  
✅ **Full auditability** - Every permission check logged  
✅ **Performance** - Aggressive caching, < 5ms checks  
✅ **Compliance-ready** - Maps to SOC2, PCI DSS, HIPAA, GDPR  

The phased approach allows incremental implementation without disrupting existing functionality.
