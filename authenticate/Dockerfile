# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
COPY common/ ./common/
RUN go mod download

# Copy source code
COPY authenticate/ ./authenticate/

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o authenticate ./authenticate/cmd/authenticate

# Runtime stage - use debug image for shell access
FROM gcr.io/distroless/static-debian12:debug-nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/authenticate .

# Copy migrations
COPY --from=builder /build/authenticate/migrations ./migrations

# Copy default config file
COPY --from=builder /build/authenticate/config.yaml /etc/telhawk/authenticate/config.yaml

# Use nonroot user
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/authenticate", "-config", "/etc/telhawk/authenticate/config.yaml"]
