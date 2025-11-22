# TelHawk Design Specification

A comprehensive UI/UX design system for TelHawk Security Intelligence Platform.

## Design Philosophy

TelHawk's interface prioritizes **clarity**, **efficiency**, and **security awareness**. As a SIEM platform, users need to quickly identify threats, investigate events, and take action. The design emphasizes:

- **Information density** without visual clutter
- **Severity-based visual hierarchy** (critical issues surface immediately)
- **Consistent navigation** across all workflows
- **Dark mode sidebar** for reduced eye strain during long monitoring sessions

---

## Layout Structure

### Overall Page Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│ Sidebar (240px)  │  Main Content Area                               │
│                  │  ┌─────────────────────────────────────────────┐ │
│ ┌──────────────┐ │  │ Top Bar (56px)                              │ │
│ │ Logo/Brand   │ │  │ [Search] [Filters] [Time Range] [User]     │ │
│ └──────────────┘ │  └─────────────────────────────────────────────┘ │
│                  │                                                   │
│ ┌──────────────┐ │  ┌─────────────────────────────────────────────┐ │
│ │ Primary Nav  │ │  │                                             │ │
│ │ - Dashboard  │ │  │ Page Content                                │ │
│ │ - Search     │ │  │ (padding: 24px)                             │ │
│ │ - Events     │ │  │                                             │ │
│ │ - Alerts     │ │  │                                             │ │
│ │ - Rules      │ │  │                                             │ │
│ └──────────────┘ │  │                                             │ │
│                  │  │                                             │ │
│ ┌──────────────┐ │  │                                             │ │
│ │ Smart Views  │ │  │                                             │ │
│ │ (Saved)      │ │  │                                             │ │
│ └──────────────┘ │  │                                             │ │
│                  │  │                                             │ │
│ ┌──────────────┐ │  └─────────────────────────────────────────────┘ │
│ │ Settings     │ │                                                   │
│ │ Admin        │ │                                                   │
│ └──────────────┘ │                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Measurements

| Element | Value | Notes |
|---------|-------|-------|
| Sidebar width | 240px | Collapsible to 64px (icons only) |
| Top bar height | 56px | Fixed position |
| Content padding | 24px | All sides |
| Card gap | 16px | Between metric cards |
| Border radius (cards) | 8px | Consistent across all cards |
| Border radius (buttons) | 6px | Slightly tighter than cards |
| Border radius (chips/badges) | 4px | Small elements |

---

## Color Palette

### Core Colors

```css
:root {
  /* Sidebar & Dark Elements */
  --sidebar-bg: #1e2530;
  --sidebar-bg-hover: #2d3748;
  --sidebar-text: #a0aec0;
  --sidebar-text-active: #ffffff;
  --sidebar-accent: #4299e1;

  /* Main Background */
  --bg-primary: #f7fafc;
  --bg-secondary: #edf2f7;

  /* Cards & Surfaces */
  --surface-primary: #ffffff;
  --surface-elevated: #ffffff;
  --surface-border: #e2e8f0;

  /* Text */
  --text-primary: #1a202c;
  --text-secondary: #4a5568;
  --text-muted: #718096;
  --text-placeholder: #a0aec0;

  /* Brand */
  --brand-primary: #2b6cb0;
  --brand-primary-hover: #2c5282;
  --brand-secondary: #4299e1;

  /* Semantic - Severity */
  --severity-critical: #c53030;
  --severity-critical-bg: #fff5f5;
  --severity-high: #dd6b20;
  --severity-high-bg: #fffaf0;
  --severity-medium: #d69e2e;
  --severity-medium-bg: #fffff0;
  --severity-low: #38a169;
  --severity-low-bg: #f0fff4;
  --severity-info: #3182ce;
  --severity-info-bg: #ebf8ff;

  /* Status */
  --status-success: #48bb78;
  --status-success-bg: #f0fff4;
  --status-warning: #ed8936;
  --status-warning-bg: #fffaf0;
  --status-error: #f56565;
  --status-error-bg: #fff5f5;
  --status-info: #4299e1;
  --status-info-bg: #ebf8ff;

  /* Trends */
  --trend-positive: #48bb78;
  --trend-negative: #f56565;
  --trend-neutral: #718096;
}
```

