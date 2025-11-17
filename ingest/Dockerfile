# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
COPY common/ ./common/
RUN go mod download

# Copy source code
COPY ingest/ ./ingest/

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ingest ./ingest/cmd/ingest

# Runtime stage - use debug image for shell access
FROM gcr.io/distroless/static-debian12:debug-nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ingest .

# Copy default config file
COPY --from=builder /build/ingest/config.yaml /etc/telhawk/ingest/config.yaml

# Use nonroot user
USER nonroot:nonroot

EXPOSE 8088

ENTRYPOINT ["/app/ingest", "-config", "/etc/telhawk/ingest/config.yaml"]
