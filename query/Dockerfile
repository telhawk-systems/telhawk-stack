FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
COPY common/ ./common/
RUN go mod download

COPY query/ ./query/
RUN go build -o query ./query/cmd/query

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/query .
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/query/config.yaml /etc/telhawk/query/config.yaml

EXPOSE 8082

CMD ["/app/query", "-config", "/etc/telhawk/query/config.yaml"]