### Tailwind Configuration

```javascript
// tailwind.config.js
export default {
  theme: {
    extend: {
      colors: {
        sidebar: {
          DEFAULT: '#1e2530',
          hover: '#2d3748',
          text: '#a0aec0',
          active: '#ffffff',
          accent: '#4299e1',
        },
        severity: {
          critical: { DEFAULT: '#c53030', bg: '#fff5f5' },
          high: { DEFAULT: '#dd6b20', bg: '#fffaf0' },
          medium: { DEFAULT: '#d69e2e', bg: '#fffff0' },
          low: { DEFAULT: '#38a169', bg: '#f0fff4' },
          info: { DEFAULT: '#3182ce', bg: '#ebf8ff' },
        },
      },
      spacing: {
        'sidebar': '240px',
        'sidebar-collapsed': '64px',
        'topbar': '56px',
      },
    },
  },
}
```

---

## Typography

### Font Stack

```css
:root {
  --font-sans: 'Inter', system-ui, -apple-system, BlinkMacSystemFont,
               'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  --font-mono: 'Roboto Mono', 'SF Mono', Monaco, Consolas,
               'Liberation Mono', 'Courier New', monospace;
}
```

**Google Fonts Import:**
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Roboto+Mono:wght@400;500&display=swap" rel="stylesheet">
```

### Type Scale

| Name | Size | Weight | Line Height | Usage |
|------|------|--------|-------------|-------|
| Display | 36px | 700 | 1.2 | Large metric numbers |
| Heading 1 | 24px | 600 | 1.3 | Page titles |
| Heading 2 | 20px | 600 | 1.4 | Section headers |
| Heading 3 | 16px | 600 | 1.5 | Card titles |
| Body | 14px | 400 | 1.5 | Primary content |
| Body Small | 13px | 400 | 1.5 | Secondary content |
| Caption | 12px | 500 | 1.4 | Labels, badges |
| Mono | 13px | 400 | 1.6 | Code, queries, IPs |

### Tailwind Classes

```html
<!-- Display (large metrics) -->
<span class="text-4xl font-bold text-gray-900">1,247</span>

<!-- Page Title -->
<h1 class="text-2xl font-semibold text-gray-900">Security Dashboard</h1>

<!-- Section Header -->
<h2 class="text-xl font-semibold text-gray-800">Recent Alerts</h2>

<!-- Card Title -->
<h3 class="text-base font-semibold text-gray-700">Failed Logins</h3>

<!-- Body Text -->
<p class="text-sm text-gray-600">Event details and description</p>

<!-- Caption/Label -->
<span class="text-xs font-medium text-gray-500 uppercase tracking-wide">
  Last 24 Hours
</span>

<!-- Monospace (IPs, queries, hashes) -->
<code class="font-mono text-sm text-gray-800 bg-gray-100 px-1 rounded">
  192.168.1.100
</code>
```

---

## Component Specifications

### Sidebar

```
┌────────────────────────────┐
│  [Logo] TelHawk            │  <- 64px height, brand area
│  Security Intelligence     │
├────────────────────────────┤
│                            │
│  📊 Dashboard              │  <- 44px height per item
│  🔍 Search                 │
│  📋 Events                 │
│  🚨 Alerts                 │
│  📜 Rules                  │
│                            │
├────────────────────────────┤
│  SAVED SEARCHES    [+]     │  <- Section header, 32px
│                            │
│  🔥 Critical Alerts        │  <- Saved items, 40px each
│  👤 Failed Logins          │
│  🌐 Network Anomalies      │
│                            │
├────────────────────────────┤
│  ⚙️ Settings               │
│  👥 Users (admin)          │
│  🔑 Tokens (admin)         │
└────────────────────────────┘
```

**Specifications:**
- Background: `#1e2530`
- Width: 240px (64px collapsed)
- Nav item height: 44px
- Nav item padding: 12px 16px
- Icon size: 20px
- Icon-text gap: 12px
- Active indicator: 3px left border, `#4299e1`
- Hover state: `#2d3748` background

### Top Bar

