# Role-Based Access Control (RBAC) & Multi-Organization Access

This document describes TelHawk's permission system, designed for MSSP/MDR deployments with multiple organizations and clients.

## Overview

TelHawk uses a tiered RBAC model with:

- **Three scope tiers**: Platform → Organization → Client
- **Role ordinals**: Power levels within each tier (lower = more powerful)
- **Resource:action permissions**: Fine-grained permission gates
- **Scoped access**: Higher-tier users can be limited to specific lower-tier entities

## Scope Hierarchy

```
Platform (singleton - the TelHawk operator)
│
├── Organization A (customer, e.g., "Acme Corp")
│   ├── Client A1 (e.g., "Acme West")
│   └── Client A2 (e.g., "Acme East")
│
└── Organization B (another customer)
    ├── Client B1
    └── Client B2
```

### Scope Determination

Scope is determined by which IDs are populated:

| Condition | Scope |
|-----------|-------|
| `client_id IS NOT NULL` | Client-scoped |
| `organization_id IS NOT NULL AND client_id IS NULL` | Organization-scoped |
| Both NULL | Platform-scoped |

### Data Isolation

| User Scope | Data Visibility |
|------------|-----------------|
| Platform | All organizations and all clients |
| Organization | Their organization + all their clients |
| Client | Their client only |

> **Strict isolation**: Organizations cannot see other organizations. Clients cannot see other clients or their parent organization's data (unless explicitly shared).

## Role Tiers & Ordinals

Each tier has roles with **ordinal values** (0-99). Lower ordinal = more powerful.

### Platform Tier

| Role | Ordinal | Description |
|------|---------|-------------|
| `root` / `admin` | 0 | **Protected** - Immutable system superadmin |
| Platform Owner | 10 | Full platform control, can manage all |
| Platform Admin | 20 | Platform administration, can manage ≤20 |
| Platform Analyst | 30 | Data visibility across platform |

### Organization Tier

| Role | Ordinal | Description |
|------|---------|-------------|
| Org Owner | 10 | Full organization + client control |
| Org Admin | 20 | Organization administration |
| Org Analyst | 30 | Data visibility for organization |

### Client Tier

| Role | Ordinal | Description |
|------|---------|-------------|
| Client Owner | 10 | Full client control |
| Client Admin | 20 | Client administration |
| Client Analyst | 30 | Data visibility for client |

### Ordinal Gaps

Ordinals 10, 20, 30 are the defaults. Gaps (11-19, 21-29, 31-99) are reserved for future custom roles, allowing fine-grained hierarchy without schema changes.

## Protected Roles

### Reserved Names

The following role names are **protected** and cannot be created, edited, or assigned except by the system:

- `root`
- `admin`

### Ordinal 0 Protection

Roles with ordinal 0 are **immutable**:

- Cannot be modified by any user
- Cannot be deleted
- Permissions cannot be changed
- Only the system can assign ordinal 0 roles

## Permission Gates

Permissions follow the `resource:action` format.

### Permission Matrix

#### Users & Authentication

| Permission | Description |
|------------|-------------|
| `users:create` | Create new users |
| `users:read` | View user profiles |
| `users:update` | Modify user details |
| `users:delete` | Delete/disable users |
| `users:assign_roles` | Assign roles to users |
| `users:reset_password` | Reset user passwords |
| `tokens:create` | Create HEC/API tokens |
| `tokens:read` | View token list |
| `tokens:revoke` | Revoke tokens |
| `tokens:manage_all` | Manage any user's tokens |

#### Detection & Response

| Permission | Description |
|------------|-------------|
| `rules:create` | Create detection rules |
| `rules:read` | View detection rules |
| `rules:update` | Modify detection rules |
| `rules:delete` | Delete detection rules |
| `rules:enable` | Enable detection rules |
| `rules:disable` | Disable detection rules |
| `alerts:read` | View alerts |
| `alerts:acknowledge` | Acknowledge alerts |
| `alerts:close` | Close alerts |
| `alerts:assign` | Assign alerts to users |
| `alerts:delete` | Delete alerts |
| `cases:create` | Create cases |
| `cases:read` | View cases |
| `cases:update` | Modify cases |
| `cases:close` | Close cases |
| `cases:delete` | Delete cases |
| `cases:assign` | Assign cases to users |

#### Search & Data

| Permission | Description |
|------------|-------------|
| `search:execute` | Run searches |
| `search:export` | Export search results |
| `search:save_queries` | Save search queries |
| `events:read` | View events |

#### System Administration

