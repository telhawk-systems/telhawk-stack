# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy entire project
COPY . .

# Build CLI binary
WORKDIR /build/cli
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o thawk .

# Create .thawk directory with proper ownership for nonroot user (UID 65532)
RUN mkdir -p /tmp/.thawk && chown -R 65532:65532 /tmp/.thawk

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/cli/thawk .

# Copy the pre-created .thawk directory with proper ownership
COPY --from=builder --chown=nonroot:nonroot /tmp/.thawk /app/.thawk

USER nonroot:nonroot

# Set HOME for nonroot user to /app where they have write access
ENV HOME=/app

ENTRYPOINT ["/app/thawk"]