```
┌─────────────────────────────────────────────────────────────────────┐
│ [← →]  [🔍 Search events, IPs, users...            ]  [⏱️ Last 24h ▾]│
│                                                                       │
│ [Filter: severity:high ×] [Filter: source:firewall ×]     [👤 Admin ▾]│
└─────────────────────────────────────────────────────────────────────┘
```

**Specifications:**
- Height: 56px
- Background: `#ffffff`
- Border bottom: 1px solid `#e2e8f0`
- Search input width: 400px (flexible)
- Search input height: 40px
- Filter chips height: 28px

### Metric Cards (Dashboard)

```
┌─────────────────────────────┐
│  Total Events               │  <- Label, 12px uppercase
│                             │
│  1,247,893                  │  <- Value, 36px bold
│                             │
│  ↑ 12% ●                    │  <- Trend, 12px with icon
│  vs last period             │
└─────────────────────────────┘
```

**Specifications:**
- Background: `#ffffff`
- Border radius: 8px
- Padding: 20px
- Shadow: `0 1px 3px rgba(0,0,0,0.1)`
- Min width: 200px
- Grid: 3-4 columns on desktop

**Trend Indicators:**
- Positive: `↑` or `▲` in green (`#48bb78`)
- Negative: `↓` or `▼` in red (`#f56565`)
- Neutral: `→` or `–` in gray (`#718096`)

### Alert/Event Cards

```
┌─────────────────────────────────────────────────────────────────────┐
│ ● CRITICAL   Failed Login Brute Force                    2 min ago  │
│                                                                      │
│ Multiple failed login attempts from 192.168.1.100 targeting admin   │
│                                                                      │
│ Source: auth-service  │  Rule: brute_force_login  │  Events: 47     │
│                                                                      │
│ [View Details]  [Acknowledge]  [Create Case]                        │
└─────────────────────────────────────────────────────────────────────┘
```

**Severity Styling:**

| Severity | Left Border | Badge BG | Badge Text |
|----------|-------------|----------|------------|
| Critical | 4px `#c53030` | `#fff5f5` | `#c53030` |
| High | 4px `#dd6b20` | `#fffaf0` | `#dd6b20` |
| Medium | 4px `#d69e2e` | `#fffff0` | `#d69e2e` |
| Low | 4px `#38a169` | `#f0fff4` | `#38a169` |
| Info | 4px `#3182ce` | `#ebf8ff` | `#3182ce` |

### Data Tables

```
┌─────────────────────────────────────────────────────────────────────┐
│ Time ▼        │ Severity │ Source         │ Message                 │
├───────────────┼──────────┼────────────────┼─────────────────────────┤
│ 14:32:01      │ ● HIGH   │ 192.168.1.50   │ SSH auth failure        │
│ 14:31:58      │ ● MEDIUM │ fw-01          │ Connection blocked      │
│ 14:31:55      │ ● LOW    │ web-server     │ Request completed       │
└─────────────────────────────────────────────────────────────────────┘
```

**Specifications:**
- Header background: `#f7fafc`
- Header text: 12px uppercase, `#4a5568`
- Row height: 48px
- Row hover: `#f7fafc`
- Row border: 1px solid `#e2e8f0`
- Alternating rows: Optional, `#fafbfc`

### Filter Chips

```
┌──────────────────────┐
│ severity:high    ×   │
└──────────────────────┘
```

**Specifications:**
- Height: 28px
- Padding: 4px 8px 4px 12px
- Border radius: 4px
- Background: `#edf2f7`
- Text: 13px `#4a5568`
- Logo icon: 14px, hover `#e53e3e`

### Time Range Selector

```
┌──────────────────────────────┐
│ ⏱️ Last 24 Hours          ▾ │
└──────────────────────────────┘

Dropdown:
┌──────────────────────────────┐
│ Last 15 minutes              │
│ Last 1 hour                  │
│ Last 24 hours           ✓    │
│ Last 7 days                  │
│ Last 30 days                 │
│ ─────────────────────────    │
│ Custom range...              │
└──────────────────────────────┘
```

**Specifications:**
- Height: 40px
- Min width: 160px
- Border: 1px solid `#e2e8f0`
- Border radius: 6px

---

## Page Layouts