| Permission | Description |
|------------|-------------|
| `system:configure` | Modify system settings |
| `system:view_audit` | View audit logs |
| `system:manage_integrations` | Configure integrations |
| `organizations:create` | Create organizations |
| `organizations:read` | View organization information |
| `organizations:update` | Modify organization settings |
| `organizations:delete` | Delete organizations |
| `clients:create` | Create clients |
| `clients:read` | View client information |
| `clients:update` | Modify client settings |
| `clients:delete` | Delete clients |

### Usage Pattern

```go
// In handlers
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
    user := auth.UserFromContext(r.Context())

    if !user.Can("users:delete") {
        httputil.Forbidden(w, "users:delete permission required")
        return
    }

    // ... proceed with deletion
}
```

## User Management Rules

### Ordinal Rule

> You can only manage users with ordinal **≥** your own (same or higher number = same or less powerful)

| Actor (Ordinal) | Can Manage |
|-----------------|------------|
| Platform Owner (10) | ≥10 (Owners, Admins, Analysts) |
| Platform Admin (20) | ≥20 (Admins, Analysts) |
| Platform Analyst (30) | ≥30 (other Analysts only) |

### Tier Rule

> You can only manage users in your tier or **below** in your tree

| Actor | Can Manage Users In |
|-------|---------------------|
| Platform Admin | Platform + any Org + any Client |
| Org Admin (Acme) | Acme Org + Acme's Clients |
| Client Admin (Acme West) | Acme West only |

### Combined Example

**Platform Admin (ordinal 20)** can:
- ✅ Create Platform Admin (ordinal 20)
- ✅ Create Platform Analyst (ordinal 30)
- ✅ Manage any Org Owner (ordinal 10, but lower tier)
- ❌ Modify Platform Owner (ordinal 10, same tier, more powerful)
- ❌ Modify `root` (ordinal 0, protected)

**Org Admin at Acme (ordinal 20)** can:
- ✅ Create Org Admin at Acme (ordinal 20)
- ✅ Manage Client Owner at Acme West (ordinal 10, lower tier)
- ❌ View users at "Other Corp" (different org tree)
- ❌ Modify Platform Analyst scoped to Acme (higher tier)

### Password Reset Protection

Password reset follows the same rules:

```
Platform Admin (20) → Can reset passwords for:
  ✅ Platform Admin (≥20, same tier)
  ✅ Platform Analyst (≥20, same tier)
  ✅ Any Org/Client user (lower tier)
  ❌ Platform Owner (ordinal 10 < 20)
  ❌ root/admin (protected)
```

## Scoped Access

Higher-tier users can be **scoped** to specific lower-tier entities, limiting their data visibility while maintaining their tier privileges.

### Scope Examples

```yaml
# Platform Analyst scoped to one organization
user: jane@telhawk.io
tier: platform
role: analyst (ordinal 30)
scope:
  - organization_id: "acme-corp"
# Jane sees: Acme Corp + all Acme clients
# Jane cannot see: Other organizations

# Org Analyst scoped to one client
user: bob@acme.com
tier: organization (Acme Corp)
role: analyst (ordinal 30)
scope:
  - client_id: "acme-west"
# Bob sees: Acme West only
# Bob cannot see: Acme East, other clients
```

### Unscoped Access

```yaml
# Platform Analyst with no scope = full platform visibility
user: alice@telhawk.io
tier: platform
role: analyst (ordinal 30)
scope: []  # empty = all
# Alice sees: ALL organizations and clients
```

## User Visibility

Users appear in two lists based on context:

### Managed Users

Users you **can edit** based on ordinal and tier rules.

- Appears in your main user management interface
- Full CRUD operations available (based on permissions)
- Same tier or below in your tree

### Shared Users

Higher-tier users **scoped to your entity** (visible but not editable).

- Separate "Shared Users" section in UI
- Read-only view
- Shows their role and scope
- Cannot modify, disable, or manage

### Example: Acme Corp Admin View

