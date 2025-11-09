# Stage 1: Build React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /build/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web ./cmd/web

# Stage 3: Production - distroless
FROM gcr.io/distroless/static-debian12:debug-nonroot

WORKDIR /app

COPY --from=backend-builder /build/web .
COPY --from=frontend-builder /build/frontend/dist ./static
COPY --from=backend-builder /build/config.yaml /etc/telhawk/web/config.yaml

USER nonroot:nonroot

EXPOSE 3000

ENTRYPOINT ["/app/web", "-config", "/etc/telhawk/web/config.yaml"]
