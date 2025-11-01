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

### 1. Login
```bash
thawk auth login -u admin -p password --auth-url http://localhost:8080
```

### 2. Create HEC Token for Ingestion
```bash
thawk token create --name production-ingest
```

### 3. Send Test Event
```bash
thawk ingest send -m "Security alert detected" -t <your-hec-token>
```

### 4. Search Events
```bash
thawk search "index=* | head 10"
thawk search "severity=high" --last 24h
```

## Commands

### Authentication

```bash
# Login
thawk auth login -u username -p password

# Check current user
thawk auth whoami

# Logout
thawk auth logout
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
# Basic search
thawk search "index=security"

# Time-based search
thawk search "source=firewall" --last 1h
thawk search "*" --earliest "-7d" --latest "now"

# JSON output
thawk search "error" --output json
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

## Configuration

Config file location: `~/.thawk/config.yaml`

Example:
```yaml
current_profile: default
profiles:
  default:
    auth_url: http://localhost:8080
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
thawk auth login -u user -p pass --profile production

# Use specific profile
thawk search "index=*" --profile production

# Switch default profile
thawk config set-profile production
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
thawk auth login -u analyst -p secure123

# Search for recent high-severity events
thawk search "severity=high OR severity=critical" --last 1h

# Create alert query
thawk alert create \
  --name "Failed Logins" \
  --query "action=login AND result=failed | stats count by user" \
  --threshold "count > 5"

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

- [Auth Service](../auth/README.md)
- [Ingestion Service](../ingest/README.md)
- [Query Service](../query/README.md)
