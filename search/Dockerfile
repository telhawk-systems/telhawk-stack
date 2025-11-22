FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
COPY common/ ./common/
RUN go mod download

COPY search/ ./search/
RUN go build -o search ./search/cmd/search

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/search .
COPY --from=builder /build/search/migrations ./migrations
COPY --from=builder /build/search/config.yaml /etc/telhawk/search/config.yaml

EXPOSE 8082

CMD ["/app/search", "-config", "/etc/telhawk/search/config.yaml"]
