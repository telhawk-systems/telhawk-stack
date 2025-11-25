# Web Development Quick Start

## Structure

```
web/
├── backend/          # Go HTTP server
│   ├── cmd/web/      # Main entry point
│   ├── internal/     # Auth, handlers, proxy
│   └── config.yaml   # Service configuration
├── frontend/         # React + TypeScript
│   ├── src/          # React source code
│   └── package.json  # NPM dependencies
├── Dockerfile        # Multi-stage build (distroless)
└── README.md         # Full documentation
```

## Development Workflow

### Option 1: Full Stack with Docker

```bash
# Build and start all services including web
docker compose up --build web

# Access UI at http://localhost:3000
# Login with default credentials (see auth service docs)
```

### Option 2: Local Development (Hot Reload)

**Terminal 1: Start Backend**
```bash
cd web/backend
export AUTH_SERVICE_URL=http://localhost:8080
export QUERY_SERVICE_URL=http://localhost:8082
export CORE_SERVICE_URL=http://localhost:8090
export DEV_MODE=true
export COOKIE_SECURE=false
go run ./cmd/web
# Backend runs on :3000
```

**Terminal 2: Start Frontend**
```bash
cd web/frontend
npm install
npm run dev
# Frontend dev server runs on :5173 with hot reload
# Proxies API calls to backend on :3000
```

**Terminal 3: Start Dependencies**
```bash
# Start auth, query, core services
docker compose up auth query core storage opensearch
```

## Key Features

✅ **Authentication**
- Login/logout with JWT tokens
- HTTP-only cookies (XSS protection)
- Automatic token refresh
- Protected routes

✅ **API Proxy**
- `/api/query/*` → Query service
- `/api/core/*` → Core service
- Adds `X-User-ID` header to proxied requests

✅ **React App**
- TypeScript with strict mode
- Vite for fast dev/build
- React Router for SPA routing
- Auth context with hooks

## Making Changes

### Adding a New Page

1. Create component in `frontend/src/pages/NewPage.tsx`
2. Add route in `frontend/src/App.tsx`
3. Wrap in `<ProtectedRoute>` if auth required

### Adding a New API Endpoint

1. Add handler in `backend/internal/handlers/`
2. Register route in `backend/cmd/web/main.go`
3. Use `authMiddleware.Protect()` for protected endpoints

### Adding a Proxy Route

```go
// In main.go
someProxy := proxy.NewProxy(cfg.SomeServiceURL, authClient)
mux.Handle("/api/some/", authMiddleware.Protect(
    http.StripPrefix("/api/some", someProxy.Handler()),
))
```

## Testing

### Backend
```bash
cd backend
go test ./...
```

### Frontend
```bash
cd frontend
npm run lint
npm run build  # Test production build
```

### Full Stack
```bash
# Build and test Docker image
docker build -t telhawk/web:test .
docker run -p 3000:3000 \
  -e AUTH_SERVICE_URL=http://host.docker.internal:8080 \
  -e QUERY_SERVICE_URL=http://host.docker.internal:8082 \
  telhawk/web:test
```

## Troubleshooting

**CORS errors in dev mode:**
- Ensure backend has `DEV_MODE=true`
- Check Vite proxy config in `vite.config.ts`

**Cookie not set:**
- Check `COOKIE_SECURE=false` for local dev (HTTP)
- Verify auth service is running and accessible

**API proxy fails:**
- Verify service URLs in backend config
- Check service health: `curl http://localhost:8082/api/v1/health`

## Production Considerations

- Set `COOKIE_SECURE=true` (HTTPS only)
- Set `DEV_MODE=false` (disables CORS, verbose logging)
- Use proper `COOKIE_DOMAIN` for multi-subdomain setups
- Frontend is served from `/app/static` in container
- Binary runs as non-root user in distroless image
