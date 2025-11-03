FROM golang:1.24.2-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o storage ./cmd/storage

FROM alpine:latest

RUN apk --no-cache add ca-certificates busybox-extras

WORKDIR /app

COPY --from=builder /build/storage .
COPY config.yaml .

EXPOSE 8083

CMD ["./storage", "-config", "config.yaml"]
