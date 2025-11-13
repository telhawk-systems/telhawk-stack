# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o alerting ./cmd/alerting

# Runtime stage
FROM gcr.io/distroless/static-debian12:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/alerting .

# Copy config file
COPY config.yaml /etc/telhawk/alerting/config.yaml

# Copy builtin detection rules
COPY rules /etc/telhawk/alerting/rules

# Copy migrations
COPY migrations /app/migrations

# Expose port
EXPOSE 8085

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/alerting", "healthz"] || exit 1

# Run the service
ENTRYPOINT ["/app/alerting"]
CMD ["-config", "/etc/telhawk/alerting/config.yaml"]