```
┌─────────────────────────────────────────────────────────┐
│ User Management - Acme Corp                             │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ Managed Users                                           │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ john@acme.com    Org Admin     [Edit] [Disable]    │ │
│ │ mary@acme.com    Org Analyst   [Edit] [Disable]    │ │
│ │ tim@acme.com     Client Owner  [Edit] [Disable]    │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ Shared Users (Platform)                          [?]    │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ jane@telhawk.io  Platform Analyst  [View Profile]  │ │
│ │ ↳ Scoped to: Acme Corp                             │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Database Schema (Reference)

> **Note**: This reflects the current schema. See `authenticate/migrations/001_init.up.sql` for the full implementation.

### Organizations

```sql
CREATE TABLE organizations (
    id UUID NOT NULL,                    -- UUIDv7: timestamp = created_at
    version_id UUID PRIMARY KEY,         -- UUIDv7: timestamp = updated_at
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}',
    created_by UUID,
    updated_by UUID,
    disabled_at TIMESTAMPTZ,
    disabled_by UUID,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID
);
```

### Clients

```sql
CREATE TABLE clients (
    id UUID NOT NULL,                    -- UUIDv7: timestamp = created_at
    version_id UUID PRIMARY KEY,         -- UUIDv7: timestamp = updated_at
    organization_id UUID NOT NULL,       -- References organizations(id)
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}',
    created_by UUID,
    updated_by UUID,
    disabled_at TIMESTAMPTZ,
    disabled_by UUID,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID
);
```

### Roles

```sql
CREATE TABLE roles (
    id UUID NOT NULL,
    version_id UUID PRIMARY KEY,
    organization_id UUID,                -- NULL = platform/template role
    client_id UUID,                      -- NULL = org/platform role
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    ordinal SMALLINT NOT NULL DEFAULT 50,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    is_template BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID,

    CONSTRAINT valid_ordinal CHECK (ordinal >= 0 AND ordinal <= 99),
    CONSTRAINT client_requires_org CHECK (
        (client_id IS NOT NULL AND organization_id IS NOT NULL) OR
        (client_id IS NULL)
    )
);
```

### User Roles

```sql
CREATE TABLE user_roles (
    id UUID PRIMARY KEY,                 -- UUIDv7: timestamp = granted_at
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    organization_id UUID,                -- NULL = platform-level
    client_id UUID,                      -- NULL = org/platform-level
    scope_organization_ids UUID[],       -- For platform users: limit to these orgs
    scope_client_ids UUID[],             -- For org users: limit to these clients
    granted_by UUID,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID,

    CONSTRAINT client_requires_org CHECK (
        (client_id IS NOT NULL AND organization_id IS NOT NULL) OR
        (client_id IS NULL)
    )
);
```

## API Permission Checks

### Middleware Pattern

```go
// RequirePermission middleware
func RequirePermission(permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := auth.UserFromContext(r.Context())
            if user == nil {
                httputil.Unauthorized(w, "authentication required")
                return
            }

            if !user.Can(permission) {
                httputil.Forbidden(w, fmt.Sprintf("%s permission required", permission))
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### User.Can() Implementation

```go
func (u *User) Can(permission string) bool {
    // Superadmin (ordinal 0) has all permissions
    if u.HasProtectedRole() {
        return true
    }

    // Check if any of user's roles grant this permission
    for _, role := range u.Roles {
        if role.HasPermission(permission) {
            return true
        }
    }

    return false
}

func (u *User) HasProtectedRole() bool {
    for _, role := range u.Roles {
        if role.IsProtected && role.Ordinal == 0 {
            return true
        }
    }
    return false
}
```

### Scoped Operations

```go
func (u *User) CanManageUser(target *User) bool {
    // Cannot manage protected users
    if target.HasProtectedRole() {
        return false
    }

    // Must have users:update permission
    if !u.Can("users:update") {
        return false
    }

    // Check tier relationship
    if !u.IsInScopeTreeOf(target) {
        return false
    }

    // Check ordinal (can manage same or higher ordinal only)
    if u.HighestOrdinal() > target.LowestOrdinal() {
        return false
    }

    return true
}
```

## Default Role Assignments

### Initial Platform Setup

On first deployment, the system creates:

1. **Default organization** and **default client**
2. **root role** (ordinal 0, protected, all permissions)
3. **admin user** with root role at platform level

### New Organization

When an organization is created:

1. Org Owner role assigned to creating user (or specified owner)
2. Default Org Admin and Org Analyst roles copied from templates

### New Client

When a client is created:

1. Client Owner role assigned to creating user (or specified owner)
2. Default Client Admin and Client Analyst roles copied from templates

## Security Considerations

### Privilege Escalation Prevention

- Users cannot grant roles with ordinal < their own
- Users cannot modify roles in higher tiers
- Protected roles (ordinal 0) are immutable

### Audit Trail

All permission-related actions should be logged:

- Role assignments/revocations
- Permission changes
- Scope modifications
- Failed authorization attempts

### Session Invalidation

When a user's roles change:

- Active sessions should be invalidated
- User must re-authenticate to get new permissions
- Consider grace period for active operations

---

## Summary

| Concept | Rule |
|---------|------|
| **Scope hierarchy** | Platform → Organization → Client |
| **Scope determination** | `client_id` NOT NULL → client, `organization_id` NOT NULL → org, both NULL → platform |
| **Ordinal** | Lower number = more powerful (0-99) |
| **Protected** | `root`/`admin` at ordinal 0, immutable |
| **Management** | Can manage ≥ ordinal in same tier or below |
| **Scope restrictions** | Higher-tier users can be limited to specific lower entities |
| **Visibility** | Scoped external users visible but not editable |
