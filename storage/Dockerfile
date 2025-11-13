FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
COPY common/ ./common/
RUN go mod download

COPY storage/ ./storage/
RUN CGO_ENABLED=0 GOOS=linux go build -o storage ./storage/cmd/storage

FROM alpine:latest

RUN apk --no-cache add ca-certificates busybox-extras

WORKDIR /app

COPY --from=builder /build/storage .
COPY --from=builder /build/storage/config.yaml .

EXPOSE 8083

CMD ["./storage", "-config", "config.yaml"]
