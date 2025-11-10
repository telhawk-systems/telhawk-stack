# Local Development

Purpose: Quick start for running and iterating locally.

- Requires Docker and Docker Compose.
- No local Go toolchain needed; containers build and run services.

Common tasks
- Start stack: `docker-compose up -d`
- Logs: `docker-compose logs -f`
- Status: `docker-compose ps`

CLI (`thawk`)
- Run once: `docker-compose run --rm thawk auth login -u admin -p <password>`
- Alias (optional): `alias thawk='docker-compose run --rm thawk'`

Notes
- Health/ready endpoints and service specifics live in service docs.
- Configuration: see `docs/CONFIGURATION.md`.
- Metrics: see `docs/PROMETHEUS_METRICS.md`.
