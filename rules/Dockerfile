# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
COPY common/ ./common/
RUN go mod download

# Copy source code
COPY rules/ ./rules/

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rules ./rules/cmd/rules

# Runtime stage
FROM gcr.io/distroless/static-debian12:debug-nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/rules .

# Copy migrations
COPY --from=builder /build/rules/migrations ./migrations

# Copy default config file
COPY --from=builder /build/rules/config.yaml /etc/telhawk/rules/config.yaml

# Use nonroot user
USER nonroot:nonroot

EXPOSE 8084

ENTRYPOINT ["/app/rules", "-config", "/etc/telhawk/rules/config.yaml"]
