# TelHawk Stack - Common Troubleshooting Guide

## Service Startup Issues

### Service Won't Start

#### Check Logs
```bash
# View logs for specific service
docker-compose logs -f <service-name>

# View last 100 lines
docker-compose logs --tail=100 <service-name>

# Check all services
docker-compose logs
```

#### Check Service Status
```bash
# See which services are running
docker-compose ps

# Check health status
docker inspect --format='{{json .State.Health}}' telhawk-stack-<service>-1 | jq
```

#### Common Causes

**Port Already in Use**
```bash
# Check what's using the port
sudo lsof -i :8080
sudo netstat -tulpn | grep 8080

# Kill the process
sudo kill -9 <PID>

# Or change port in docker-compose.yml
```

**Missing Dependencies**
```bash
# Service depends on another service
# Check if dependency is healthy
docker-compose ps

# Restart with dependencies
docker-compose up -d --force-recreate <service-name>
```

**Configuration Errors**
```bash
# Check config file syntax
cat <service>/config.yaml | yq eval

# Verify environment variables
docker-compose config | grep -A 10 <service-name>
```

## Database Connection Issues

### PostgreSQL Connection Failed

```bash
# Check if PostgreSQL is running
docker-compose ps auth-db

# Check PostgreSQL logs
docker-compose logs -f auth-db

# Test connection
docker-compose exec auth-db psql -U telhawk -d telhawk_auth -c "SELECT 1;"

# Check connection string
# Should be: postgres://telhawk:password@auth-db:5432/telhawk_auth?sslmode=disable
```

#### Common Causes

**Wrong Credentials**
- Check `AUTH_DB_PASSWORD` environment variable
- Verify password in docker-compose.yml

**Database Not Ready**
- PostgreSQL may not be fully initialized
- Wait 10-30 seconds and retry
- Check health status: `docker-compose ps auth-db`

**Network Issues**
- Services must be on same Docker network
- Check networks: `docker network ls`
- Verify service names match (auth-db, not localhost)

### OpenSearch Connection Failed

```bash
# Check if OpenSearch is running
docker-compose ps opensearch

# Check OpenSearch logs
docker-compose logs -f opensearch

# Test connection (from host)
curl -k -u admin:TelHawk123! https://localhost:9200

# Test connection (from container)
docker-compose exec storage curl -k https://opensearch:9200
```

#### Common Causes

**TLS Certificate Issues**
```bash
# Check certificates exist
docker volume inspect telhawk-stack_opensearch-certs

# Regenerate certificates
docker volume rm telhawk-stack_opensearch-certs
docker-compose up -d
```

**Wrong Credentials**
- Default: `admin:TelHawk123!`
- Check `STORAGE_OPENSEARCH_PASSWORD` and `QUERY_OPENSEARCH_PASSWORD`

**Not Enough Memory**
```bash
# OpenSearch requires significant memory
# Check Docker resources
docker stats

# Increase vm.max_map_count (Linux)
sudo sysctl -w vm.max_map_count=262144
```

### Redis Connection Issues

```bash
# Check Redis status
docker-compose ps redis

# Test connection
docker-compose exec redis redis-cli ping
# Should return: PONG

# Check if rate limiting works without Redis
# Services should degrade gracefully
```

## Build Issues

### Go Build Failures

```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify

# Check for syntax errors
go build ./...
```

#### Common Errors

**Missing Dependencies**
```bash
# Error: cannot find package
go get <package-name>
go mod tidy
```

**Version Conflicts**
```bash
# Check go.mod for conflicts
cat go.mod

# Update specific dependency
go get -u <package-name>@latest
```

**GOPROXY Issues**
```bash
# Set proxy
export GOPROXY=https://proxy.golang.org,direct

# Or use direct
export GOPROXY=direct
```

### Docker Build Failures

```bash
# Clear Docker build cache
docker builder prune

# Build without cache
docker-compose build --no-cache <service-name>

# Check for disk space
docker system df

# Clean up old images
docker system prune -a
```

#### Common Errors

**No Space Left on Device**
```bash
# Check disk usage
df -h

# Clean Docker
docker system prune -a --volumes
```

**Network Timeout During Build**
```bash
# Increase timeout in Dockerfile
# Or check network connectivity
ping github.com
```

## Runtime Issues

### Events Not Being Ingested

#### Check Ingest Service
```bash
# Test HEC endpoint
curl -X POST http://localhost:8088/services/collector/event \
  -H "Authorization: Splunk <HEC-TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"event": "test event", "sourcetype": "generic_event"}'

# Check ingest logs
docker-compose logs -f ingest

# Verify HEC token
docker-compose run --rm thawk token list
```

