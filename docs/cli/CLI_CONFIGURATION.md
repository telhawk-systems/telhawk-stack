# TelHawk CLI Configuration Guide

## Overview

The `thawk` CLI routes all API operations through the web backend (`http://localhost:3000`) for consistent access control and authentication. Only HEC ingestion goes directly to the ingest service.

## Configuration Management

The CLI uses enterprise-grade configuration with viper:

- **YAML config file**: `~/.thawk/config.yaml` (optional)
- **Environment variables**: Override any setting
- **Command-line flags**: Final override (optional)

### Priority Order

1. Command-line flags (highest priority)
2. Environment variables
3. Config file
4. Built-in defaults (lowest priority)

## Default URLs

All services route through the web backend:

| Service   | Default URL               | Notes                          |
|-----------|---------------------------|--------------------------------|
| Auth      | `http://localhost:3000`   | Login, tokens, user management |
| Query     | `http://localhost:3000`   | Search, saved searches         |
| Rules     | `http://localhost:3000`   | Detection rules                |
| Alerting  | `http://localhost:3000`   | Alerts, cases                  |
| Ingest    | `http://localhost:8088`   | HEC endpoint (direct)          |

## Environment Variables

```bash
# Override default URLs
export THAWK_AUTH_URL=http://localhost:3000
export THAWK_QUERY_URL=http://localhost:3000
export THAWK_RULES_URL=http://localhost:3000
export THAWK_ALERTING_URL=http://localhost:3000
export THAWK_INGEST_URL=http://localhost:8088

# Now all commands use these URLs by default
thawk auth login -u admin -p password
thawk search "severity:high" --last 1h
```

## Using the Wrapper Script (Recommended)

The `./scripts/thawk` wrapper automatically handles building and execution:

```bash
# The wrapper sets environment variables for Docker network access
./scripts/thawk auth login -u admin -p admin123
./scripts/thawk rules list
./scripts/thawk alerts list
./scripts/thawk search "class_uid:3002" --last 24h

# Optional: Create an alias for convenience
alias thawk='./scripts/thawk'
thawk auth whoami
```

**How it works:** The wrapper runs thawk via the devtools container with these environment variables:
- `THAWK_AUTH_URL=http://web:3000`
- `THAWK_QUERY_URL=http://web:3000`
- `THAWK_RULES_URL=http://web:3000`
- `THAWK_ALERTING_URL=http://web:3000`
- `THAWK_INGEST_URL=http://ingest:8088`

## Config File (Optional)

Create `~/.thawk/config.yaml`:

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
    access_token: ""
    refresh_token: ""

  production:
    auth_url: https://telhawk.example.com
    access_token: "your-token"
    refresh_token: "your-refresh-token"
```

Use profiles:

```bash
# Use default profile
thawk auth login -u admin -p password

# Use production profile
thawk --profile production auth whoami
```

## Command-Line Flags

Override URLs for individual commands:

```bash
# Override URL for search command
thawk search "index=*" --url http://custom:3000

# Override auth URL for login
thawk auth login --auth-url http://custom:3000 -u admin -p password
```

## Search Command Options

The search command supports both SPL-style queries and raw JSON:

```bash
# SPL-style query (default)
thawk search "severity:high" --last 1h
thawk search "class_uid:3002 AND status_id:2" --earliest "-7d" --latest "now"

# Raw JSON query from stdin
echo '{"filter":{"class_uid":3002}}' | thawk search --raw

# Raw JSON query from file
thawk search --raw < query.json
cat query.json | thawk search --raw

# Output formats
thawk search "error" --output json
thawk search "error" --output yaml
```

### Search Flags

| Flag         | Description                                      |
|--------------|--------------------------------------------------|
| `--raw`      | Read raw JSON query from stdin                   |
| `--last`     | Time range shorthand (e.g., `1h`, `24h`, `7d`)   |
| `--earliest` | Earliest time (e.g., `-1h`, `-7d`, RFC3339)      |
| `--latest`   | Latest time (e.g., `now`, `-1h`, RFC3339)        |
| `--url`      | Web backend URL (default: `http://localhost:3000`) |
| `--output`   | Output format: `table`, `json`, `yaml`           |

## Examples

### Development (Local)

```bash
# Services running locally via docker-compose
./scripts/thawk auth login -u admin -p admin123
./scripts/thawk search "severity:high" --last 1h
```

### Direct CLI (outside Docker)

```bash
# Set environment or use defaults
thawk auth login -u admin -p password
thawk search "index=security" --last 24h
```

### Production

```bash
# Use environment variables
export THAWK_AUTH_URL=https://telhawk.prod.example.com

thawk auth login -u admin -p prodpassword
thawk search "severity:critical" --last 1h --output json
```

### CI/CD

```bash
#!/bin/bash
# CI pipeline script

export THAWK_AUTH_URL=$CI_AUTH_URL

# Login
thawk auth login -u $CI_USERNAME -p $CI_PASSWORD

# Run search and export results
thawk search "class_uid:3002" --last 24h --output json > auth_events.json
```

## Configuration Best Practices

1. **Development**: Use `./scripts/thawk` wrapper or default localhost URLs
2. **Docker**: Let the wrapper script set environment variables
3. **Production**: Use environment variables from secrets management
4. **CI/CD**: Always use environment variables, never commit credentials

## Troubleshooting

### "connection refused"
- Check if services are running: `docker-compose ps`
- Verify URLs are correct
- Use `./scripts/thawk` for Docker network access

### "404 page not found"
- Ensure you're hitting the web backend (`localhost:3000`), not direct services
- Check that the web service is running

### "not logged in"
- Run `thawk auth login` first
- Check `~/.thawk/config.yaml` for saved credentials

### Environment variables not working
- Check variable names: `THAWK_AUTH_URL` (not `AUTH_URL`)
- Use `export` to set variables in your shell
- Verify with `echo $THAWK_AUTH_URL`
