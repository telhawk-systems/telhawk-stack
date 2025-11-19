# TelHawk CLI Configuration Guide

## Configuration Management

The `thawk` CLI now uses enterprise-grade configuration with viper:

- **YAML config file**: `~/.thawk/config.yaml` (optional)
- **Environment variables**: Override any setting
- **Command-line flags**: Final override (optional)

### Priority Order

1. Command-line flags (highest priority)
2. Environment variables
3. Config file
4. Built-in defaults (lowest priority)

## Environment Variables

```bash
# Set default service URLs
export THAWK_AUTH_URL=http://localhost:8080
export THAWK_INGEST_URL=http://localhost:8088

# Now all commands use these URLs by default
thawk auth login -u admin -p password
thawk ingest send -m "test event" -t <token>
```

## Using thawk Wrapper Script

The `./scripts/thawk` wrapper automatically handles building and execution:

```bash
# The wrapper sets environment variables for internal services
./scripts/thawk auth login -u admin -p password
./scripts/thawk rules list
./scripts/thawk alerts list

# Optional: Create an alias for convenience
alias thawk='./scripts/thawk'
thawk auth whoami
```

**How it works:** The wrapper runs thawk via the devtools container with these environment variables:
- `THAWK_AUTH_URL=http://auth:8080`
- `THAWK_INGEST_URL=http://ingest:8088`
- `THAWK_QUERY_URL=http://query:8082`
- `THAWK_RULES_URL=http://rules:8084`
- `THAWK_ALERTING_URL=http://alerting:8085`

## Config File (Optional)

Create `~/.thawk/config.yaml`:

```yaml
current_profile: default

defaults:
  auth_url: http://localhost:8080
  ingest_url: http://localhost:8088

profiles:
  default:
    auth_url: http://localhost:8080
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

## Command-Line Flags (Still Supported)

You can still override with flags:

```bash
# Override auth URL for one command
thawk auth login --auth-url http://custom:8080 -u admin -p password

# Override ingest URL
thawk ingest send --ingest-url http://custom:8088 -m "test" -t <token>
```

## Examples

### Development (Local)

```bash
# Services running locally
thawk auth login -u admin -p devpass
thawk ingest send -m "dev event" -t <token>
```

### Docker Compose

```bash
# Environment variables set automatically
./scripts/thawk auth login -u admin -p password
```

### Production

```bash
# Use environment variables
export THAWK_AUTH_URL=https://telhawk-auth.prod.example.com
export THAWK_INGEST_URL=https://telhawk-ingest.prod.example.com

thawk auth login -u admin -p prodpassword
```

### CI/CD

```bash
#!/bin/bash
# CI pipeline script

export THAWK_AUTH_URL=$CI_AUTH_URL
export THAWK_INGEST_URL=$CI_INGEST_URL

# Login
thawk auth login -u $CI_USERNAME -p $CI_PASSWORD

# Send test event
thawk ingest send -m "CI test event" -t $HEC_TOKEN
```

## Configuration Best Practices

1. **Development**: Use default localhost URLs or config file
2. **Docker**: Let docker-compose set environment variables
3. **Production**: Use environment variables from secrets management
4. **CI/CD**: Always use environment variables, never commit credentials

## Troubleshooting

### "connection refused"
- Check if services are running
- Verify URLs are correct
- Check if environment variables are set: `echo $THAWK_AUTH_URL`

### "config file not found" 
- This is OK! CLI works without config file
- Config file is optional, environment variables work standalone

### Environment variables not working
- Check variable names: `THAWK_AUTH_URL` (not `AUTH_URL`)
- Use `export` to set variables in your shell
- In docker-compose, variables are automatically set
