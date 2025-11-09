FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o core ./cmd/core

FROM alpine:latest

RUN apk --no-cache add ca-certificates busybox-extras

WORKDIR /app

COPY --from=builder /build/core .

EXPOSE 8090

CMD ["./core"]