#### Check Core Service
```bash
# Check core logs for normalization errors
docker-compose logs -f core

# Check Dead Letter Queue
curl http://localhost:8090/dlq/list

# View DLQ files
docker-compose exec core ls -la /var/lib/telhawk/dlq/
```

#### Check Storage Service
```bash
# Check storage logs
docker-compose logs -f storage

# Verify OpenSearch connection
curl -k -u admin:TelHawk123! https://localhost:9200/_cat/indices?v

# Check for telhawk-events indices
curl -k -u admin:TelHawk123! https://localhost:9200/_cat/indices/telhawk-events-*
```

### Events Not Showing in Search

```bash
# Check if indices exist
curl -k -u admin:TelHawk123! https://localhost:9200/_cat/indices/telhawk-events-*

# Check index count
curl -k -u admin:TelHawk123! https://localhost:9200/telhawk-events-*/_count

# Search directly in OpenSearch
curl -k -u admin:TelHawk123! -X GET "https://localhost:9200/telhawk-events-*/_search?pretty" \
  -H "Content-Type: application/json" \
  -d '{"query": {"match_all": {}}, "size": 10}'

# Check query service logs
docker-compose logs -f query
```

### Authentication Failures

```bash
# Test login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# Check auth service logs
docker-compose logs -f auth

# Check database
docker-compose exec auth-db psql -U telhawk -d telhawk_auth -c "SELECT * FROM users;"

# Check JWT secret is set
docker-compose exec auth env | grep JWT_SECRET
```

#### Common Causes

**Default User Not Created**
```bash
# Check if migration ran
docker-compose exec auth-db psql -U telhawk -d telhawk_auth -c "SELECT * FROM users;"

# Re-run migrations
docker-compose restart auth
```

**JWT Secret Not Set**
```bash
# Set in docker-compose.yml or .env
AUTH_JWT_SECRET=your-secret-key

# Restart auth service
docker-compose restart auth
```

**Token Expired**
```bash
# Login again to get fresh token
# Or adjust token expiration in auth config
```

### Rate Limiting Issues

```bash
# Check Redis connection
docker-compose exec redis redis-cli ping

# Check rate limit config
docker-compose exec ingest env | grep RATE_LIMIT

# View rate limit metrics
curl http://localhost:8088/metrics | grep rate_limit

# Temporarily disable rate limiting (development only)
# Set INGEST_RATE_LIMIT_IP_ENABLED=false
```

## Performance Issues

### Slow Event Ingestion

```bash
# Check service health
docker-compose ps

# Check resource usage
docker stats

# Check OpenSearch performance
curl -k -u admin:TelHawk123! https://localhost:9200/_cluster/health?pretty

# Check bulk queue
curl http://localhost:8083/metrics | grep queue
```

#### Common Causes

**OpenSearch Overwhelmed**
- Increase heap size in docker-compose.yml
- Add more OpenSearch nodes
- Adjust bulk size in storage config

**CPU/Memory Constraints**
```bash
# Check Docker resources
docker stats

# Increase resources in Docker Desktop settings
# Or adjust service limits in docker-compose.yml
```

### Slow Queries

```bash
# Check OpenSearch cluster health
curl -k -u admin:TelHawk123! https://localhost:9200/_cluster/health?pretty

# Check index stats
curl -k -u admin:TelHawk123! https://localhost:9200/_cat/indices/telhawk-events-*?v&s=docs.count:desc

# Check query performance
curl -k -u admin:TelHawk123! -X GET "https://localhost:9200/telhawk-events-*/_search?pretty&explain=true"
```

#### Optimization

**Add Indices**
- Ensure proper field mappings
- Add indices on frequently queried fields

**Limit Time Range**
- Query specific date indices
- Use time-based filtering

**Use Aggregations Wisely**
- Limit aggregation cardinality
- Use sampling for large datasets

## Network Issues

### Services Can't Communicate

```bash
# Check Docker networks
docker network ls

# Inspect network
docker network inspect telhawk-stack_default

# Check if services are on same network
docker inspect <container-id> | grep NetworkMode

# Test connectivity between services
docker-compose exec core ping auth
docker-compose exec storage curl http://core:8090/healthz
```

#### Common Causes

**Wrong Service Name**
- Use service name from docker-compose.yml
- Not `localhost`, use `auth`, `core`, `storage`, etc.

**Port Not Exposed Internally**
- Check docker-compose.yml for port mappings
- Internal services use container ports, not host ports

## Certificate/TLS Issues

### TLS Handshake Failures

