# TelHawk UI/UX Design Philosophy

**Last Updated:** 2025-11-08
**Status:** Living Document

## Core Principle: Information Density Meets Usability

Most SIEM tools suffer from fundamentally broken user interfaces. They display minimal useful information while wasting vast amounts of screen real estate on empty space, bloated navigation bars, and nested menus. Security engineering teams are forced to work with tools that seem designed to hide information rather than surface it.

**We believe security analysts deserve better.**

TelHawk is built on the principle that **every pixel should serve a purpose**. Our interface prioritizes information density without sacrificing readability or usability. We design for the analyst who needs to triage 50 alerts before lunch, not the executive who wants pretty dashboards.

---

## Design Philosophy: The Card-Based Event View

### The Problem with Traditional SIEM Tables

Traditional SIEM interfaces use rigid table layouts with fixed columns:

```
┌────────────┬──────────┬─────────┬─────────┬─────────┬──────────┐
│ Time       │ Severity │ Source  │ Dest    │ User    │ Actions  │
├────────────┼──────────┼─────────┼─────────┼─────────┼──────────┤
│ 10:23 AM   │ High     │ N/A     │ N/A     │ jsmith  │ View     │
│ 10:24 AM   │ Medium   │ 10.0... │ 192.... │ N/A     │ View     │
│ 10:25 AM   │ Low      │ N/A     │ N/A     │ N/A     │ View     │
└────────────┴──────────┴─────────┴─────────┴─────────┴──────────┘
```

**Problems:**
1. **Fixed columns force "N/A" everywhere** - A generic table tries to show all possible fields, resulting in half-empty columns. Authentication events need user/IP fields, network events need src/dst/port fields, file events need path/operation fields - no single table layout works for all types.
2. **Horizontal scroll hell** - Adding enough columns to cover all event types creates unusable interfaces
3. **Information is hidden** - Most useful context requires clicking "View Details" on every single event
4. **Context collapse** - You can't see what type of event you're looking at without reading multiple columns

### TelHawk's Solution: Type-Aware Card Layout

Each event is rendered as a **self-contained card** that displays only the fields relevant to its type:

```
┌─────────────────────────────────────────────────────────────────┐
│ 🔐 Authentication                           High    10:23:15 AM │
├─────────────────────────────────────────────────────────────────┤
│ Username: jsmith              Source IP: 192.168.1.45           │
│ Status: FAILED                Auth Protocol: LDAP               │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 🌐 Network Activity                      Medium    10:24:33 AM  │
├─────────────────────────────────────────────────────────────────┤
│ Source: 10.0.1.5:54321        Destination: 192.168.50.10:443   │
│ Protocol: TCP                 Direction: outbound               │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 📁 File Activity                            Low    10:25:12 AM  │
├─────────────────────────────────────────────────────────────────┤
│ File Path: /etc/passwd        Operation: Read                   │
│ User: root                    Size: 2,341 bytes                 │
└─────────────────────────────────────────────────────────────────┘
```

**Benefits:**
1. **Zero wasted space** - Every field shown is relevant to that specific event type
2. **Immediate context** - Icon and event type name make scanning trivial
3. **No horizontal scroll** - Cards stack vertically, fitting any screen width
4. **Information at a glance** - Most important details visible without clicking
5. **Type-specific intelligence** - Authentication events show auth fields, network events show network fields, etc.

### Supported Event Types (OCSF 1.1.0)

| Event Class | Icon | Key Fields Displayed |
|-------------|------|---------------------|
| Authentication (3002) | 🔐 | Username, Source IP, Status, Auth Protocol |
| Network Activity (4001) | 🌐 | Source IP:Port, Dest IP:Port, Protocol, Direction |
| Process Activity (1007) | ⚙️ | Process Name, PID, Command Line, User |
| File Activity (4006) | 📁 | File Path, Operation, User, Size |
| DNS Activity (4003) | 🔍 | Query Hostname, Record Type, Answer, Source IP |
| HTTP Activity (4002) | 🌐 | Method, URL, Status Code, Client IP |
| Detection Finding (2004) | 🚨 | Finding Title, MITRE Tactic, Technique, Risk Score |
| Unknown/Raw Events | ❓ | Source Type, Source, Host, Message (Splunk-like) |

