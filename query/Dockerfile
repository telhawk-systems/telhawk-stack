FROM golang:1.24.2-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o query ./cmd/query

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/query .
COPY config.yaml /etc/telhawk/query/config.yaml

EXPOSE 8082

CMD ["/app/query", "-config", "/etc/telhawk/query/config.yaml"]