```bash
# Check if certificates exist
docker volume ls | grep certs
docker volume inspect telhawk-stack_telhawk-certs

# Regenerate certificates
docker volume rm telhawk-stack_telhawk-certs
docker volume rm telhawk-stack_opensearch-certs
docker-compose up -d

# For development, skip verification
export AUTH_TLS_SKIP_VERIFY=true
# Set for all services in docker-compose.yml
```

### Certificate Verification Failed

```bash
# Check certificate validity
docker-compose exec auth openssl x509 -in /certs/server.crt -text -noout

# Check certificate expiration
docker-compose exec auth openssl x509 -in /certs/server.crt -noout -dates

# In development, use TLS_SKIP_VERIFY=true
# In production, ensure proper certificates
```

## Data Issues

### Data Loss After Restart

```bash
# Check if volumes are being removed
docker-compose down -v  # This DELETES data!

# Should use (keeps data):
docker-compose down

# Check volume mounts
docker volume ls
docker volume inspect telhawk-stack_opensearch-data
docker volume inspect telhawk-stack_auth-db-data
```

### Disk Full

```bash
# Check disk usage
df -h

# Check Docker disk usage
docker system df

# Clean up
docker system prune -a
docker volume prune

# Remove old indices
curl -k -u admin:TelHawk123! -X DELETE https://localhost:9200/telhawk-events-2024.01.*
```

## Migration Issues

### Migrations Not Running

```bash
# Check migration status
docker-compose exec auth-db psql -U telhawk -d telhawk_auth -c "\dt"

# Check auth service logs for migration errors
docker-compose logs auth | grep migration

# Manually run migrations
cd auth
migrate -database "postgres://telhawk:password@localhost:5432/telhawk_auth?sslmode=disable" -path migrations up
```

### Migration Failed

```bash
# Check which version
migrate -database "postgres://..." -path migrations version

# Rollback
migrate -database "postgres://..." -path migrations down 1

# Fix the migration file
# Then re-apply
migrate -database "postgres://..." -path migrations up
```

## Frontend Issues

### Web UI Not Loading

```bash
# Check if web service is running
docker-compose ps web

# Check logs
docker-compose logs -f web

# Test direct access
curl http://localhost:3000

# Check if static files exist
docker-compose exec web ls -la /app/static/
```

### API Calls Failing (CORS)

```bash
# Check browser console for CORS errors
# Add CORS headers to web backend
# Or use proxy configuration in frontend
```

### Frontend Not Updating

```bash
# Clear browser cache
# Hard reload: Ctrl+Shift+R (Linux/Windows) or Cmd+Shift+R (Mac)

# Rebuild frontend
cd web/frontend
npm run build

# Rebuild Docker image
docker-compose build web
docker-compose up -d web
```

## General Debugging Commands

### View All Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f <service-name>

# With timestamps
docker-compose logs -f --timestamps

# Filter by string
docker-compose logs -f | grep ERROR
```

### Inspect Service
```bash
# View service configuration
docker-compose config

# View container details
docker inspect <container-name>

# View environment variables
docker-compose exec <service-name> env

# Execute command in container
docker-compose exec <service-name> sh
```

### Network Debugging
```bash
# Test connectivity
docker-compose exec <service> ping <other-service>

# Test HTTP endpoint
docker-compose exec <service> curl http://<other-service>:<port>/healthz

# View network details
docker network inspect telhawk-stack_default
```

### Complete Reset

If all else fails, complete reset:

```bash
# Stop everything
docker-compose down -v

# Remove volumes (DELETES ALL DATA)
docker volume rm $(docker volume ls -q | grep telhawk-stack)

# Remove images
docker rmi $(docker images | grep telhawk | awk '{print $3}')

# Rebuild and restart
docker-compose build --no-cache
docker-compose up -d

# Check logs
docker-compose logs -f
```

## Getting Help

### Check Documentation
- `README.md`: Project overview
- `DOCKER.md`: Docker-specific guide
- `docs/CONFIGURATION.md`: Configuration reference
- `docs/TLS_CONFIGURATION.md`: TLS setup
- Service-specific READMEs in service directories

### Debug Mode
Enable verbose logging:
```bash
# Set log level to debug
<SERVICE>_LOG_LEVEL=debug

# Example
AUTH_LOG_LEVEL=debug docker-compose up -d auth
```

### Health Checks
```bash
# All services expose health endpoints
curl http://localhost:8080/healthz  # auth
curl http://localhost:8088/healthz  # ingest
curl http://localhost:8090/readyz   # core
curl http://localhost:8083/healthz  # storage
curl http://localhost:8082/healthz  # query
curl http://localhost:3000/healthz  # web
```