### Fallback Handling: The Splunk Approach

For unknown event types or raw events, we fall back to a **Splunk-inspired view** showing universal fields:
- **Source Type** (`sourcetype`) - Where the data came from
- **Source** (`source`) - Specific data source identifier
- **Host** - Hostname or IP of the originating system
- **Message** - Raw event message or description

This ensures that **every event is renderable**, even if we don't have a custom view for it yet.

---

## The Filter Bar: Progressive Disclosure of Complexity

### The Problem: No One Has Good Filters

Look at the major SIEM platforms:
- **Splunk**: Requires writing SPL queries. Powerful but hostile to new users.
- **Elastic**: Forces you into KQL or DSL. Same problem.
- **Sentinel**: KQL again. Notice a pattern?
- **Chronicle**: UI filters are hidden in dropdowns that require 5 clicks.

**Every major SIEM treats filtering as an afterthought.**

The common pattern: Either write code (query language) or suffer through nested dropdown menus that hide what you're filtering on. Both approaches fail:
- **Query languages** are powerful but require training and slow down triage
- **Dropdown menus** hide context and make it impossible to see your active filters at a glance

### TelHawk's Filter Bar: Progressive Disclosure

Our filter bar follows the principle of **progressive disclosure**: Start simple, reveal complexity only when needed.

**Critical Design Constraint:** The filter bar is designed to filter to **ONE event type**. Type-specific filters only appear after selecting a single event class, because filtering across multiple event types with type-specific fields is not useful (different events have different fields).

Users who need cross-type queries can use the query language escape hatch.

#### Level 1: Event Class Filter (Default State)

```
┌─────────────────────────────────────────────────────────────────┐
│ Search Events                                                    │
├─────────────────────────────────────────────────────────────────┤
│ [ Filter by Event Class ▼ ]                                     │
└─────────────────────────────────────────────────────────────────┘
```

**Single button:** "Filter by Event Class"
- Clicking opens a **searchable dropdown** with all OCSF event classes
- Type to filter: "auth" → shows Authentication, "net" → shows Network Activity
- Visual: Shows event type icon and name

#### Level 2: Type-Specific Filters (After Selection)

Once an event class is selected, the filter bar **expands** to show common fields for that type:

**Example: Authentication Events**
```
┌─────────────────────────────────────────────────────────────────┐
│ Active Filters:                                                  │
├─────────────────────────────────────────────────────────────────┤
│ [🔐 Authentication ×]                                            │
│                                                                  │
│ Add Filter:                                                      │
│ [ Username ▼ ] [ Source IP ▼ ] [ Status ▼ ] [ + More ]          │
└─────────────────────────────────────────────────────────────────┘
```

**Example: Network Activity**
```
┌─────────────────────────────────────────────────────────────────┐
│ Active Filters:                                                  │
├─────────────────────────────────────────────────────────────────┤
│ [🌐 Network Activity ×]                                          │
│                                                                  │
│ Add Filter:                                                      │
│ [ Source IP ▼ ] [ Dest IP ▼ ] [ Protocol ▼ ] [ Port ▼ ] [ + More ] │
└─────────────────────────────────────────────────────────────────┘
```

**Example: Detection Findings**
```
┌─────────────────────────────────────────────────────────────────┐
│ Active Filters:                                                  │
├─────────────────────────────────────────────────────────────────┤
│ [🚨 Detection Finding ×]                                         │
│                                                                  │
│ Add Filter:                                                      │
│ [ Tactic ▼ ] [ Technique ▼ ] [ Severity ▼ ] [ Risk Score ▼ ] [ + More ] │
└─────────────────────────────────────────────────────────────────┘
```

#### Level 3: Active Filters (Chip-Based Display)

As filters are applied, they appear as **removable chips**:

```
┌─────────────────────────────────────────────────────────────────┐
│ Active Filters:                                                  │
├─────────────────────────────────────────────────────────────────┤
│ [🔐 Authentication ×] [Status: Failed ×] [User: jsmith ×]        │
│                                                                  │
│ Add Filter:                                                      │
│ [ Source IP ▼ ] [ Auth Protocol ▼ ] [ + More ]                  │
└─────────────────────────────────────────────────────────────────┘
```

