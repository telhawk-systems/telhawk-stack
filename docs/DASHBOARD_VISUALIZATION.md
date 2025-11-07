# Dashboard Visualization Components

## Overview

The TelHawk web UI now includes comprehensive dashboard visualization components that provide real-time security event analytics using the OpenSearch aggregation API.

## Components

### DashboardOverview

The main dashboard component that displays security metrics, charts, and visualizations.

**Features:**
- Time range selector (15m, 1h, 24h, 7d)
- Auto-refresh toggle (30-second interval)
- Manual refresh button
- Four metric cards showing key statistics
- Three interactive charts

**Location:** `web/frontend/src/components/dashboard/DashboardOverview.tsx`

### MetricCard

Reusable card component for displaying key metrics with optional icons and trend indicators.

**Props:**
- `title`: Card title
- `value`: Primary metric value (string or number)
- `subtitle`: Optional secondary text
- `icon`: Optional React icon component
- `trend`: Optional trend indicator with direction

**Location:** `web/frontend/src/components/dashboard/MetricCard.tsx`

### SeverityChart

Pie chart visualization showing event distribution by severity level.

**Features:**
- Color-coded severity levels (Critical=red, High=orange, Medium=yellow, Low=blue, Info=gray)
- Interactive legend
- Percentage labels
- Hover tooltips

**Location:** `web/frontend/src/components/dashboard/SeverityChart.tsx`

### EventClassChart

Horizontal bar chart showing top event classes by count.

**Features:**
- Displays top 10 event classes
- Truncated labels for readability
- Interactive tooltips
- Count-based sorting

**Location:** `web/frontend/src/components/dashboard/EventClassChart.tsx`

### TimelineChart

Line chart showing event volume over time.

**Features:**
- Hourly bucketing (configurable)
- Time-formatted X-axis labels
- Interactive data points
- Grid lines for readability

**Location:** `web/frontend/src/components/dashboard/TimelineChart.tsx`

## Dashboard Metrics

The dashboard fetches aggregated metrics from the query service API:

```typescript
{
  severity_count: {        // Terms aggregation on severity field
    buckets: [{ key: "1", doc_count: 42 }, ...]
  },
  events_by_class: {       // Terms aggregation on class_name field
    buckets: [{ key: "Authentication", doc_count: 156 }, ...]
  },
  timeline: {              // Date histogram with 1h interval
    buckets: [{ key: 1699027200000, doc_count: 23 }, ...]
  },
  unique_users: {          // Cardinality aggregation
    value: 15
  },
  unique_ips: {            // Cardinality aggregation
    value: 87
  }
}
```

## API Integration

### getDashboardMetrics

New API method in `web/frontend/src/services/api.ts`:

```typescript
async getDashboardMetrics(timeRange?: { start: string; end: string }): Promise<any>
```

**Request:**
- POST to `/api/query/api/v1/search`
- Query with time range filter
- `limit: 0` (no documents, only aggregations)
- Five aggregations (severity, class, timeline, users, IPs)

**Response:**
- Total matches count
- Aggregation buckets and values
- Request latency

## Usage

The dashboard is integrated into the main DashboardPage with tab navigation:

```tsx
<DashboardPage>
  <Tab>Overview</Tab>    <!-- DashboardOverview component -->
  <Tab>Search</Tab>      <!-- Existing SearchConsole -->
</DashboardPage>
```

### Time Range Selection

Users can select from predefined time ranges:
- Last 15 minutes
- Last hour
- Last 24 hours (default)
- Last 7 days

Time ranges are automatically calculated and passed to the API as ISO 8601 timestamps.

### Auto-Refresh

When enabled, the dashboard automatically refreshes every 30 seconds to display the latest metrics. This is useful for SOC operators monitoring live events.

## Chart Library

The dashboard uses **Recharts** (v3.3.0), a composable charting library built on React components and D3.

**Why Recharts:**
- Native React components
- Responsive by default
- Lightweight (~150KB minified)
- SVG-based rendering
- Extensive chart types
- Active maintenance

## Metric Cards

Four key metrics are displayed:

1. **Total Events** - Count of all events in time range
2. **Critical Events** - Count of severity level 1 events
3. **Unique Users** - Distinct actor.user.name values
4. **Unique IPs** - Distinct src_endpoint.ip values

Each card includes an icon and descriptive subtitle.

## Performance Considerations

- **Aggregation-only queries**: `limit: 0` skips document retrieval, only computing aggregations
- **Client-side caching**: Time range selection updates cause new API calls
- **Lazy loading**: Charts only render when data is available
- **Debouncing**: Manual refresh prevents rapid successive API calls

## Future Enhancements

Potential improvements for the dashboard:

1. **Saved Views**: Store and recall custom time ranges
2. **More Aggregations**: 
   - Top source/destination IPs
   - Top targeted hosts
   - Alert type distribution
   - Success/failure rates
3. **Custom Chart Selection**: User-configurable dashboard widgets
4. **Drill-down**: Click chart elements to filter events
5. **Export**: Download charts as PNG or data as CSV
6. **Threshold Alerts**: Visual indicators when metrics exceed thresholds
7. **Comparison Mode**: Compare current period to previous period

## Development

### Building

```bash
cd web/frontend
npm install
npm run build
```

### Running Locally

```bash
npm run dev
```

The dashboard will be available at `http://localhost:5173` (default Vite port).

## Testing

To test the dashboard:

1. Ensure services are running (docker-compose up)
2. Ingest sample events via HEC endpoint
3. Navigate to Dashboard → Overview tab
4. Verify metrics load and charts render
5. Test time range selector
6. Test auto-refresh toggle

## Troubleshooting

**No data showing:**
- Check if events exist in OpenSearch
- Verify query service is running
- Check browser console for API errors
- Confirm time range includes events

**Charts not rendering:**
- Check for JavaScript errors
- Verify Recharts package is installed
- Ensure data format matches chart expectations

**Slow loading:**
- Check OpenSearch query performance
- Consider reducing aggregation bucket sizes
- Verify network latency to backend
