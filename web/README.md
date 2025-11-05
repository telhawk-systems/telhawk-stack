# TelHawk Web UI

React-based web interface for TelHawk with Go backend for authentication and API proxying.

## Architecture

- **Backend (Go)**: Authentication, session management, API proxy to query/core services
- **Frontend (React + TypeScript)**: Single-page application with Vite

## Development

### Backend
```bash
cd backend
go mod download
go run ./cmd/web
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

Frontend dev server runs on port 5173 and proxies API calls to backend on port 3000.

## Production Build

```bash
docker build -t telhawk/web:latest .
```

The Dockerfile:
1. Builds React frontend (Vite)
2. Builds Go backend
3. Combines into distroless image serving static files + API

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_PORT` | 3000 | Web server port |
| `STATIC_DIR` | ./static | Path to React build files |
| `AUTH_SERVICE_URL` | http://auth:8080 | Auth service endpoint |
| `QUERY_SERVICE_URL` | http://query:8082 | Query service endpoint |
| `CORE_SERVICE_URL` | http://core:8090 | Core service endpoint |
| `COOKIE_DOMAIN` | "" | Cookie domain (empty for default) |
| `COOKIE_SECURE` | true | Use secure cookies (HTTPS only) |
| `DEV_MODE` | false | Enable dev mode (CORS, debug logging) |

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login with username/password
- `POST /api/auth/logout` - Logout and revoke tokens
- `GET /api/auth/me` - Get current user info

### Proxy Endpoints
- `/api/query/*` - Proxied to query service (protected)
- `/api/core/*` - Proxied to core service (protected)

All proxied requests include `X-User-ID` header with authenticated user ID.

## Features

- ✅ JWT-based authentication with refresh tokens
- ✅ HTTP-only secure cookies
- ✅ Automatic token refresh
- ✅ Protected routes
- ✅ API proxy with user context
- ✅ SPA routing support
- ✅ Distroless Docker image

## Security

- Tokens stored in HTTP-only cookies (not accessible to JavaScript)
- SameSite=Strict cookie policy
- Automatic token refresh on expiry
- HTTPS enforced in production (COOKIE_SECURE=true)
- Distroless base image for minimal attack surface
- Runs as non-root user

## TODO

- [ ] Add search console UI components
- [ ] Add dashboard visualization
- [ ] Implement role-based UI rendering
- [ ] Add session timeout warnings
- [ ] Improve error handling and user feedback