**Design Principles:**
1. **Visual clarity**: Each filter is a chip that can be removed with one click
2. **No hidden filters**: Everything you're filtering on is visible at a glance
3. **Contextual options**: Filter buttons change based on event class
4. **Smart defaults**: Show 3-5 most common filters, hide advanced ones under "+ More"

### Type-Specific Filter Sets

Each event class has a curated set of common filters based on analyst workflows:

| Event Class | Primary Filters | Secondary Filters (+ More) |
|-------------|----------------|---------------------------|
| Authentication | Username, Source IP, Status, Auth Protocol | Destination Host, Session ID, MFA Status |
| Network | Source IP, Dest IP, Protocol, Port | Direction, Boundary, Bytes Transferred |
| Process | Process Name, User, Parent Process | Command Line, PID, Executable Hash |
| File | File Path, Operation, User | File Size, Hash (MD5/SHA256), Modified Time |
| DNS | Query Hostname, Record Type, Response Code | DNS Server, Answer Count, TTL |
| HTTP | Method, URL Path, Status Code, Client IP | User Agent, Content Type, Response Size |
| Detection | Tactic, Technique, Severity, Risk Score | Analytic Name, Confidence, Affected Resources |

### Implementation Notes

**Filter Dropdown Behavior:**
- Click filter button → Dropdown appears
- Dropdown shows:
  - **Text input** for free-form entry (IP addresses, usernames, etc.)
  - **Common values** pulled from recent events (last 1000)
  - **"Add Custom"** option for regex or advanced queries
- Select value → Appears as chip in "Active Filters"
- Multiple values for same field create OR logic (e.g., "Status: Failed OR Status: Locked")

**Query Translation:**
Behind the scenes, filter chips translate to OpenSearch queries:
```
[Authentication] [Status: Failed] [User: jsmith]
    ↓
class_uid:3002 AND status:"Failed" AND (user.name:"jsmith" OR actor.user.name:"jsmith")
```

**Power User Escape Hatch:**
Advanced users can still write raw queries. The filter bar and query box work together:
- **Filters → Auto-update query box** (implemented) - Filter chips automatically generate the OpenSearch query
- **Manual query editing** (available) - Users can click "Show Advanced Query" to write queries directly
- **Query → Filter parsing** (future enhancement) - Attempt to parse simple queries back into filter chips when possible
- **Complex queries** - When using advanced query syntax that can't be represented as filters, filter chips are disabled with an info message

---

## View Modes: Card vs Table

### The Choice Problem

Different analysts have different preferences and workflows:
- **Threat hunters** often prefer cards for scanning diverse events quickly
- **Incident responders** may prefer tables for comparing specific fields across many events
- **Compliance analysts** often need tabular exports for reports

**TelHawk doesn't force a choice. We support both.**

### Card View (Default)

Card view is the default because it handles **mixed event types** gracefully:
- Authentication events show user fields
- Network events show IP/port fields
- Detection events show MITRE ATT&CK fields
- No "N/A" columns cluttering the interface

**When to use cards:**
- Scanning across multiple event types
- Quick triage of diverse security events
- When context matters more than comparison

### Table View (Type-Specific)

Table view works best when **filtered to a single event type**. When you filter to one OCSF class, the table shows **type-specific columns**:

**Important:** If no event class filter is active (showing mixed event types), the table view button is disabled with a tooltip: "Filter to a single event class to enable table view." This prevents the "N/A everywhere" problem that plagues traditional SIEMs.

**Authentication Events Table:**
```
┌──────────────┬──────────┬──────────────┬─────────────┬──────────────┐
│ Timestamp    │ Username │ Source IP    │ Status      │ Auth Method  │
├──────────────┼──────────┼──────────────┼─────────────┼──────────────┤
│ 10:23:15 AM  │ jsmith   │ 192.168.1.45 │ Failed      │ LDAP         │
│ 10:23:18 AM  │ jsmith   │ 192.168.1.45 │ Failed      │ LDAP         │
│ 10:23:22 AM  │ jsmith   │ 192.168.1.45 │ Success     │ LDAP         │
└──────────────┴──────────┴──────────────┴─────────────┴──────────────┘
```

