# TelHawk CLI (`thawk`)

Command-line interface for TelHawk Stack SIEM.

## Installation

```bash
cd cli
go build -o ../bin/thawk .
```

Add to PATH:
```bash
sudo cp ../bin/thawk /usr/local/bin/
# or
export PATH=$PATH:/path/to/telhawk-stack/bin
```

## Quick Start

### Using the Wrapper Script (Recommended)

The `./scripts/thawk` wrapper handles building and execution automatically:

```bash
# Login
./scripts/thawk login -u admin -p admin123

# Search events
./scripts/thawk search "severity:high" --last 1h

# List detection rules
./scripts/thawk rules list
```

### Direct Usage

```bash
# 1. Login
thawk login -u admin -p password

# 2. Create HEC Token for Ingestion
thawk token create --name production-ingest

# 3. Send Test Event
thawk ingest send -m "Security alert detected" -t <your-hec-token>

# 4. Search Events
thawk search "index=* | head 10"
thawk search "severity=high" --last 24h
```

## Commands

### Authentication

```bash
# Login
thawk login -u username -p password

# Check current user
thawk whoami

# Logout
thawk logout
```

### HEC Token Management

```bash
# Create HEC token
thawk token create --name my-token --expires 30d

# List tokens
thawk token list

# Revoke token
thawk token revoke <token-string>
```

### Search

```bash
# Basic SPL-style search
thawk search "index=security"

# Time-based search
thawk search "source=firewall" --last 1h
thawk search "*" --earliest "-7d" --latest "now"

# JSON output
thawk search "error" --output json

# Raw JSON query (via stdin)
echo '{"filter":{"class_uid":3002}}' | thawk search --raw

# Raw JSON query from file
thawk search --raw < query.json
cat query.json | thawk search --raw
```

### Detection Rules

```bash
# List all rules
thawk rules list

# Get rule details
thawk rules get <rule-id>

# Create rule from JSON file
thawk rules create rules/failed_logins.json
```

### Alerts

```bash
# List alerts
thawk alerts list

# List cases
thawk alerts cases list
```

### Ingestion

```bash
# Send event with message
thawk ingest send --message "Login failed" --token <hec-token>

# Send JSON event
thawk ingest send --json '{"user":"admin","action":"login"}' --token <hec-token>

# Specify source and sourcetype
thawk ingest send -m "Alert" -t <token> --source app1 --sourcetype syslog
```

### Event Seeder (Testing)

```bash
# Generate events from detection rules
thawk seeder run --token <hec-token> --from-rules ./alerting/rules/

# List supported rules
thawk seeder list-rules ./alerting/rules/
```

## Configuration

All CLI operations go through the web backend (`http://localhost:3000` by default).

Config file location: `~/.thawk/config.yaml`

Example:
```yaml
current_profile: default
defaults:
  auth_url: http://localhost:3000
  ingest_url: http://localhost:8088
  query_url: http://localhost:3000
  rules_url: http://localhost:3000
  alerting_url: http://localhost:3000
profiles:
  default:
    auth_url: http://localhost:3000
    access_token: eyJhbGc...
    refresh_token: abc123...
  production:
    auth_url: https://siem.company.com
    access_token: eyJhbGc...
    refresh_token: def456...
```

### Multiple Profiles

```bash
# Login to different profile
thawk login -u user -p pass --profile production

# Use specific profile
thawk search "index=*" --profile production
```

## Output Formats

```bash
# Table format (default)
thawk token list

# JSON format
thawk token list --output json

# YAML format
thawk token list --output yaml
```

## Global Flags

- `--config <file>` - Config file path (default: ~/.thawk/config.yaml)
- `--profile <name>` - Profile to use (default: default)
- `--output <format>` - Output format: table, json, yaml (default: table)

## Examples

### SOC Analyst Workflow

```bash
# Login
thawk login -u analyst -p secure123

# Search for recent high-severity events
thawk search "severity=high OR severity=critical" --last 1h

# Raw JSON query for authentication failures
echo '{"filter":{"class_uid":3002,"status_id":2}}' | thawk search --raw --output json

# Export search results
thawk search "index=security" --last 24h --output json > results.json
```

### Security Engineer Workflow

```bash
# Create HEC token for new data source
thawk token create --name "firewall-01" --expires 365d

# Test ingestion
thawk ingest send \
  --json '{"src_ip":"10.0.0.1","dst_ip":"8.8.8.8","action":"block"}' \
  --token <hec-token> \
  --source firewall-01 \
  --sourcetype firewall

# Verify data ingestion
thawk search "source=firewall-01" --last 5m
```

## Development

```bash
# Run without building
go run . auth login -u test -p test

# Build
go build -o thawk .

# Install globally
go install
```

## Shell Completion

```bash
# Bash
thawk completion bash > /etc/bash_completion.d/thawk

# Zsh
thawk completion zsh > "${fpath[1]}/_thawk"

# Fish
thawk completion fish > ~/.config/fish/completions/thawk.fish
```

## See Also

- [CLI Configuration Guide](../docs/CLI_CONFIGURATION.md)
- [Auth Service](../auth/README.md)
- [Ingestion Service](../ingest/README.md)
- [Query Service](../query/README.md)
