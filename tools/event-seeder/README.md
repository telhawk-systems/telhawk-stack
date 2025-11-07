# Event Seeder

A tool for generating realistic fake OCSF events and sending them to the TelHawk Stack HEC endpoint for development and testing.

## Features

- Generates events for 7 major OCSF event classes:
  - **Authentication (3002)**: Login attempts, MFA, logouts with realistic success/failure rates
  - **Network Activity (4001)**: TCP/UDP/ICMP connections, firewall events
  - **Process Activity (1007)**: Process launches with command lines and parent processes
  - **File Activity (4006)**: File create/read/update/delete/rename operations
  - **DNS Activity (4003)**: DNS queries and responses with various record types
  - **HTTP Activity (4002)**: HTTP requests with realistic status codes and paths
  - **Detection Finding (2004)**: Security alerts with MITRE ATT&CK tactics

- Configurable event volume, timing, and distribution
- Batch sending for efficiency
- Realistic fake data using gofakeit library
- Optional time spreading for historical data simulation

## Installation

```bash
cd tools/event-seeder
go mod download
go build
```

## Usage

### Basic Usage

Generate 100 mixed events:

```bash
./event-seeder -token YOUR_HEC_TOKEN
```

### Custom Event Count

Generate 1000 events:

```bash
./event-seeder -token YOUR_HEC_TOKEN -count 1000
```

### Specific Event Types

Generate only authentication and detection events:

```bash
./event-seeder -token YOUR_HEC_TOKEN -types auth,detection
```

### Historical Data

Spread events over the past 7 days:

```bash
./event-seeder -token YOUR_HEC_TOKEN -count 10000 -time-spread 168h
```

### High-Speed Ingestion

Send events as fast as possible with larger batches:

```bash
./event-seeder -token YOUR_HEC_TOKEN -count 10000 -interval 0 -batch-size 100
```

### Custom HEC Endpoint

Point to a different HEC endpoint:

```bash
./event-seeder -hec-url https://my-hec-endpoint:8088 -token YOUR_HEC_TOKEN
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-hec-url` | `http://localhost:8088` | HEC endpoint URL |
| `-token` | *required* | HEC authentication token |
| `-count` | `100` | Number of events to generate |
| `-interval` | `100ms` | Interval between batches (0 for no delay) |
| `-batch-size` | `10` | Number of events per batch |
| `-types` | `auth,network,process,file,dns,http,detection` | Comma-separated event types |
| `-time-spread` | `24h` | Spread events over this time period (0 for real-time) |

## Event Types

Available event types for the `-types` flag:

- `auth` - Authentication events (login, logout, MFA, password change)
  - Logins: 85% success rate with realistic failure reasons
  - Logout/MFA/Password changes: 98% success rate (rare failures only)
- `network` - Network activity (TCP/UDP/ICMP connections, firewall events)
- `process` - Process activity (launches with command lines and parent processes)
- `file` - File operations (create, read, update, delete, rename)
- `dns` - DNS queries (A, AAAA, CNAME, MX records with responses)
- `http` - HTTP requests (GET/POST/PUT/DELETE with realistic status codes)
- `detection` - Security detections (MITRE ATT&CK tactics, medium-high severity)

### Event Characteristics

All generated events include:
- Realistic fake data using gofakeit library
- Proper OCSF schema compliance (class_uid, activity_id, severity_id)
- Nested objects (user, actor, endpoints, process hierarchy)
- Appropriate severity levels based on event type and outcome
- Metadata identifying source as "Event Seeder"
- Configurable timestamps (real-time or historical with time-spread)

## Examples

### Development Testing

Quickly populate with a small diverse dataset:

```bash
./event-seeder -token YOUR_TOKEN -count 50 -interval 50ms
```

### Load Testing

Generate high volume for performance testing:

```bash
./event-seeder -token YOUR_TOKEN -count 50000 -interval 0 -batch-size 100
```

### Security Scenario Simulation

Generate a week of authentication and detection events:

```bash
./event-seeder -token YOUR_TOKEN \
  -types auth,detection \
  -count 5000 \
  -time-spread 168h \
  -interval 0
```

### Dashboard Population

Create diverse recent events for dashboard testing:

```bash
./event-seeder -token YOUR_TOKEN \
  -count 1000 \
  -time-spread 1h \
  -interval 0
```

## Getting a HEC Token

Use the TelHawk CLI to create a HEC token:

```bash
# From the telhawk-stack directory
docker-compose exec cli thawk hec create --name "Event Seeder"
```

Or create one through the web UI after logging in.

## Verifying Events

After running the seeder, verify events in the web UI:

1. Navigate to http://localhost:3000
2. Log in with your credentials
3. Go to the Search tab
4. Set time range to "Last 24 hours" or appropriate range
5. Run a query: `*` or filter by class: `class_name:Authentication`

Or use the query API directly:

```bash
curl -X POST http://localhost:8082/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "*",
    "time_range": {"start": "now-1h", "end": "now"}
  }'
```