**Network Events Table:**
```
┌──────────────┬─────────────────┬─────────────────┬──────────┬───────────┐
│ Timestamp    │ Source          │ Destination     │ Protocol │ Direction │
├──────────────┼─────────────────┼─────────────────┼──────────┼───────────┤
│ 10:24:33 AM  │ 10.0.1.5:54321  │ 192.168.50.10:443│ TCP     │ Outbound  │
│ 10:24:34 AM  │ 10.0.1.5:54322  │ 192.168.50.10:443│ TCP     │ Outbound  │
└──────────────┴─────────────────┴─────────────────┴──────────┴───────────┘
```

**Detection Findings Table:**
```
┌──────────────┬─────────────────────┬──────────────┬───────────┬──────────┐
│ Timestamp    │ Finding             │ Tactic       │ Technique │ Severity │
├──────────────┼─────────────────────┼──────────────┼───────────┼──────────┤
│ 10:25:12 AM  │ Lateral Movement    │ Lateral Mov. │ T1021.002 │ High     │
│ 10:26:45 AM  │ Credential Dumping  │ Cred Access  │ T1003.001 │ Critical │
└──────────────┴─────────────────────┴──────────────┴───────────┴──────────┘
```

**Table View Features (Planned):**
- **Column headers = filter buttons** - Click a column header to add a filter for that field
- **Inline filtering** - Type in column headers to filter values
- **Sortable columns** - Click to sort by any field
- **Export-friendly** - CSV/JSON export uses visible columns
- **Responsive** - Horizontal scroll on smaller screens (no compromises)

**When to use tables:**
- Filtered to a single event type
- Comparing specific fields across many events
- Sorting/filtering by specific attributes
- Exporting data for reports or further analysis

### View Mode Toggle (Planned)

A simple toggle at the top of the results area:
```
┌─────────────────────────────────────────────────────────────────┐
│ View Mode:  [📇 Cards]  [📊 Table]                              │
└─────────────────────────────────────────────────────────────────┘
```

**Intelligent defaults:**
- No event class filter → Cards (handles mixed types)
- Single event class selected → User's last preference (defaults to Cards)
- Preference saved per-user in browser localStorage

**Current Status:** Card view is implemented. Table view toggle planned for future release.

---

## Customization: CRM-Style Personalization

### The Problem with Rigid UIs

Most SIEM interfaces are one-size-fits-all:
- Fixed fields in cards
- Fixed columns in tables
- No way to adapt to org-specific needs
- Everyone sees the same data the same way

**Security teams are not homogeneous.** A bank's SOC has different priorities than a healthcare provider. A threat hunting team looks at different fields than a compliance team.

### TelHawk's Solution: JSON-Based Customization

Power users can customize field displays, table columns, and filter sets through **JSON configuration files**. This is similar to how modern CRMs (like Salesforce or HubSpot) allow field customization.

#### Configuration Levels

**1. System Defaults** (shipped with TelHawk)
```json
{
  "event_classes": {
    "3002": {
      "name": "Authentication",
      "icon": "🔐",
      "card_fields": ["user.name", "src_endpoint.ip", "status", "auth_protocol.name"],
      "table_columns": ["time", "user.name", "src_endpoint.ip", "status", "auth_protocol.name"],
      "primary_filters": ["user.name", "src_endpoint.ip", "status", "auth_protocol.name"],
      "secondary_filters": ["dst_endpoint.hostname", "session.uid", "auth_factor"]
    }
  }
}
```

**2. Organization Overrides** (shared across team)
- Org admin uploads custom config via Settings page
- Overrides system defaults for all users
- Example: Add "cost_center" field to all event types for chargeback reporting

**3. Personal Overrides** (per-user customization)
- Individual users can further customize their view
- Stored in browser localStorage or user profile
- Example: Compliance analyst adds "policy_id" field to Detection events

#### What Can Be Customized?

All customization uses **OCSF field paths** (e.g., `user.name`, `src_endpoint.ip`) rather than display labels. The UI automatically converts field paths to human-readable labels.

