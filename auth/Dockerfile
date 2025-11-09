# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth ./cmd/auth

# Runtime stage - use debug image for shell access
FROM gcr.io/distroless/static-debian12:debug-nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/auth .

# Copy migrations
COPY --from=builder /build/migrations ./migrations

# Copy default config file
COPY --from=builder /build/config.yaml /etc/telhawk/auth/config.yaml

# Use nonroot user
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/auth", "-config", "/etc/telhawk/auth/config.yaml"]
