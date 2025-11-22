# Role-Based Access Control (RBAC) & Multi-Tenancy

This document describes TelHawk's permission system, designed for MSSP/MDR multi-tenant deployments.

## Overview

TelHawk uses a tiered, multi-tenant RBAC model with:

- **Three tenant tiers**: Platform → Organization → Client
- **Role ordinals**: Power levels within each tier (lower = more powerful)
- **Resource:action permissions**: Fine-grained permission gates
- **Scoped access**: Higher-tier users can be limited to specific lower-tier entities

## Tenant Hierarchy

```
Platform (singleton - the TelHawk operator)
│
├── Organization A (customer, e.g., "Acme Corp")
│   ├── Client A1 (sub-tenant, e.g., "Acme West")
│   └── Client A2 (sub-tenant, e.g., "Acme East")
│
└── Organization B (another customer)
    ├── Client B1
    └── Client B2
```

### Data Isolation

| User Tier | Data Visibility |
|-----------|-----------------|
| Platform  | All organizations and all clients |
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
| `tenants:create` | Create organizations/clients |
| `tenants:read` | View tenant information |
| `tenants:update` | Modify tenant settings |
| `tenants:delete` | Delete tenants |

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

> **Note**: This is the target schema design. Implementation may vary.

### Tenants

```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    type VARCHAR(20) NOT NULL,  -- 'platform', 'organization', 'client'
    parent_id UUID REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    disabled_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT valid_type CHECK (type IN ('platform', 'organization', 'client')),
    CONSTRAINT platform_no_parent CHECK (
        (type = 'platform' AND parent_id IS NULL) OR
        (type != 'platform' AND parent_id IS NOT NULL)
    )
);

-- Ensure only one platform tenant exists
CREATE UNIQUE INDEX one_platform ON tenants ((true)) WHERE type = 'platform';
```

### Roles

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    tenant_type VARCHAR(20) NOT NULL,  -- which tier this role belongs to
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    ordinal SMALLINT NOT NULL DEFAULT 50,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,  -- true = cannot delete
    is_protected BOOLEAN DEFAULT FALSE,  -- true = ordinal 0, immutable
    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT valid_ordinal CHECK (ordinal >= 0 AND ordinal <= 99),
    CONSTRAINT protected_slug CHECK (
        (is_protected = TRUE AND slug IN ('root', 'admin')) OR
        (is_protected = FALSE AND slug NOT IN ('root', 'admin'))
    ),
    UNIQUE (tenant_type, slug)
);
```

### Permissions

```sql
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
    is_locked BOOLEAN DEFAULT FALSE,  -- locked for protected roles

    PRIMARY KEY (role_id, permission_id)
);
```

### User Roles

```sql
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,

    -- Scope restrictions (NULL = unrestricted within tenant)
    scope_organization_ids UUID[],  -- for platform users
    scope_client_ids UUID[],        -- for org users

    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,

    UNIQUE (user_id, role_id, tenant_id)
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

### Tenant-Scoped Operations

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
    if !u.IsInTierTreeOf(target) {
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

1. **Platform tenant** (singleton)
2. **root role** (ordinal 0, protected, all permissions)
3. **admin user** with root role

### New Organization

When an organization is created:

1. Org Owner role assigned to creating user (or specified owner)
2. Default Org Admin and Org Analyst roles available

### New Client

When a client is created:

1. Client Owner role assigned to creating user (or specified owner)
2. Default Client Admin and Client Analyst roles available

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
| **Tier hierarchy** | Platform → Organization → Client |
| **Ordinal** | Lower number = more powerful (0-99) |
| **Protected** | `root`/`admin` at ordinal 0, immutable |
| **Management** | Can manage ≥ ordinal in same tier or below |
| **Scope** | Higher-tier users can be limited to specific lower entities |
| **Visibility** | Scoped external users visible but not editable |