**Card Fields:**
```json
{
  "card_fields": [
    "user.name",           // OCSF field paths (displays as "Username")
    "src_endpoint.ip",     // Displays as "Source IP"
    "metadata.labels.environment",  // Custom fields
    "enrichment.threat_intel.risk_score"  // Enrichment data
  ]
}
```

**Table Columns:**
```json
{
  "table_columns": [
    {"field": "time", "label": "Timestamp", "width": "150px", "sortable": true},
    {"field": "user.name", "label": "Username", "width": "120px", "sortable": true},
    {"field": "src_endpoint.ip", "label": "Source IP", "width": "130px", "sortable": false},
    {"field": "status", "label": "Result", "width": "100px", "sortable": true}
  ]
}
```

**Filter Sets:**
```json
{
  "primary_filters": [
    {"field": "user.name", "label": "Username", "type": "text"},
    {"field": "src_endpoint.ip", "label": "Source IP", "type": "ip"},
    {"field": "status", "label": "Status", "type": "enum", "values": ["Success", "Failed", "Locked"]}
  ]
}
```

**Field Transformations:**
```json
{
  "field_transforms": {
    "severity_id": {
      "display": "badge",  // Render as colored badge
      "mapping": {
        "1": {"label": "Info", "color": "blue"},
        "2": {"label": "Low", "color": "green"},
        "3": {"label": "Medium", "color": "yellow"},
        "4": {"label": "High", "color": "orange"},
        "5": {"label": "Critical", "color": "red"}
      }
    }
  }
}
```

#### Configuration UI

**Settings Page → Field Customization:**
1. Select event class to customize
2. Visual editor shows current fields
3. Drag-and-drop to reorder fields
4. Add/remove fields from OCSF schema
5. Preview changes in real-time
6. Export/import JSON for sharing
7. Reset to defaults

**Advanced Mode:**
- Direct JSON editor for power users
- Schema validation with helpful error messages
- Version control for configs (track changes)
- Import from file or URL

#### Use Cases

**Financial Services (add business context):**
```json
{
  "card_fields": [
    "user.name",
    "account_number",      // Custom enrichment field
    "transaction_value",   // Business context
    "compliance_flags"
  ]
}
```

**Healthcare (HIPAA compliance focus):**
```json
{
  "card_fields": [
    "user.name",
    "patient_id_accessed",  // PHI tracking
    "phi_category",
    "consent_status"
  ]
}
```

**MSP/MSSP (multi-organization):**
```json
{
  "card_fields": [
    "client_id",
    "customer_name",
    "user.name",
    "billing_code"
  ]
}
```

#### Implementation Notes

**Field Resolution:**
- JSON config specifies OCSF field paths (e.g., `actor.user.name`)
- Runtime checks if field exists in event
- Falls back gracefully if field is missing
- Supports nested object notation and array indexing

**Performance:**
- Configs cached in memory after first load
- Changes require page refresh or WebSocket push
- No impact on query performance (cosmetic only)

**Validation:**
- JSON schema validation on upload
- Warns if referencing non-existent OCSF fields
- Preview mode shows sample events with config applied

---

## Why This Matters: Real-World Scenarios

### Scenario 1: Failed Login Investigation

**Traditional SIEM:**
1. Search for "authentication"
2. Add filter "status = failed" (3 clicks in dropdown)
3. Scroll through table with half empty columns
4. Click "View Details" on each event to see source IP
5. Copy IP, add new filter for IP (5 more clicks)
6. Repeat for each suspicious IP

**TelHawk (Current + Planned):**
1. Click "Filter by Event Class" → Select "Authentication" *(implemented)*
2. Click "Status" → Select "Failed" *(Level 2 - planned)*
3. **Cards immediately show: Username, Source IP, Status for every event** *(implemented)*
4. Click source IP in card → Auto-adds filter chip *(planned enhancement)*
5. All details visible without clicking "View Details" *(implemented)*

**Time saved:** ~60-80% per investigation

### Scenario 2: Network Lateral Movement

**Traditional SIEM:**
- Hunt through table columns trying to find source/dest IPs
- Half the columns show "N/A" because events are mixed types
- Have to manually correlate 192.168.1.45 appearing in both source and dest columns
- Write complex query to pivot between source/dest

