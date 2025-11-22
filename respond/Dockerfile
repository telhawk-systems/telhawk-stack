# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build/respond
# Copy root go.mod
COPY go.mod go.sum ../

# Copy common module
COPY common/ ../common/

# Install build dependencies
RUN apk add --no-cache git

# Copy search module (dependency for correlation) to parent directory
COPY search/ ../search/

# Copy go mod files
# Service files copied below
WORKDIR /build
RUN go mod download
WORKDIR /build/respond

# Copy source code
COPY respond/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o respond ./cmd/respond

# Runtime stage
FROM gcr.io/distroless/static-debian12:debug-nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/respond/respond .

# Copy config file
COPY --from=builder /build/respond/config.yaml /etc/telhawk/respond/config.yaml

# Copy builtin detection rules (from alerting)
COPY --from=builder /build/respond/rules /etc/telhawk/respond/rules

# Copy migrations
COPY --from=builder /build/respond/migrations /app/migrations

# Use nonroot user
USER nonroot:nonroot

# Expose port
EXPOSE 8086

# Run the service
ENTRYPOINT ["/app/respond"]
CMD ["-config", "/etc/telhawk/respond/config.yaml"]
