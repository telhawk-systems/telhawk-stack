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
1. **Fixed columns force "N/A" everywhere** - Authentication events have users but no IPs, network events have IPs but no users, file events have neither
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
- Filters → Auto-update query box
- Query box → Parse and convert to filters when possible
- Complex queries → Show in query box, disable filter chips (with warning)

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

**TelHawk:**
1. Click "Filter by Event Class" → Select "Authentication"
2. Click "Status" → Select "Failed"
3. **Cards immediately show: Username, Source IP, Status for every event**
4. Click source IP in card → Auto-adds filter chip
5. All details visible without clicking "View Details"

**Time saved:** ~60-80% per investigation

### Scenario 2: Network Lateral Movement

**Traditional SIEM:**
- Hunt through table columns trying to find source/dest IPs
- Half the columns show "N/A" because events are mixed types
- Have to manually correlate 192.168.1.45 appearing in both source and dest columns
- Write complex query to pivot between source/dest

**TelHawk:**
- Filter to "Network Activity"
- Every card clearly shows: `Source: 10.0.1.5:54321 → Dest: 192.168.1.45:445`
- Click any IP → Auto-filter to events involving that IP (source OR dest)
- Visual pattern recognition: Same IPs appearing repeatedly stand out

### Scenario 3: MITRE ATT&CK Hunting

**Traditional SIEM:**
- No easy way to filter by tactic or technique
- Have to know specific field names (`attack.tactic.name`?)
- Write manual query with nested fields
- Results come back in generic table

**TelHawk:**
- Filter to "Detection Finding"
- Click "Tactic" → Dropdown shows all tactics seen recently
- Select "Lateral Movement"
- Every card shows: Finding, Tactic, Technique, Risk Score
- Click technique → Drill down to specific technique
- Add "High" severity filter → 2 clicks total

---

## Design Guidelines for Future Features

### Information Density Rules

1. **Show, don't hide** - If data exists, display it. Don't make users click for basic context.
2. **Context over chrome** - Navigation bars and menus should be minimal. Content is king.
3. **Responsive without compromise** - Mobile/small screens get scrolling, not hidden features.
4. **Consistent spacing** - White space should separate concepts, not waste screen real estate.

### Filter Bar Expansion

Future filter types to add:
- **Time range filters** - Quick chips: "Last Hour", "Last 24h", "Last 7d", "Custom"
- **Severity filters** - Visual: Color-coded chips matching severity badges
- **Saved filters** - Pin common filter combinations, share with team
- **Filter templates** - "Failed Logins", "Lateral Movement", "Data Exfiltration" presets
- **Boolean logic UI** - Visual AND/OR/NOT operators for complex filters

### Card Layout Enhancements

- **Inline actions** - Add "Investigate" actions directly on cards (e.g., "Lookup IP in ThreatIntel")
- **Related events indicator** - Show count of related events (same user, same IP, etc.)
- **Severity-based visual weight** - Critical events get bolder borders, more prominent display
- **Timeline view** - Optional: Display events in temporal sequence with connecting lines

---

## Anti-Patterns to Avoid

### Things We Will NOT Do

1. **❌ Nested dropdown menus** - If a filter requires more than 2 clicks, it's too buried
2. **❌ Modal dialogs for filters** - Modals hide context. Everything stays in-page.
3. **❌ "Advanced" vs "Simple" mode** - No artificial complexity gates. Progressive disclosure only.
4. **❌ Hidden columns** - If a column can be hidden, it shouldn't be a column. Use cards.
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
- TBD: Filter bar implementation (planned)