**TelHawk (Current + Planned):**
- Filter to "Network Activity" *(implemented)*
- Every card clearly shows: `Source: 10.0.1.5:54321 → Dest: 192.168.1.45:445` *(implemented)*
- Click any IP → Auto-filter to events involving that IP (source OR dest) *(planned enhancement)*
- Visual pattern recognition: Same IPs appearing repeatedly stand out *(current behavior with card view)*

### Scenario 3: MITRE ATT&CK Hunting

**Traditional SIEM:**
- No easy way to filter by tactic or technique
- Have to know specific field names (`attack.tactic.name`?)
- Write manual query with nested fields
- Results come back in generic table

**TelHawk (Current + Planned):**
- Filter to "Detection Finding" *(implemented)*
- Click "Tactic" → Dropdown shows all tactics seen recently *(Level 2 - planned)*
- Select "Lateral Movement" *(Level 2 - planned)*
- Every card shows: Finding, Tactic, Technique, Risk Score *(implemented)*
- Click technique → Drill down to specific technique *(planned enhancement)*
- Add "High" severity filter → 2 clicks total *(Level 2 - planned)*

---

## Design Guidelines for Future Features

### Information Density Rules

1. **Show, don't hide** - If data exists, display it. Don't make users click for basic context.
2. **Context over chrome** - Navigation bars and menus should be minimal. Content is king.
3. **Responsive without compromise** - Mobile/small screens get scrolling, not hidden features.
4. **Consistent spacing** - White space should separate concepts, not waste screen real estate.

### Filter Bar Expansion

**Level 2 Implementation (Type-Specific Filters):**
- After selecting event class, show 3-5 common filter buttons
- Each filter button opens dropdown with:
  - Text input for manual entry
  - Common values from recent events
  - Smart suggestions based on field type (IP ranges, username patterns, etc.)
- Multiple values for same field = OR logic
- Multiple different fields = AND logic

**Future Enhancements:**
- **Saved filter sets** - Pin common filter combinations, share with team
- **Filter templates** - "Failed Logins", "Lateral Movement", "Data Exfiltration" presets
- **Boolean logic UI** - Visual AND/OR/NOT operators for complex filters
- **Date range chips** - Quick chips: "Last Hour", "Last 24h", "Last 7d" (integrated with time selector)

### View Mode Enhancements

**Card View:**
- **Clickable field values** - Click any field value (IP, username, etc.) to add as filter
- **Inline actions** - Context menu on values: "Investigate in ThreatIntel", "Copy to clipboard", "Add to filter"
- **Related events indicator** - Show count of related events (same user, same IP, etc.)
- **Severity-based visual weight** - Critical events get bolder borders, more prominent display
- **Compact mode toggle** - Reduce spacing for power users who want more events on screen

**Table View:**
- **Clickable field values** - Click any cell value to add as filter (consistent with card view)
- **Column customization** - Drag-and-drop to reorder columns
- **Column header filters** - Click column header to filter/sort by that field
- **Bulk actions** - Select multiple events for bulk operations
- **Export options** - CSV, JSON, Excel with current filters applied
- **Frozen columns** - Lock timestamp column while scrolling horizontally

### Customization Enhancements

**Visual Customization Editor:**
- Drag-and-drop field builder
- Live preview with real events
- Template library (Financial, Healthcare, MSP, etc.)
- Import/export configurations
- Version control with rollback

**Advanced Features:**
- **Custom field calculations** - Create computed fields with formulas
- **Conditional formatting** - Highlight based on rules
- **Field aliases** - Rename fields for org-specific terminology
- **Enrichment integration** - Add external data sources to field options

---

## Anti-Patterns to Avoid

### Things We Will NOT Do

1. **❌ Nested dropdown menus** - If a filter requires more than 2 clicks, it's too buried
2. **❌ Modal dialogs for filters** - Modals hide context. Everything stays in-page.
3. **❌ "Advanced" vs "Simple" mode** - No artificial complexity gates. Progressive disclosure only.
4. **❌ Generic tables with hidden/configurable columns** - Don't force users to configure which columns to show in a one-size-fits-all table. Instead, use type-aware displays: cards for mixed events, type-specific tables when filtered to one event class.
5. **❌ Pagination without context** - Always show "X of Y total" and "Load More" rather than blind page numbers
6. **❌ Mystery fields** - Every field label should be self-explanatory. No `src_ep` or `act_id` abbreviations.