### Dashboard

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Security Overview                             │
│  [This Week ▾]  vs [Last Week ▾]                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │ Total      │  │ Critical   │  │ Open       │  │ Mean Time  │    │
│  │ Events     │  │ Alerts     │  │ Cases      │  │ to Detect  │    │
│  │ 1.2M       │  │ 23         │  │ 8          │  │ 4m 32s     │    │
│  │ ↑ 15%      │  │ ↓ 8%       │  │ ↑ 2        │  │ ↓ 12%      │    │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘    │
│                                                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────┐  ┌─────────────────────────┐   │
│  │ Events by Severity              │  │ Top Sources             │   │
│  │ [Stacked area chart]            │  │                         │   │
│  │                                 │  │ 1. firewall-01   45%    │   │
│  │                                 │  │ 2. auth-service  23%    │   │
│  │                                 │  │ 3. web-proxy     18%    │   │
│  │                                 │  │ 4. dns-server    14%    │   │
│  └─────────────────────────────────┘  └─────────────────────────┘   │
│                                                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Recent Alerts                                    [View All →]       │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ ● CRITICAL  Brute force detected on admin account   2m ago    │  │
│  │ ● HIGH      Unusual outbound traffic from server-12  15m ago  │  │
│  │ ● MEDIUM    Failed authentication from unknown IP    23m ago  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Search Page

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ 🔍 severity:high AND source_ip:192.168.*                      │  │
│  │                                                                │  │
│  │ [Search]  [Save Search]                                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Active Filters:                                                     │
│  [severity:high ×] [time:last-24h ×] [class:network ×]              │
│                                                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Showing 1,247 of 15,893 events                    [Export ▾]       │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Time        │ Severity │ Class      │ Source      │ Message   │  │
│  ├─────────────┼──────────┼────────────┼─────────────┼───────────┤  │
│  │ 14:32:01    │ ● HIGH   │ Network    │ 192.168.1.1 │ Blocked   │  │
│  │ 14:31:58    │ ● HIGH   │ Auth       │ 192.168.1.5 │ Failed    │  │
│  │ ...         │          │            │             │           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  [← Prev]  Page 1 of 127  [Next →]                                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Alerts Page

```
┌─────────────────────────────────────────────────────────────────────┐
│  Alerts                                                              │
│  [All ▾] [Open] [Acknowledged] [Resolved]          [⏱️ Last 7 days] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ ● CRITICAL                                                     │  │
│  │ Brute Force Attack Detected                         2 min ago  │  │
│  │                                                                │  │
│  │ 47 failed login attempts from 192.168.1.100 in 5 minutes      │  │
│  │                                                                │  │
│  │ Rule: failed_login_threshold  │  Source: auth-service         │  │
│  │                                                                │  │
│  │ [View Events]  [Acknowledge]  [Create Case]                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ ● HIGH                                                         │  │
│  │ Suspicious Outbound Connection                     15 min ago  │  │
│  │ ...                                                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Responsive Breakpoints

| Breakpoint | Width | Layout Changes |
|------------|-------|----------------|
| Mobile | < 640px | Sidebar hidden, hamburger menu, single column |
| Tablet | 640-1024px | Collapsed sidebar (64px), 2-column grid |
| Desktop | 1024-1440px | Full sidebar, 3-column grid |
| Wide | > 1440px | Full sidebar, 4-column grid, max-width container |

### Mobile Adaptations

- Sidebar becomes slide-out drawer
- Metric cards stack vertically
- Tables become card-based lists
- Filter chips scroll horizontally
- Search bar full width

---

## Animations & Transitions

```css
/* Default transition */
.transition-default {
  transition: all 150ms ease-in-out;
}

/* Hover states */
.transition-hover {
  transition: background-color 150ms ease, color 150ms ease;
}

/* Modal/Drawer */
.transition-modal {
  transition: opacity 200ms ease-out, transform 200ms ease-out;
}