### Query Language Philosophy

**OpenSearch query_string is powerful. We don't hide it.**

But we also don't force users to learn it for basic tasks. The filter bar is training wheels that become transparent as users gain expertise:
- **Beginner**: Uses filter bar exclusively
- **Intermediate**: Uses filter bar, occasionally edits generated query
- **Advanced**: Writes queries directly, filter bar stays out of the way
- **Expert**: Writes complex queries, saves as filter templates for the team

---

## Success Metrics

We measure UI success by:

1. **Time to first result** - How fast can an analyst find relevant events?
2. **Clicks to context** - How many clicks to see full event details? (Goal: 0-1)
3. **Filter adoption rate** - % of searches using filter bar vs raw queries
4. **Investigation velocity** - Events triaged per hour
5. **Cognitive load** - Can an analyst understand an event in <3 seconds?

---

## Future Vision

### Intelligent Filter Suggestions

Based on current events and context, suggest filters:
```
┌─────────────────────────────────────────────────────────────────┐
│ 💡 Suggested Filters:                                           │
├─────────────────────────────────────────────────────────────────┤
│ Most failed logins from: 192.168.1.45 [Add Filter]              │
│ Same username failed on 3 other systems [Investigate]           │
│ Unusual auth protocol (NTLM) for this user [Add Filter]         │
└─────────────────────────────────────────────────────────────────┘
```

### Collaborative Filtering

Analysts on the same team can:
- Share filter combinations: "Here's how I found the lateral movement"
- Pin important filters to team dashboard
- See what filters colleagues are using for active incidents

### Context-Aware Filtering

System learns common investigation patterns:
- "When filtering for Failed Auth, analysts usually also filter by Source IP"
- Auto-suggest next logical filter based on current selection
- Remember per-analyst filter preferences

### Saved Views & Workspaces

Users can save complete workspace configurations:
```json
{
  "name": "Failed Authentication Hunt",
  "event_class": 3002,
  "filters": [
    {"field": "status", "value": "Failed"},
    {"field": "severity_id", "operator": ">=", "value": 3}
  ],
  "view_mode": "table",
  "table_columns": ["time", "user.name", "src_endpoint.ip", "status", "auth_protocol.name"],
  "sort": {"field": "time", "order": "desc"},
  "time_range": "24h"
}
```

Share with team, schedule as reports, or set as personal defaults.

### Advanced Customization

**Computed Fields:**
```json
{
  "computed_fields": {
    "risk_score": {
      "formula": "severity_id * confidence / 100",
      "label": "Risk Score",
      "type": "number"
    }
  }
}
```

**Conditional Formatting:**
```json
{
  "conditional_formats": {
    "user.name": {
      "highlight_if": "user.name IN privileged_users_list",
      "color": "orange"
    }
  }
}
```

---

## Conclusion

**SIEM interfaces should serve analysts, not frustrate them.**

TelHawk's card-based view and progressive filter bar represent a fundamental rethinking of how security data should be presented. We prioritize information density, contextual intelligence, and zero-friction filtering.

Every design decision asks: **Does this help an analyst triage faster?**

If the answer is no, we don't build it.

---

**Related Documentation:**
- [Event Type-Specific Views Implementation](../web/frontend/src/components/events/)
- [OCSF Event Classes](https://schema.ocsf.io/1.1.0/classes)
- [OpenSearch Query DSL](../query/README.md)

**Change Log:**
- 2025-11-08: Initial document created
- 2025-11-08: Card-based view implemented and deployed
- 2025-11-08: Filter bar Level 1 (event class selection) implemented
- 2025-11-08: Major revision - clarified filter bar filters to ONE event type
- 2025-11-08: Added View Modes section (card vs table)
- 2025-11-08: Added Customization section (JSON-based CRM-style configuration)
- TBD: Filter bar Level 2 (type-specific field filters) - planned
- TBD: Table view with type-specific columns - planned
- TBD: Field customization UI - planned