/* Sidebar collapse */
.transition-sidebar {
  transition: width 200ms ease-in-out;
}
```

### Motion Principles

1. **Subtle** - Animations enhance, never distract
2. **Fast** - 150-200ms for most interactions
3. **Purposeful** - Motion indicates state change or relationship
4. **Consistent** - Same transition for same action type

---

## Accessibility

### Color Contrast

All text meets WCAG 2.1 AA standards:
- Normal text: 4.5:1 minimum contrast ratio
- Large text: 3:1 minimum contrast ratio
- UI components: 3:1 minimum contrast ratio

### Focus States

```css
/* Focus ring for keyboard navigation */
.focus-ring:focus-visible {
  outline: 2px solid #4299e1;
  outline-offset: 2px;
}

/* Alternative for dark backgrounds */
.focus-ring-light:focus-visible {
  outline: 2px solid #ffffff;
  outline-offset: 2px;
}
```

### ARIA Guidelines

- All interactive elements have accessible names
- Status messages use `role="status"` or `aria-live`
- Tables use proper `<th>` with `scope` attributes
- Modals trap focus and have `aria-modal="true"`
- Severity indicators have text alternatives

---

## Icons

### Recommended Icon Sets

1. **Heroicons** (primary) - Consistent with Tailwind ecosystem
2. **Lucide** (alternative) - Extensive, well-maintained

### Icon Sizing

| Context | Size | Class |
|---------|------|-------|
| Inline text | 16px | `w-4 h-4` |
| Navigation | 20px | `w-5 h-5` |
| Buttons | 20px | `w-5 h-5` |
| Empty states | 48px | `w-12 h-12` |
| Illustrations | 64-96px | `w-16 h-16` to `w-24 h-24` |

### Security-Specific Icons

| Concept | Icon | Usage |
|---------|------|-------|
| Alert/Warning | `ExclamationTriangle` | Alerts, warnings |
| Security | `ShieldCheck` | Security status |
| Lock | `LockClosed` | Authentication |
| Search | `MagnifyingGlass` | Search functionality |
| Filter | `Funnel` | Filtering |
| Network | `Globe` | Network events |
| User | `User` | User events |
| Time | `Clock` | Time-based elements |
| Rule | `DocumentText` | Detection rules |
| Case | `Briefcase` | Security cases |

---

## Implementation Checklist

### Phase 1: Foundation
- [ ] Update `tailwind.config.js` with custom colors and spacing
- [ ] Create CSS variables in `index.css`
- [ ] Add Google Fonts import (Inter + Roboto Mono) to `index.html`

### Phase 2: Layout
- [ ] Implement new sidebar component
- [ ] Update top bar with search and filters
- [ ] Create responsive container system
- [ ] Add sidebar collapse functionality

### Phase 3: Components
- [ ] Metric card component
- [ ] Alert/event card component
- [ ] Filter chip component
- [ ] Time range selector
- [ ] Data table with sorting

### Phase 4: Pages
- [ ] Dashboard with metrics grid
- [ ] Search page redesign
- [ ] Alerts page redesign
- [ ] Rules management page

### Phase 5: Polish
- [ ] Add transitions and animations
- [ ] Implement dark mode (optional)
- [ ] Accessibility audit
- [ ] Performance optimization

---

## File Organization

```
src/
├── components/
│   ├── layout/
│   │   ├── Sidebar.tsx
│   │   ├── TopBar.tsx
│   │   ├── Layout.tsx
│   │   └── Container.tsx
│   ├── common/
│   │   ├── MetricCard.tsx
│   │   ├── FilterChip.tsx
│   │   ├── TimeRangeSelector.tsx
│   │   ├── SeverityBadge.tsx
│   │   └── DataTable.tsx
│   ├── alerts/
│   │   ├── AlertCard.tsx
│   │   └── AlertList.tsx
│   ├── events/
│   │   └── EventsTable.tsx
│   └── dashboard/
│       ├── MetricsGrid.tsx
│       └── RecentAlerts.tsx
├── styles/
│   └── index.css
└── types/
    └── design.ts  # TypeScript types for design tokens
```

---

## Reference

Based on analysis of modern dashboard patterns with adaptations for security/SIEM use cases.

Key influences:
- Information density optimized for monitoring workflows
- Severity-based visual hierarchy (security-first)
- Dark sidebar for reduced eye strain during extended use
- Monospace typography for technical data (IPs, hashes, queries)
